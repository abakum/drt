package main

//go install github.com/abakum/drt/cmd/drt@main
//go get github.com/cardinalby/xgo-pack@master
//go install github.com/cardinalby/xgo-pack
//xgo-pack init
//sudo rm -r dist/tmp
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
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	readme "github.com/abakum/drt"
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
	args0 = trimExt(filepath.Base(os.Args[0]))
	// b.c
	exe,
	dir,
	// a
	ext string
	// .c
	sources = make(map[string]*ATT) // Файлы источники и результатов
	etc     = []string{}            // Тэги
)

var _ = version.Ver

//go:generate go run github.com/abakum/version

//go:embed VERSION
var VERSION string

func main() {
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
				log.Printf(`%s 'mklink /h "%s" "%s"'`+"\n", m, ffe, exe)
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

	// Нельзя делать цикл по sources так как drCSV вызывает timeLine который добавляет в sources
	for _, file := range mapKeys(sources, false) {
		// Только источники
		source := sources[file]
		out, album, ext, title := oaet(file)
		if ext == ".csv" {
			drCSV(album, out, file)
			continue
		}

		a, probes := probe(filepath.Dir(file), filepath.Base(file), false)
		fmt.Println(append(probes, probeA(file, true)...))

		source.album = album
		source.title = title
		source.tags = readTags(file)
		source.tags.print(2, file, false)
		source.audio = a

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
	const (
		src = "Исходные медиафайлы------------------------------"
		trg = "Результирующие медиафайлы------------------------"
	)
	log.Println(src)
	for _, file := range mapKeys(sources, false) {
		if Ext(file) == ".csv" {
			continue
		}
		sources[file].tags.print(2, file, true)
	}
	results := mapKeys(sources, true)
	if len(results) > 0 {
		log.Println(trg)
		for _, file := range results {
			sources[file].tags.print(2, file, true)
		}
	}
	r := bufio.NewReader(os.Stdin)
	for {
		// Выводим хэштэги
		for _, file := range mapKeys(sources) {
			e := Ext(file)
			if e == ".csv" {
				continue
			}
			if ht := sources[file].tags[HT]; len(ht) > 0 {
				log.Println(e, ht[0])
			}
		}
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
			for _, file := range mapKeys(sources, false) {
				source := sources[file]
				a, probes := probe(filepath.Dir(file), filepath.Base(file), false)
				fmt.Println(append(probes, probeA(file, true)...))
				source.audio = a
				source.tags.set("", tags)
				source.tags.write(file)
				source.tags = readTags(file)
				// добавляет в sources
				if slices.Contains(probes, "format_name=mpegts") {
					mov := file + ".mov"
					if f, err := open(mov); err == nil {
						f.Close()
						sources[mov] = sources[file]
						delete(sources, file)
						file = mov
						_, probes = probe(filepath.Dir(file), filepath.Base(file), false)
						fmt.Println(append(probes, probeA(file, true)...))
					} else {
						args := []string{
							"-hide_banner",
							"-v", "error",
							"-i", filepath.Base(file),
							"-c", "copy", mov,
						}
						rs, err := run(ctx, os.Stdout, "ffmpeg", filepath.Dir(file), args...)
						if err == nil && rs == 0 {
							log.Println(file, "~>", mov)
							sources[mov] = sources[file]
							delete(sources, file)
							file = mov
							_, probes = probe(filepath.Dir(file), filepath.Base(file), false)
							fmt.Println(append(probes, probeA(file, true)...))
						} else {
							log.Println("Не удалось создать файл", mov, err, "код завершения", rs)
						}
					}
				} else {
					source.tags.timeLine(source.album, filepath.Dir(file), file, a)
				}
			}
		}
		log.Println(src)
		for _, file := range mapKeys(sources, false) {
			if Ext(file) == ".csv" {
				continue
			}
			swrpp(file, sources[file], tags)
		}
		results := mapKeys(sources, true)
		if len(results) > 0 {
			log.Println(trg)
			for _, file := range results {
				swrpp(file, sources[file], tags)
			}
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

// Упорядочим цикл по m
func mapKeys(m map[string]*ATT, out ...bool) (keys []string) {
	for k, v := range m {
		if v == nil {
			continue
		}
		if len(out) > 0 {
			if out[0] != v.out {
				continue
			}
		}
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return
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
	evtp()
	readme.Print()
	dr := drTags

	if len(xdg.ApplicationDirs) < 1 {
		log.Println("xdg.ApplicationDirs", xdg.ApplicationDirs)
		return
	}
	applications := xdg.ApplicationDirs[0]
	switch runtime.GOOS {
	case "darwin":
		dr += ".app" //dir
	case "windows":
		dr += ".lnk"
	default:
		dr += ".desktop"
	}
	adr := filepath.Join(applications, dr)
	oldname := filepath.Join(dir, drt) + ext

	verb := "install"
	f, err := os.Open(adr)
	if err == nil {
		// Установлен
		f.Close()
		verb = "uninstall"
		oldname = ""
	}
	if !yes(verb + " " + drTags) {
		return
	}
	defer ctrlC()

	desktop := filepath.Join(xdg.UserDirs.Desktop, dr)
	link := filepath.Join(dir, drTags)

	switch runtime.GOOS {
	case "darwin":
		install(oldname, adr, link)
	case "windows":
		install(oldname,
			adr,
			desktop,
			filepath.Join(xdg.DataDirs[0], `Microsoft\Windows\SendTo`, dr),
		)
	case "linux":
		sh := filepath.Join(xdg.DataHome, "nautilus/scripts", drTags)
		xdgDesktopIcon := "xdg-desktop-icon"

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

			for _, lnk := range []string{adr, link, sh} {
				log.Println(lnk, "~> /dev/null", os.Remove(lnk))
			}

			cmd = exec.CommandContext(ctx, "update-desktop-database", applications)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Println(cmd.Args, cmd.Run())

			return
		}
		install(oldname, desktop, sh, link, applications, adr, xdgDesktopIcon, verb)
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
	gui := trimExt(filepath.Base(exe)) != drt
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

	// Читаем .csv с UTF-16 LE BOM
	encoding := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	decoder := encoding.NewDecoder()
	reader := csv.NewReader(decoder.Reader(f))
	i := 0
	vals, err := reader.Read()
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
		row.vals, err = reader.Read()
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

func trimExt(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

func Keys[Map ~map[K]V, K comparable, V any](m Map) (keys []K) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func Values[Map ~map[K]V, K comparable, V any](m Map) (values []V) {
	for _, v := range m {
		values = append(values, v)
	}
	return
}

func ln(oldname, newname string, link, hard bool) (err error) {
	if link {
		opt := "/s"
		osLink := os.Symlink
		m := "symbolic"
		if hard {
			osLink = os.Link
			opt = ""
			m = "hard"
		}
		err = osLink(oldname, newname)
		log.Println("ln", opt, oldname, newname, err)
		if err == nil {
			return
		}
		log.Printf("Error creating %s link: %v\n", m, err)
		return
	}
	err = os.WriteFile(newname, []byte(`#!/usr/bin/env bash

set -o nounset
set -o errexit
`+oldname+` "${@}"`), 0744)
	if err != nil {
		log.Println("Error write .sh:", err)
	}
	return
}
