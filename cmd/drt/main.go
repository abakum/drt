package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

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
	defer func() {
		fmt.Print("\r\nЖми <Enter>")
		os.Stdin.Read([]byte{0})
	}()

	// a/b.c
	args0 := filepath.Base(os.Args[0])
	// b.c
	args0 = strings.TrimSuffix(args0, filepath.Ext(args0))
	// b
	var (
		rc   uint32
		err  error
		file string
	)
	// log.Println(name)
	switch strings.ToLower(args0) {
	case "ffmpeg", "ffprobe":
		root := ""
		ok := false
		inp := ""
		for _, arg := range os.Args[1:] {
			if ok {
				inp = arg
				log.Println("Все файлы для", args0, "должны быть на том же диске что и infile", inp)
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
		rc, err = run(ctx, strings.ToLower(args0), root, os.Args[1:]...)
		if err != nil {
			panic(err)
		}

		os.Exit(int(rc))
	}

	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		ext := filepath.Ext(exe)
		for _, ff := range []string{"ffmpeg", "ffprobe"} {
			ffe := filepath.Join(dir, ff) + ext
			f, err := os.Open(ffe)
			if err == nil {
				f.Close()
			} else {
				m := "Можно сделать ссылку"
				if runtime.GOOS == "windows" {
					log.Printf(`%s 'mklink "%s" "%s"'`+"\n", m, ffe, exe)
				} else {
					log.Printf(`%s 'ln -s "%s" "%s"'`+"\r\n", m, exe, ffe)
				}
			}
		}
	}

	if len(os.Args) < 2 {
		fmt.Print(`Например: 20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор
Вот классические тэги:
Date=Дата записи как 20250227
Album=Запись как 20250227 Классный концерт
TrackNumber=Номер произведения как 02
Composer=Композитор как Шопен
Title=Название произведение как Баллада для фортепиано № 1 соль минор
MovementNumber=Если части произведения то их номера
Movement=Если части произведения то их названия
Artist=Исполнители как Юлия Абакумова
AlbumArtist=Остальные исполнители кроме солиста
Conductor=Руководители солиста как Владимир Дайч или оркестра или концертмейстер
Comment=Комментарий
Genre=Classical
InitialKey=Тональность как Gm. До-диез мажор как C#. Си-бемоль минор как Bbm. Си мажор как B.
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

	// Выводим сведения о  args1
	args1, err := filepath.Abs(os.Args[1])
	if err != nil {
		args1 = os.Args[1]
	}
	// a/b/c.d
	out := filepath.Dir(args1)
	// a/b
	album := filepath.Base(out)
	// b

	ext := filepath.Ext(args1)

	title := filepath.Base(args1)
	title = strings.TrimSuffix(title, ext)

	if strings.ToLower(ext) == ".csv" {
		out = filepath.Dir(out)
		// a

		f, err := os.Open(args1)
		if err != nil {
			log.Fatalln("Ошибка открытия", err)
			return
		}
		defer f.Close()

		// Читаем заголовок metadata.csv UTF-16 LE BOM
		utf := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
		nr := csv.NewReader(utf.NewDecoder().Reader(f))
		i := 0
		vals, err := nr.Read()
		i++
		if err != nil {
			log.Fatalln("Ошибка разбора заголовка", err, vals)
			return
		}
		row := newRow(vals)
		log.Println("Результаты в", out)
		csvTags := newTags()
		//Читаем остальные строки metadata.csv
		for {
			var err error
			row.vals, err = nr.Read()
			i++
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Println("Ошибка разбора строки", i, vals)
				continue
			}
			file := row.val("File Name")
			audio := row.val("Resolution") == ""
			image := row.val("Duration TC") == "00:00:00:01"
			in := row.val("Clip Directory")
			inFile := filepath.Join(in, file)
			if in != "" {
				if image {
					continue
				}
				// Файл
				fileTags := newTags()
				fileTags.csv(file, row, "Description", "Keywords", "Comments")
				if audio {
					fileTags.print(2, "Аудио "+inFile, true)
				} else {
					fileTags.print(2, "Видео "+inFile, true)
				}
				if len(fileTags) == 0 {
					fileTags.add("", readTags(inFile))
				}
				csvTags.add("", fileTags)
				if audio {
					probeA(inFile, false)
				} else {
					probeV(in, file)
					row.print("Resolution")
					row.print("Frame Rate")
					row.print("Video Codec")
				}
				row.print("Audio Bit Depth")
				row.print("Audio Sample Rate")
				row.print("Audio Codec")
				continue
			}
			resTags := newTags()
			resTags.csv(file, row, "Description", "Keywords", "Comments")
			resTags.timeLine(album, out, file)
		}
		csvTags.print(2, "Тэги из "+args1, false)
		return
	}

	// Это не csv
	fileTags := newTags()
	fileTags.add("Тэги из "+args1, readTags(args1))
	if len(os.Args) > 2 {
		fileTags.add("Тэги из командной строки", newTags(os.Args[2:]...))
		fileTags.parse(album, title)
		fileTags.write(args1)
	}
	probe(filepath.Dir(args1), filepath.Base(args1))
	probeA(args1, true)
}
