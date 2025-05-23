package main

import (
	"bufio"
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
	ctx      context.Context
	cncl     context.CancelFunc
	argsTags bool
	etc      []string
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
			f, err := open(ffe)
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
		help()
		return
	}

	// drt file... fileX param... paramY
	files := []string{}
	// [file... fileX]
	etc = []string{}
	// [param... paramY]
	for _, args1 := range os.Args[1:] {
		args1, err := filepath.Abs(args1)
		if err != nil {
			break
		}
		f, err := open(args1)
		if err != nil {
			break
		}
		f.Close()
		files = append(files, args1)
	}
	if len(files) < 1 {
		help()
		return
	}

	if len(os.Args) > 1+len(files) {
		etc = os.Args[1+len(files):]
	}
	argsTags = strings.Contains(strings.Join(etc, " "), "=")

	for _, args1 := range files {
		// Выводим сведения о  args1
		out, album, ext, title := oaet(args1)
		if ext == ".csv" {
			out = filepath.Dir(out)
			// a

			f, err := open(args1)
			if err != nil {
				log.Fatalln("Ошибка открытия", err)
				continue
			}

			// Читаем заголовок metadata.csv UTF-16 LE BOM
			utf := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
			nr := csv.NewReader(utf.NewDecoder().Reader(f))
			i := 0
			vals, err := nr.Read()
			i++
			if err != nil {
				log.Fatalln("Ошибка разбора заголовка", err, vals)
				f.Close()
				continue
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
			f.Close()
			csvTags.print(2, "Тэги из "+args1, false)
			continue
		}

		// Это не csv
		fileTags := newTags()
		fileTags.add("Тэги из "+args1, readTags(args1))

		if argsTags {
			fileTags.set("Тэги из командной строки", newTags(etc...))
		}
		fileTags.parse(album, title)
		if len(fileTags) > 0 {
			fileTags.write(args1)
		}
		probe(filepath.Dir(args1), filepath.Base(args1))
		probeA(args1, true)
	}
	if argsTags {
		// drt file tag=
		return
	}
	// drt file foo
	// drt file
	if len(etc) == 0 {
		// drt file
		r := bufio.NewReader(os.Stdin)
		fmt.Println("Введи тэг=значение:")
		for {
			s, err := r.ReadString('\n')
			s = strings.TrimSpace(s)
			if err == nil && s != "" {
				etc = append(etc, s)
				continue
			}
			break
		}
	} else {
		// drt file foo
		return
	}
	// log.Println(etc)
	if len(etc) > 0 {
		for _, file := range files {
			_, album, ext, title := oaet(file)
			if ext == ".csv" {
				continue
			}
			t := readTags(file)
			t.set("Консольный ввод", newTags(etc...))
			// t.print(3, "Вот что пишем", false)
			t.parse(album, title)
			t.write(file)
		}
		for i, result := range results {
			t := tlTags[i]
			t.set("Консольный ввод", newTags(etc...))
			t.parse(albums[i], titles[i])
			t.write(result)
		}
	}
}

func oaet(args1 string) (out, album, ext, title string) {
	// a/b/c.d
	out = filepath.Dir(args1)
	// a/b
	album = filepath.Base(out)
	// b
	ext = filepath.Ext(args1)
	// .d
	title = filepath.Base(args1)
	// c.d
	title = strings.TrimSuffix(title, ext)
	// c
	ext = strings.ToLower(ext)
	return
}

