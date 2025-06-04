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

//go install github.com/tc-hib/go-winres@latest
//go-winres init
//go get github.com/abakum/version
//go generate
//ie4uinit.exe -show

import (
	"bufio"
	"context"
	_ "embed"
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

	version "github.com/abakum/version/lib"
	"github.com/adrg/xdg"
	"github.com/xlab/closer"
	"golang.org/x/text/encoding/unicode"
)

const (
	drt    = "drt"    // для консоли
	drTags = "drTags" // для GUI
)

var (
	ctx      context.Context
	cncl     context.CancelFunc
	argsTags bool // Тэги в командной строке
	win      = runtime.GOOS == "windows"
	// a/b.c
	args0 = filepath.Base(os.Args[0])
	// b.c
	exe,
	dir,
	// a
	ext string
	// .c
	sources = make(map[string]*ATT) // Файлы источники
	etc     = []string{}            // Тэги
)

var _ = version.Ver

//go:generate go run github.com/abakum/version

//go:embed VERSION
var VERSION string

func main() {
	args0 = trimExt(args0)
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
	wd, _ := os.Getwd()
	log.Println(exe, VERSION, wd)

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
		rc, err = run(ctx, os.Stdout, strings.ToLower(args0), root, os.Args[1:]...)
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
	//---------------------------------------------------------------------------
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
		if _, ok := sources[args1]; !ok {
			sources[args1] = &ATT{}
		}
	}
	if len(sources) < 1 {
		help()
		return
	}

	if len(args) > len(sources) {
		etc = args[len(sources):]
	}
	argsTags = strings.Contains(strings.Join(etc, " "), "=")

	for file, source := range sources {
		out, album, ext, title := oaet(file)
		if ext == ".csv" {
			drCSV(album, out, file)
			continue
		}

		_, probes := probe(filepath.Dir(file), filepath.Base(file), false)
		fmt.Println(append(probes, probeA(file, true)...))

		source.album = album
		source.title = title
		source.tags = readTags(file)
		source.tags.print(2, file, false)

		source.tags.parse(album, title)

		if argsTags {
			source.tags.set("Меняю", newTags(etc...))
			source.tags.write(file)
			readTags(file).print(2, file, false)
		}
	}
	if argsTags || len(etc) > 0 || dash {
		// drt file tag=
		// drt file foo
		// drt -
		return
	}
	// drt file
	for file, result := range results {
		result.tags.print(2, file, true)
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
		tags := newTags(etc...)
		if _, ok := tags["=="]; ok {
			delete(tags, "==")
			// Нельзя делать цикл по sources так как timeLine добавляет в sources
			files := []string{} // Список файлов
			for file := range sources {
				files = append(files, file)
			}
			for _, file := range files {
				a, probes := probe(filepath.Dir(file), filepath.Base(file), false)
				fmt.Println(append(probes, probeA(file, true)...))
				source := sources[file]
				source.tags.set("", tags)
				source.tags.write(file)
				source.tags = readTags(file)
				// добавляет и в sources и в  results
				source.tags.timeLine(source.album, filepath.Dir(file), file, a)
			}
		}
		for file, att := range sources {
			_, _, ext, _ := oaet(file)
			if ext == ".csv" {
				// drCSV(album, out, file)
				continue
			}
			if Ext(file) == ".csv" {
				continue
			}
			swrpp(file, att, tags)
		}
		for file, att := range results {
			swrpp(file, att, tags)
		}
	}
}
func swrpp(file string, att *ATT, tags Tags) {
	att.tags.set("", tags)
	att.tags.write(file)

	att.tags = readTags(file)
	att.tags.print(2, file, false)
	att.tags.parse(att.album, att.title)
}

