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
	drt     = "drt"    // для консоли
	drTags  = "drTags" // для GUI
	dotCSV  = ".csv"
	dotMOV  = ".mov"
	dotMP4  = ".mp4"
	dotMP3  = ".mp3"
	dotFLAC = ".flac"
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
	probes  []string
	in      = bufio.NewScanner(os.Stdin)
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
		log.Println("nautilus", args)
	case len(os.Args) > 1:
		switch strings.ToLower(os.Args[1]) {
		case "-":
			//(echo a.mov&&echo comment=)|drt -
			dash = true
			for in.Scan() {
				args = append(args, in.Text())
			}
			log.Println("drt -", args)
		case "-h", "--help":
			help()
			return
		default:
			args = os.Args[1:]
		}
	}
	//---------------------------------------------------------------------------
	dirs := []string{}
	for _, file := range args {
		file, err := filepath.Abs(file)
		if err != nil {
			break
		}
		f, err := open(file)
		if err != nil {
			if err.Error() == "isDir" {
				dirs = append(dirs, f.Name())
				continue
			}
			break
		}
		f.Close()
		if _, ok := sources[file]; !ok {
			sources[file] = &ATT{}
		}
	}
	lenFD := len(sources) + len(dirs)
	if len(args) > lenFD {
		etc = args[lenFD:]
		argsTags = strings.Contains(strings.Join(etc, " "), "=")
	}

	if len(dirs) > 0 {
		log.Println("Слежу за", dirs)
	}

	// Нельзя делать цикл по sources так как drCSV вызывает timeLine который добавляет в sources
	for _, file := range mapKeys(sources, false) {
		// Только источники
		source := sources[file]
		out, album, ext, title := oaet(file)
		if ext == dotCSV {
			drCSV(album, out, file)
			continue
		}

		source.album = album
		source.title = title
		source.audio, probes = probe(filepath.Dir(file), filepath.Base(file), false)
		fmt.Println(append(probes, probeA(file, true)...))
		swrpp(file, source, nil)

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
		out, album, ext, _ := oaet(file)
		if ext == dotCSV {
			drCSV(album, out, file)
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
	for {
		// Выводим хэштэги
		for _, file := range mapKeys(sources) {
			e := Ext(file)
			if e == dotCSV {
				continue
			}
			if ht := sources[file].tags[HT]; len(ht) > 0 {
				log.Println(e, ht[0])
			}
		}
		etc = nil
		prompt := `Пустая строка подтверждает ввод, ^С прерывает ввод.
Введи "имя файла" или drag-n-drop или тэг=значение`
		fmt.Println(prompt)
		eof := false
	scan:
		for eof = true; in.Scan(); eof = true {
			eof = false
			s := strings.TrimSpace(in.Text())
			if s == "" {
				break
			}
			if !strings.Contains(s, `"`) && !strings.Contains(s, `'`) {
				// tags
				etc = append(etc, s)
				continue
			}
			// drag-n-drop?
			files, err := SplitCommandLine(s)
			// log.Println("drag-n-drop?", files, err)
			if err != nil {
				continue
			}
			for _, file := range files {
				f, err := open(file)
				if err == nil {
					f.Close()
				} else if err.Error() != "isDir" {
					log.Println("медиафайл?", file)
					fmt.Println(prompt)
					continue scan
				}
			}
			log.Println("drag-n-drop", files)
			// drag-n-drop
			for _, file := range files {
				f, err := open(file)
				if err == nil {
					switch Ext(file) {
					case dotCSV, dotMOV, dotMP4, dotFLAC, dotMP3:
					default:
						continue
					}
					// file
					f.Close()
					if _, ok := sources[file]; !ok {
						out, album, ext, title := oaet(file)
						source := &ATT{album, title, newTags(), false, ""}
						sources[file] = source
						if ext == dotCSV {
							drCSV(album, out, file)
						} else {
							source.audio, probes = probe(filepath.Dir(file), filepath.Base(file), false)
							fmt.Println(append(probes, probeA(file, true)...))
							swrpp(file, source, nil)
						}
					}
				} else if err.Error() == "isDir" {
					// dir
					dirs = append(dirs, file)
				}
			}
			if len(dirs) > 0 {
				log.Println("Слежу за", dirs)
			}
			fmt.Println(prompt)
		}
		if in.Err() != nil || eof {
			return
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
				swrpp(file, source, tags)
				// добавляет в sources
				if slices.Contains(probes, "format_name=mpegts") {
					mov := file + dotMOV
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
		if len(sources)+len(dirs) == 0 {
			// help
			break
		}
		if len(sources) == 0 {
			continue
		}
		log.Println(src)
		for _, file := range mapKeys(sources, false) {
			out, album, ext, _ := oaet(file)
			if ext == dotCSV {
				drCSV(album, out, file)
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
	help()
}

func swrpp(file string, att *ATT, tags Tags) {
	if tags != nil {
		att.tags.set("", tags)
		att.tags.write(file)
	}

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
			desktop,
			filepath.Join(xdg.DataDirs[0], `Microsoft\Windows\SendTo`, dr),
			adr,
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

	if !in.Scan() {
		return
	}
	switch strings.ToLower(strings.TrimSpace(in.Text())) {
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
			resTags.timeLine(album, out, fileName, dotCSV)
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
