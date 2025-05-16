package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/abakum/go-taglib"
	"github.com/tetratelabs/wazero"
	"golang.org/x/text/encoding/unicode"
)

func main() {
	log.SetFlags(log.Lshortfile)
	ctx, cncl := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cncl()

	// a/b.c
	name := filepath.Base(os.Args[0])
	// b.c
	name = strings.TrimSuffix(name, filepath.Ext(name))
	// b
	var (
		rc   uint32
		err  error
		file string
	)
	log.Println(name)
	switch strings.ToLower(name) {
	case "ffmpeg", "ffprobe":
		root := ""
		ok := false
		inp := ""
		for _, arg := range os.Args[1:] {
			if ok {
				inp = arg
				log.Println("Все файлы для", name, "должны быть на том же диске что и infile", inp)
				break
			}
			ok = arg == "-i"
		}
		// c.d
		file, err = filepath.Abs(inp)
		// a:\b\c.d
		if err == nil {
			if inp == "" {
				root = file // cwd
			} else {
				root = filepath.Dir(file)
			}
			// a:\b
		}
		log.Println("Каталог с infile", root)
		if root != "" {
			ok = false
			for i, arg := range os.Args[1:] {
				if ok {
					inp, err = filepath.Rel(root, arg)
					if err == nil {
						log.Println("Имя infile", inp)
						os.Args[i+1] = inp
					}
					break
				}
				ok = arg == "-i"
			}
		} else {
			root = "."
		}
		rc, err = run(ctx, strings.ToLower(name), root, os.Args[1:]...)
		if err != nil {
			panic(err)
		}

		os.Exit(int(rc))
	}
	if len(os.Args) < 2 {
		fmt.Print(`Например: 20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор
Вот классические тэги если несколько то через /:
Date=Дата записи как 20250227
Album=Запись как 20250227 Классный концерт
TrackNumber=Номер произведения как 02
Composer=Композитор как Фредерик Шопен
Title=Название произведение как Шопен Баллада для фортепиано № 1 соль минор
MovementNumber=Если части произведения то их номера
Movement=Если части произведения то их названия
Artist=Исполнитель как Юлия Абакумова
AlbumArtist=Остальные исполнители кроме солиста
Conductor=Руководитель солиста как Владимир Дайч или оркестра или концертмейстер
Comment=Комментарий
Genre=Classical
InitialKey=Тональность как Gm. До-диез мажор как C#, Ре-бемоль мажор как Db
InvolvedPeople=Остальные причастные к записи как РГК им С. В. Рахманинова
Lyricist=Авторы текста и переводчики
Arranger=Авторы переложения или оранжировки
Subtitle=Подзаголовок как Патетическая соната
Work=Авторские публикации или каталоги как BWV или opus posthumum как Op. 21
Grouping=Группировки, например для музыкальных форм как Баллады для фортепиано
Остальные тэги https://taglib.org/api/p_propertymapping.html
Расширенно по mp3 https://id3.org/id3v2.3.0`)
		return
	}
	args1, err := filepath.Abs(os.Args[1])
	if err != nil {
		args1 = os.Args[1]
	}
	ext := filepath.Ext(args1)
	if strings.ToLower(ext) == ".csv" {
		out := filepath.Dir(args1)
		out = filepath.Dir(out)
		log.Println("Результат ищем в", out)

		f, err := os.Open(args1)
		if err != nil {
			log.Fatalln("Ошибка открытия", err)
			return
		}
		defer f.Close()

		utf := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
		nr := csv.NewReader(utf.NewDecoder().Reader(f))
		r, err := nr.Read()
		// log.Println(r)
		if err != nil {
			log.Fatalln("Ошибка разбора заголовка", err)
			return
		}
		header := make(map[string]int)
		for j, c := range r {
			header[c] = j
		}

		// log.Println(header)
		i := 1
		for {
			i++
			r, err := nr.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Println("Ошибка разбора строки", i, r)
				continue
			}
			// log.Println(r)
			file := keyVal("File Name", header, r)
			audio := keyVal("Resolution", header, r) == ""
			image := keyVal("Duration TC", header, r) == "00:00:00:01"
			in := keyVal("Clip Directory", header, r)
			inFile := filepath.Join(in, file)
			if in != "" {
				// Файл
				if audio {
					log.Println("Аудио", inFile)
					p, err := taglib.ReadProperties(inFile)
					if err != nil {
						log.Println("bit_rate=?", err)
					} else {
						fmt.Println("bit_rate=" + strconv.FormatUint(uint64(p.Bitrate), 10))
					}

				} else if !image {
					log.Println("Видео", inFile)
					rc, err := run(ctx, "ffprobe", in,
						"-hide_banner",
						"-v", "error",
						"-select_streams", "v:0",
						"-show_entries", "stream=bit_rate",
						"-of", "default=noprint_wrappers=1", //:nokey=1
						file,
					)
					if err != nil {
						log.Println("bit_rate=?", err, "код завершения", "ffprobe", rc)
					}
					prop("Resolution", header, r)
					prop("Frame Rate", header, r)
					prop("Video Codec", header, r)
				}
				prop("Audio Bit Depth", header, r)
				prop("Audio Sample Rate", header, r)
				prop("Audio Codec", header, r)
				continue
			}
			timeLine(filepath.Join(out, file), keyVal("Description", header, r))
		}
		return
	}
	tags, err := taglib.ReadTags(os.Args[1])
	if err != nil {
		fmt.Println(os.Args, err)
		return
	}
	block := map[string]bool{
		"ENCODING":          true,
		"ENCODER":           true,
		"COMPATIBLE_BRANDS": true,
		"MINOR_VERSION":     true,
		"MAJOR_BRAND":       true,
	}
	for k, v := range tags {
		if block[k] {
			continue
		}
		fmt.Println(k + "=" + strings.Join(v, "/"))
	}
}