func Ext(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

func oaet(args1 string) (out, album, ext, title string) {
	// a/b/c.d
	out = filepath.Dir(args1)
	// a/b
	album = filepath.Base(out)
	// b
	ext = Ext(args1)
	// .d
	title = filepath.Base(args1)
	// c.d
	// title = strings.TrimSuffix(title, ext)
	title = trimExt(title)
	// c
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
Если в видеофайле звук:
 - в pcm и ввести == то запишу .mp4 со звуком в alac, .flac, .mp3
 - в alac или flac и ввести == то запишу .flac, .mp3
 - иначе запишу a.mp3
Если в аудиофайле звук:
 - в pcm в alac или flac и ввести == то запишу a.flac, a.mp3
 - иначе запишу a.mp3 если аудиофайл не .mp3

Остальные тэги https://taglib.org/api/p_propertymapping.html
Расширенно про mp3 https://id3.org/id3v2.3.0
Страничка drTags https://github.com/abakum/drt
`)
	dr := drTags + ".desktop"
	if win {
		dr = drTags + ".lnk"
	}
	desktop := filepath.Join(xdg.UserDirs.Desktop, dr)
	verb := "install"
	f, err := open(desktop)
	if err == nil {
		f.Close()
		verb = "uninstall"
	}
	if !yes(verb + " " + drTags) {
		return
	}
	defer ctrlC()
	oldname := filepath.Join(dir, drt) + ext

	switch runtime.GOOS {
	case "windows":
		lnks := []string{
			desktop,
			filepath.Join(xdg.DataDirs[0], `Microsoft\Windows\SendTo`, dr),
			filepath.Join(xdg.DataDirs[0], `Microsoft\Windows\Start Menu\Programs`, dr),
		}
		if verb == "uninstall" {
			for _, lnk := range lnks {
				log.Println(lnk, "~> nul", os.Remove(lnk))
			}
			return
		}
		install(oldname, lnks...)

		// swap(ST{os.UserHomeDir, "Desktop", "Переместить drTags на рабочий стол чтоб на него можно было бросать файлы для тэггирования", "", ""},
		// ST{os.UserConfigDir, `Microsoft\Windows\SendTo`, "Переместить drTags в меню Отправить", "", ""})
	case "linux":
		link := filepath.Join(dir, drTags)
		sh := filepath.Join(xdg.DataHome, "nautilus/scripts", drTags)
		xdgDesktopIcon := "xdg-desktop-icon"

		application := path.Join(xdg.DataHome, "application")
		local := path.Join(application, dr)
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

			// log.Println(local, "~> /dev/null", os.Remove(local))
			for _, lnk := range []string{local, sh, link} {
				log.Println(lnk, "~> /dev/null", os.Remove(lnk))
			}

			cmd = exec.CommandContext(ctx, "update-desktop-database", application)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Println(cmd.Args, cmd.Run())

			// log.Println(sh, "~> /dev/null", os.Remove(sh))

			// log.Println(link, "~> /dev/null", os.Remove(link))
			return
		}
		install(oldname, desktop, sh, link, application, local, xdgDesktopIcon, verb)
		// 		if f, err = open(link); err == nil {
		// 			f.Close()
		// 		} else {
		// 			mkLink(oldname, link, true, true)
		// 		}

		// 		ex := drTags
		// 		if _, err := exec.LookPath(ex); err != nil {
		// 			//Если не в путёвом
		// 			ex = link
		// 		}

		// 		log.Println("Создаю меню для nautilus", sh,
		// 			os.WriteFile(sh, []byte(fmt.Sprintf(`#!/bin/bash
		// gnome-terminal --title %s -- %s`, drTags, drTags)), 0744))
		// 		deskTop(desktop, ex)
		// 		cmd := exec.CommandContext(ctx, "desktop-file-install", "--rebuild-mime-info-cache", desktop, "--dir="+application) //
		// 		cmd.Stdin = os.Stdin
		// 		cmd.Stdout = os.Stdout
		// 		cmd.Stderr = os.Stderr
		// 		log.Println(cmd.Args, cmd.Run())
		// 		if f, err = open(local); err == nil {
		// 			f.Close()
		// 		} else {
		// 			deskTop(local, ex)
		// 		}
		// 		cmd = exec.CommandContext(ctx, xdgDesktopIcon, verb, "--novendor", local)
		// 		cmd.Stdin = os.Stdin
		// 		cmd.Stdout = os.Stdout
		// 		cmd.Stderr = os.Stderr
		// 		log.Println(cmd.Args, cmd.Run())
	}
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

func ctrlC() {
	gui := args0 == drTags
	if win {
		gui = !strings.HasPrefix(os.Environ()[0], "=")
	}
	if !gui {
		return
	}
	log.Println("Жми ^C")
	closer.Hold()
}

func drCSV(album, out, args1 string) {
	f, err := open(args1)
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
	// csvTags := newTags()
	//Читаем остальные строки metadata.csv
	for {
		var err error
		row.vals, err = nr.Read()
		i++
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			log.Println("Ошибка разбора строки", i, vals)
			continue
		}
		fileName := row.val("File Name")
		in := row.val("Clip Directory")
		if in == "" {
			// timeLine
			resTags := newTags()
			resTags.csv(fileName, row, "Description", "Keywords", "Comments")
			resTags.timeLine(album, out, fileName, "csv")
			continue
		}
		// image := row.val("Duration TC") == "00:00:00:01"
		// if image {
		// 	continue
		// }
		// fileTags := newTags()
		// fileTags.csv(fileName, row, "Description", "Keywords", "Comments")
		// inFile := filepath.Join(in, fileName)
		// audio := row.val("Resolution") == ""
		// if audio {
		// 	fileTags.print(2, "Аудио "+inFile, true)
		// } else {
		// 	fileTags.print(2, "Видео "+inFile, true)
		// }
		// if len(fileTags) == 0 {
		// 	fileTags.add("", readTags(inFile))
		// }
		// if audio {
		// 	log.Println(probeA(inFile, false),
		// 		row.print("Audio Bit Depth"),
		// 		row.print("Audio Sample Rate"),
		// 		row.print("Audio Codec"))

		// } else {
		// 	_, probes := probe(in, fileName, true)
		// 	log.Println(probes,
		// 		row.print("Resolution"),
		// 		row.print("Frame Rate"),
		// 		row.print("Video Codec"),
		// 		row.print("Audio Bit Depth"),
		// 		row.print("Audio Sample Rate"),
		// 		row.print("Audio Codec"),
		// 	)
		// }
	}
}

func trimExt(args0 string) string {
	return strings.TrimSuffix(args0, filepath.Ext(args0))
}
