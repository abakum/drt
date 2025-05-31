package main

//go get github.com/cardinalby/xgo-pack@master
//go install github.com/cardinalby/xgo-pack
//xgo-pack init
//#sudo apt install genisoimage
//sudo rm -r dist/tmp
//#sudo /home/koka/go/bin/xgo-pack build
//xgo-pack build
//goreleaser init
//goreleaser release --snapshot --clean
//cd /home/koka/src/drt/dist/linux_amd64
//sudo dpkg -r drt
//sudo dpkg -i drt.deb
//sudo desktop-file-install --set-key=Exec --set-value="drt %F" /usr/share/applications/drt.desktop

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	"github.com/xlab/closer"
	"golang.org/x/text/encoding/unicode"
)

const (
	drt    = "drt"    // для консоли
	drTags = "drTags" // для GUI
	del    = "Удаляю"
)

var (
	ctx      context.Context
	cncl     context.CancelFunc
	argsTags bool
	etc      []string
	win      = runtime.GOOS == "windows"
	// a/b.c
	args0 = filepath.Base(os.Args[0])
	// b.c
	exe,
	dir,
	// a
	ext string
	// .c
)

func main() {
	args0 = strings.TrimSuffix(args0, filepath.Ext(args0))
	// b
	var (
		rc   uint32
		file string
		err  error
	)

	log.SetFlags(log.Lshortfile)
	exe, err = os.Executable()
	if err != nil {
		if lp, err := exec.LookPath(args0); err == nil {
			exe = lp
		} else if abs, err := filepath.Abs(args0); err == nil {
			exe = abs
		} else {
			log.Fatalf("Где я? %v", err)
		}
	}

	ctx, cncl = context.WithCancel(context.Background())
	defer closer.Close()
	closer.Bind(cncl)

	// ctx, cncl = signal.NotifyContext(context.Background(), closer.DefaultSignalSet...)
	// defer cncl()

	// log.Println(name)
	switch strings.ToLower(args0) {
	case ffmpeg, ffprobe:
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

	dir = filepath.Dir(exe)
	ext = filepath.Ext(exe)
	for _, ff := range []string{ffmpeg, ffprobe} {
		ff = ff + ext
		if _, err := exec.LookPath(ff); err != nil {
			// Если не установлены ffmpeg или ffprobe
			ffe := filepath.Join(dir, ff)
			m := "Можно сделать ссылку"
			if win {
				log.Printf(`%s 'mklink -h "%s" "%s"'`+"\n", m, ffe, exe)
			} else {
				log.Printf(`%s 'ln "%s" "%s"'`+"\r\n", m, exe, ffe)
			}
		}
	}

	// drt file... fileX param... paramY
	files := []string{}
	// [file... fileX]
	etc = []string{}
	// [param... paramY]

	nautilus := os.Getenv("NAUTILUS_SCRIPT_SELECTED_FILE_PATHS")
	nautilus = strings.TrimSpace(nautilus)

	var args []string
	dash := false
	switch {
	case nautilus != "":
		args = strings.Split(nautilus, "\n")
	case len(os.Args) > 1:
		if os.Args[1] == "-" {
			dash = true
			bs, _ := os.ReadFile(os.Stdin.Name())
			args = strings.Split(strings.TrimSpace(string(bs)), "\n")
		} else {
			args = os.Args[1:]
		}
	default:
		help()
		return
	}

	for _, args1 := range args {
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

	if len(args) > len(files) {
		etc = args[len(files):]
	}
	argsTags = strings.Contains(strings.Join(etc, " "), "=")

	for _, args1 := range files {
		// Выводим сведения о  args1
		out, album, ext, title := oaet(args1)
		if ext == ".csv" {
			// out = filepath.Dir(out)
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
			// log.Println("Результаты в", out)
			// csvTags := newTags()
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
					// csvTags.add("", fileTags)
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
			// csvTags.print(2, "Тэги из "+args1, false)
			continue
		}

		// Это не csv

		probe(filepath.Dir(args1), filepath.Base(args1))
		probeA(args1, true)

		fileTags := readTags(args1)
		fileTags.parse(album, title)
		fileTags.print(2, args1, false)

		if argsTags {
			fileTags.set("Меняю", newTags(etc...))
			fileTags.write(args1)
			readTags(args1).print(2, args1, false)
		}

	}
	if argsTags || len(etc) > 0 || dash {
		// drt file tag=
		// drt file foo
		// drt -
		return
	}
	// drt file
	for _, result := range results {
		result.tags.print(2, result.file, true)
	}
	for _, file := range files {
		_, album, ext, title := oaet(file)
		if ext == ".csv" {
			continue
		}
		results = append(results, FATT{file, album, title, readTags(file)})
	}
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Пустая строка завершает ввод записью, ^С отменяет ввод. Введи тэг=значение:")
		etc = nil
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				log.Println(err)
				return
			}
			s = strings.TrimSpace(s)
			if s != "" {
				etc = append(etc, s)
				continue
			}
			break
		}
		// log.Println(etc)
		for _, result := range results {
			result.tags.set("", newTags(etc...))
			result.tags.write(result.file)

			result.tags = readTags(result.file)
			result.tags.parse(result.album, result.title)
			result.tags.print(2, result.file, false)
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
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема" и клип с lpcm в
"\2025\20250227 Классный концерт 02 Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.mov" 
то после запуска "drt 02.csv" появятся файлы:
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.alac.mov"
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.flac"
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.mp3"
или был клип с flac в
"\2025\20250227 Классный концерт 02 Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.mp4"
то после запуска "drt 02.csv" появятся файлы:
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.flac"
"20250227 Классный концерт 02 Фридерик_Шопен Баллада для фортепиано №1 соль минор части Вступление Первая_тема.mp3"
с тэгами:
Date=20250227
Album=0250227 Классный концерт
TrackNumber=02
Composer=Фридерик Шопен
Title=Баллада для фортепиано №1 соль минор
MovementName=Вступление
MovementName=Первая тема
Grouping=Фортепиано
Grouping=Баллада для фортепиано
InitialKey=Gm
Для знаков при ключе используется английская нотация где cи мажор как B, си-бемоль минор как Bbm, до-диез мажор как C#.
В Description или Keywords таймлайна для классики можно указать:
TitleSort=Тогда это строчка будет источником для тэгов а не имя файла
Title=Баллада для фортепиано №1 соль минор
Composer=Фридерик Шопен
Artist=Иван Петров
AlbumArtist=Остальные исполнители кроме солиста
Conductor=Руководители солиста или оркестра или концертмейстер
Genre=Classical
InvolvedPeople=Остальные люди например Перевертмейстер и группы причастные к выступлению например РГК им С. В. Рахманинова
Lyricist=Авторы текста и переводчики
Arranger=Авторы переложения или оранжировки
Subtitle=Подзаголовок например Патетическая соната
Work=Авторские публикации или каталоги как BWV или opus posthumum как Op.21
Grouping=Группировки например для музыкальных форм как Баллада для фортепиано или по инструментам
Если тэг один а значений несколько просто повторяй строчки.
Так пиши в Keywords или Description:
Artist=Иван Петров
Artist=Пётр Сидоров
Или через / в Description:
MovementName=Вступление/Первая тема
Или с новой строки в Description:
Movement=Вступление
Первая тема
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
	switch runtime.GOOS {
	case "windows":
		swap(ST{os.UserHomeDir, "Desktop", "Переместить drTags на рабочий стол чтоб на него можно было бросать файлы для тэггирования", "", ""},
			ST{os.UserConfigDir, `Microsoft\Windows\SendTo`, "Переместить drTags в меню Отправить", "", ""})
	case "linux":
		dr := drTags + ".desktop"
		sh := filepath.Join(xdg.DataHome, "nautilus/scripts", drTags)
		hard := filepath.Join(dir, drTags)
		desktop := path.Join(xdg.UserDirs.Desktop, dr)
		f, err := open(desktop)
		xdgDesktopIcon := "xdg-desktop-icon"

		application := path.Join(xdg.DataHome, "application")
		local := path.Join(application, dr)
		verb := "install"
		if err == nil {
			f.Close()
			verb = "uninstall"
		}
		if !yes(verb + " " + drTags) {
			return
		}
		defer ctrlC()
		if verb == "uninstall" {
			cmd := exec.CommandContext(ctx, xdgDesktopIcon, verb, desktop)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			log.Println(cmd.Args, err)
			if err != nil {
				log.Println(desktop, "~> /dev/null", os.Remove(desktop))
			}

			log.Println(local, "~> /dev/null", os.Remove(local))

			cmd = exec.CommandContext(ctx, "update-desktop-database", application)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Println(cmd.Args, cmd.Run())

			log.Println(sh, "~> /dev/null", os.Remove(sh))

			log.Println(hard, "~> /dev/null", os.Remove(hard))
			return
		}
		// install
		if f, err = open(hard); err == nil {
			f.Close()
		} else {
			mkLink(filepath.Join(dir, drt), hard, true, true)
		}

		ex := drTags
		if _, err := exec.LookPath(ex); err != nil {
			//Если не в путёвом
			ex = hard
		}

		log.Println("Создаю меню для nautilus", sh,
			os.WriteFile(sh, []byte(fmt.Sprintf(`#!/bin/bash
gnome-terminal --title %s -- %s`, drTags, drTags)), 0744))
		deskTop(desktop, ex)
		cmd := exec.CommandContext(ctx, "desktop-file-install", "--rebuild-mime-info-cache", desktop, "--dir="+application) //
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Println(cmd.Args, cmd.Run())
		if f, err = open(local); err == nil {
			f.Close()
		} else {
			deskTop(local, ex)
		}
		cmd = exec.CommandContext(ctx, xdgDesktopIcon, verb, "--novendor", local)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Println(cmd.Args, cmd.Run())
	}
}

func deskTop(desktop, ex string) {
	log.Println("Создаю ярлык", desktop,
		os.WriteFile(desktop, []byte(`[Desktop Entry]
Name=drTags
Type=Application
Exec=`+ex+` %F
Terminal=true
Icon=edit-find-replace
NoDisplay=false
MimeType=text/csv;audio/mpeg;audio/flac;audio/mp4;video/mp4;video/quicktime;
Categories=AudioVideo;AudioVideoEditing;
Keywords=media info;metadata;tag;video;audio;codec;csv;mp3;flac;m4a;mp4;mov;davinci;resolve
`), 0644))
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

type ST struct {
	userDir  func() (string, error)
	p, m     string
	root, dr string
}

func swap(sts ...ST) {
	stm := make(map[bool]ST, 2)
	var err error
	for i, st := range sts {
		st.root, err = st.userDir()
		if err != nil {
			return
		}
		st.dr = filepath.Join(st.root, st.p, drTags+ext)
		stm[i > 0] = st
	}
	i := false
	f, err := open(stm[!i].dr)
	if err == nil {
		f.Close()
	} else {
		i = !i
	}
	if yes(stm[i].m) {
		defer ctrlC()
	} else {
		i = !i
	}
	rename(exe, stm[!i].dr, stm[i].dr)
}

func rename(exe, s, t string) {
	f, err := open(s)
	if err == nil {
		f.Close()
		log.Println(s, "~>", t, os.Rename(s, t))
	} else {
		f, err = open(t)
		if err != nil {
			log.Println(exe, "->", t, mkLink(exe, t, true, true))
		} else {
			f.Close()
		}
	}
}

func ctrlC() {
	if args0 != drTags {
		return
	}
	log.Println("Жми ^C")
	closer.Hold()
}