func help() {
	fmt.Print(`drt file [...fileN] [tag1=val1 [...tagN=valN]]
Где file...fileN это медиафайлы или файлы .csv от DaVinci Resolve c Description или Keywords в которых указаны тэги.
Если в файле "\2025\20250227 Классный концерт\02.csv" есть таймлайн
"20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор" и клипы с незжатым звуком в
"\2025\20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор.mov" или 
"\2025\20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор.mp4"
то после запуска "drt 02.csv" вместо них создадутся файлы:
"20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор.alac.mov"
"20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор.flac"
"20250227 Классный концерт 02 Шопен Баллада для фортепиано № 1 соль минор.mp3"
с тэгами:
Date=20250227
Album=0250227 Классный концерт
TrackNumber=02
Composer=Шопен
Title=Баллада для фортепиано № 1 соль минор
InitialKey=Gm
Для знаков при ключе используется английская нотация где cи мажор как B, си-бемоль минор как Bbm, до-диез мажор как C#.
В Description или Keywords таймлайна для классики можно указать:
Composer=Фридерик Шопен
MovementNumber=Если части произведения то их номера
Movement=Если части произведения то их названия
Artist=Иван Петров
AlbumArtist=Остальные исполнители кроме солиста
Conductor=Руководители солиста или оркестра или концертмейстер
Genre=Classical
InvolvedPeople=Остальные люди и группы причастные к выступлению например РГК им С. В. Рахманинова
Lyricist=Авторы текста и переводчики
Arranger=Авторы переложения или оранжировки
Subtitle=Подзаголовок например Патетическая соната
Work=Авторские публикации или каталоги как BWV или opus posthumum как Op. 21
Grouping=Группировки например для музыкальных форм как Баллады для фортепиано
Если тэг один а значений несколько просто повторяйте строчки.
Так пишем в Keywords или Description:
Artist=Иван Петров
Artist=Пётр Сидоров
Или через / в Description:
MovementNumber=1/2
Или с новой строки в Description:
Movement=Скерцо
Адажио
Если Comments таймлайна не пуст то он запишется в тэг Comment.
Если в командной строке нет тэгов то их можно ввести в консоле.
Если в консольном вводе строка не начинается с тэга то это значение к предыдущему тэгу:
Artist=Иван Петров
Пётр Сидоров
Если в консольном вводе первая строка не начинается с тэга то это значение к тэгу Comment
Завершай консольный ввод пустой строкой. Чтоб ввести пустую строку в Comment введи /
Чтоб убрать все значение тэга X введи X=. Чтоб убрать значение всех тэгов введи =

Остальные тэги https://taglib.org/api/p_propertymapping.html
Расширенно про mp3 https://id3.org/id3v2.3.0
Страничка drTags https://github.com/abakum/drt
`)
	swap(os.UserHomeDir, "Desktop", "Переместить drTags на рабочий стол чтоб на него можно было бросать файлы для тэггирования",
		os.UserConfigDir, `Microsoft\Windows\SendTo`, "Переместить drTags в меню Отправить")

}

func yes(s string) (ok bool) {
	log.Output(3, s+"? y|yes|д|да")

	r := bufio.NewReader(os.Stdin)
	s, err := r.ReadString('\n')
	if err != nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "д", "да":
		return true
	}
	return
}

func swap(userDir func() (string, error), p, m string, userDirM func() (string, error), pM, mM string) {
	if runtime.GOOS != "windows" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	root, err := userDir()
	if err != nil {
		return
	}
	rootM, err := userDirM()
	if err != nil {
		return
	}
	drt := filepath.Join(root, p, filepath.Base(exe))
	drtM := filepath.Join(rootM, pM, filepath.Base(exe))
	f, err := open(drt)
	if err == nil {
		f.Close()
		if yes(mM) {
			rename(exe, drt, drtM)
		} else {
			rename(exe, drtM, drt)
		}
	} else {
		if yes(m) {
			rename(exe, drtM, drt)
		} else {
			rename(exe, drt, drtM)
		}
	}
}

func rename(exe, s, t string) {
	f, err := open(s)
	if err == nil {
		f.Close()
		log.Println(s, "~>", t, os.Rename(s, t))
	} else {
		f, err = open(t)
		if err != nil {
			log.Println(exe, "~>", t, mkLink(exe, t, true, true))
		} else {
			f.Close()
		}
	}
}