func isFileExist(name string) bool {
	fi, err := os.Stat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return err == nil && !fi.IsDir()
}

func isFirstAfterSecond(first, second string) bool {
	fi2, err := os.Stat(second)
	if err != nil {
		return true
	}

	if fi, err := os.Stat(first); err == nil && fi.ModTime().After(fi2.ModTime()) {
		return true
	}
	return false
}

func run(ctx context.Context, bin, root string, args ...string) (rc uint32, err error) {
	cacheDir := os.TempDir()
	if ucd, err := os.UserCacheDir(); err == nil {
		cacheDir = ucd
	}
	os.Setenv("WAZERO_COMPILATION_CACHE", cacheDir)
	rc, err = ffmpreg.Run(ctx, wasm.Args{
		Name:   bin,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Args:   args,
		Config: func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
			for _, kv := range os.Environ() {
				i := strings.IndexByte(kv, '=')
				if i < 1 {
					continue
				}
				cfg = cfg.WithEnv(kv[:i], kv[i+1:])
			}
			fscfg := wazero.NewFSConfig()

			fscfg = fscfg.WithDirMount(root, "/")
			return cfg.WithFSConfig(fscfg)
		},
	})
	return
}

func keyVal(key string, m map[string]int, s []string) (val string) {
	i, ok := m[key]
	if !ok {
		return
	}
	if len(s) > i {
		val = s[i]
	}
	return
}

func timeLine(file, description string) {
	res, err := os.Open(file + ".mov") // удерживаем
	if err != nil {
		log.Println("Нет результата", file+".mov")
		res, err = os.Open(file + ".mp4")
		if err != nil {
			log.Println("Нет результата", file+".mp4")
			return
		}
	}
	defer res.Close()
	if isFirstAfterSecond(res.Name(), file+".mp3") ||
		isFirstAfterSecond(res.Name(), file+".flac") ||
		isFirstAfterSecond(res.Name(), file+".mp4") {
		if strings.HasSuffix(res.Name(), ".mov") {
			// DR 17
			log.Println("Значит результат в mov с lpcm. Пишем mp4 с alac, mp3, flac")
			res.Close()
			log.Println("Удаляем mov")
		} else {
			// DR>17 или mov уже удалили и в mp4 уже alac
			log.Println("Значит результат в mp4 с flac или alac. Пишем mp3, flac")
		}
		if description != "" {
			log.Println("Тэггируем", description)
		}
	}
}

func prop(key string, header map[string]int, r []string) {
	val := keyVal(key, header, r)
	if val != "" {
		fmt.Println(key + "=" + keyVal(key, header, r))
	}
}
