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

var (
	ctx  context.Context
	cncl context.CancelFunc
)

func main() {
	log.SetFlags(log.Lshortfile)
	ctx, cncl = signal.NotifyContext(
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
Вот классические тэги:
Date=Дата записи как 20250227
Album=Запись как 20250227 Классный концерт
TrackNumber=Номер произведения как 02
Composer=Композитор как Фредерик Шопен
Title=Название произведение как Шопен Баллада для фортепиано № 1 соль минор
MovementNumber=Если части произведения то их номера
Movement=Если части произведения то их названия
Artist=Исполнители как Юлия Абакумова
AlbumArtist=Остальные исполнители кроме солиста
Conductor=Руководители солиста как Владимир Дайч или оркестра или концертмейстер
Comment=Комментарий
Genre=Classical
InitialKey=Тональность как Gm. До-диез мажор как C#, Ре-бемоль мажор как Db
InvolvedPeople=Остальные причастные к записи как РГК им С. В. Рахманинова
Lyricist=Авторы текста и переводчики
Arranger=Авторы переложения или оранжировки
Subtitle=Подзаголовок как Патетическая соната
Work=Авторские публикации или каталоги как BWV или opus posthumum как Op. 21
Grouping=Группировки, например для музыкальных форм как Баллады для фортепиано
Если значений правее = несколько повторяйте строчки. Например:
MovementNumber=1
Movement=Скерцо
MovementNumber=2
Movement=Адажио

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
		// a/b/c.d
		out := filepath.Dir(args1)
		// a/b
		out = filepath.Dir(out)
		// a
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
			log.Fatalln("Ошибка разбора заголовка", err, r)
			return
		}
		header := make(map[string]int)
		for j, c := range r {
			header[c] = j
		}

		log.Println(header)
		csvTags := make(map[string][]string)
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
			log.Println(r)
			file := keyVal("File Name", header, r)
			audio := keyVal("Resolution", header, r) == ""
			image := keyVal("Duration TC", header, r) == "00:00:00:01"
			in := keyVal("Clip Directory", header, r)
			inFile := filepath.Join(in, file)
			if in != "" {
				// Файл
				if audio {
					log.Println("Аудио", inFile)
					csvTags, err = appendTags(inFile, csvTags)
					p, err := taglib.ReadProperties(inFile)
					if err != nil {
						log.Println("bit_rate=?", err)
					} else {
						fmt.Println("bit_rate=" + strconv.FormatUint(uint64(p.Bitrate), 10))
					}
				} else if !image {
					log.Println("Видео", inFile)
					csvTags, err = appendTags(inFile, csvTags)
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
			timeLine(out, file, keyVal("Description", header, r))
		}
		log.Print(args1)
		printTags(csvTags, false)
		return
	}
	appendTags(os.Args[1], nil)
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

func timeLine(in, file, description string) {
	log.Println(in, file, description)
	inFile := filepath.Join(in, file)
	res, err := os.Open(inFile + ".mov") // удерживаем
	if err != nil {
		res, err = os.Open(inFile + ".mp4")
		if err != nil {
			log.Println("Нет результата", inFile+".mov")
			log.Println("Нет результата", inFile+".mp4")
			return
		}
	}
	defer res.Close()
	// slices.SortFunc(yourSlice, func(a, b T) int { return a.Date.Compare(b.Date) })
	// flac, err := os.Open(file + ".flac")
	// if err == nil {
	// 	defer flac.Close()
	// 	if isFirstAfterSecond(flac.Name(), flac.Name()) {
	// 		// После записи звука во флак его тэггировали

	// 	}
	// }
	if isFirstAfterSecond(res.Name(), inFile+".mp3") ||
		isFirstAfterSecond(res.Name(), inFile+".flac") ||
		isFirstAfterSecond(res.Name(), inFile+".mp4") {
		if strings.HasSuffix(res.Name(), ".mov") {
			// DR 17
			log.Println("Значит результат в mov с lpcm. Пишем mp4 с alac, mp3, flac")
			rs, err := run(ctx, "ffmpeg", in,
				"-hide_banner",
				"-v", "error",
				"-i", file+".mov",
				"-c:v", "copy", "-c:a", "alac", "-y", file+".mp4",
				"-vn", "-compression_level", "12", "-y", file+".flac",
				"-vn", "-q", "0", "-joint_stereo", "0", "-y", file+".mp3",
			)
			if err == nil && rs == 0 {
				res.Close()
				log.Println("Удаляем mov", os.Remove(res.Name()))
			} else {
				log.Println("Не удалось сохранить файлы mp4, flac, mp3", err, "код завершения", rs)

			}
		} else {
			// DR>17 или mov уже удалили и в mp4 уже alac
			log.Println("Значит результат в mp4 с flac или alac. Пишем mp3, flac")
			rs, err := run(ctx, "ffmpeg", in,
				"-hide_banner",
				"-v", "error",
				"-i", file+".mp4",
				"-vn", "-compression_level", "12", "-y", file+".flac",
				"-vn", "-q", "0", "-joint_stereo", "0", "-y", file+".mp3",
			)
			if err != nil || rs != 0 {
				log.Println("Не удалось сохранить файлы flac, mp3", err, "код завершения", rs)
			}
		}
	} else {
		log.Println("Файлы mp4, flac, mp3 не требуют обновления")
	}
	if description != "" {
		log.Println("Тэггируем", description)
		for _, ext := range []string{".mp4", ".flac", ".mp3"} {
			tags := make(map[string][]string)
			tags["ARTIST"] = []string{"Юлия Абакумова"}
			taglib.WriteTags(inFile+ext, tags, 0)
		}
	}
}

func prop(key string, header map[string]int, r []string) {
	val := keyVal(key, header, r)
	if val != "" {
		fmt.Println(key + "=" + keyVal(key, header, r))
	}
}

var block = map[string]bool{
	taglib.Encoding:     true,
	"ENCODER":           true,
	"COMPATIBLE_BRANDS": true,
	"MINOR_VERSION":     true,
	"MAJOR_BRAND":       true,
	"CREATION_TIME":     true,
}

func printTags(tags map[string][]string, slash bool) {
	for k, vals := range tags {
		if slash {
			fmt.Println(k + "=" + strings.Join(vals, "/"))
		} else {
			for _, val := range vals {
				fmt.Println(k + "=" + val)
			}
		}
	}
}

func appendTags(fName string, allTags map[string][]string) (tags map[string][]string, err error) {
	tags = make(map[string][]string)
	for k, vals := range allTags {
		tags[k] = vals
	}
	ReadTags, err := taglib.ReadTags(fName)
	if err != nil {
		return
	}
	log.Println(fName)
	ReadTags = filtBlock(ReadTags)
	ReadTags = deDubl(ReadTags)
	printTags(ReadTags, false)
	for k, vals := range ReadTags {
		tags[k] = append(tags[k], vals...)
	}
	tags = deDubl(tags)
	return
}

func deDubl(inTag map[string][]string) (tags map[string][]string) {
	tags = make(map[string][]string)
	for k, vals := range inTag {
		has := make(map[string]bool)
		uniqs := []string{}
		for _, val := range vals {
			if has[val] {
				continue
			}
			has[val] = true
			uniqs = append(uniqs, val)
		}
		tags[k] = uniqs
	}
	return
}

func filtBlock(inTag map[string][]string) (tags map[string][]string) {
	tags = make(map[string][]string)
	for k, vals := range inTag {
		if block[k] {
			continue
		}
		tags[k] = vals
	}
	return
}
