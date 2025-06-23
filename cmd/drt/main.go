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
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	readme "github.com/abakum/drt"
	version "github.com/abakum/version/lib"
	"github.com/adrg/xdg"
	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/flock"
	"github.com/xlab/closer"
	"golang.org/x/text/encoding/unicode"
)

const (
	drt     = "drt"    // для консоли
	drTags  = "drTags" // для GUI
	dotCSV  = ".csv"
	dotMOV  = ".mov"
	dotMXF  = ".mxf"
	dotAVI  = ".avi"
	dotWAV  = ".wav"
	dotAAC  = ".aac"
	dotMP4  = ".mp4"
	dotFLAC = ".flac"
	dotMP3  = ".mp3"
	dotJPG  = ".jpg"
	prompt  = `Пустая строка подтверждает ввод, ^С прерывает ввод.
Введи "имя файла" или drag-n-drop или тэг=значение`
)

var (
	dots = []string{dotCSV, dotMOV, dotMXF, dotAVI, dotWAV, dotAAC, // результаты DR
		dotMP4, dotFLAC, dotMP3} // Результаты drt - mp4 с alac
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
	// sources = make(map[string]*ATT) // Файлы источники и результатов
	sources  sync.Map
	sourcesR = func(f func(file string, att *ATT) bool) {
		sources.Range(func(key, value any) bool {
			return f(key.(string), value.(*ATT))
		})
	}
	sourcesL = func(file string) (att *ATT, ok bool) {
		v, o := sources.Load(file)
		if !o || v == nil {
			return nil, o
		}
		return v.(*ATT), o
	}

	etc     = []string{} // Тэги
	in      = bufio.NewScanner(os.Stdin)
	watcher *fsnotify.Watcher
	// futures = make(map[string]string) // Найти csv по mov mp4
	futures  sync.Map
	futuresL = func(file string) (fileCSV string, ok bool) {
		v, o := futures.Load(file)
		if !o || v == nil {
			return "", o
		}
		return v.(string), o
	}
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
				dirs = append(dirs, file)
				continue
			}
			break
		}
		f.Close()
		sources.Store(file, &ATT{})
	}

	lenFD := lenM(&sources) + len(dirs)
	if len(args) > lenFD {
		etc = args[lenFD:]
		argsTags = strings.Contains(strings.Join(etc, " "), "=")
	}

	for _, file := range mapKeys("*", false) {
		// Только источники
		att, _ := sourcesL(file)
		if att == nil {
			continue
		}
		out, album, ext, title := oaet(file)
		if ext == dotCSV {
			if argsTags || len(etc) > 0 || dash {
				drCSV(album, out, file)
			}
			continue
		}

		att.album = album
		att.title = title
		att.audio, _ = probe(filepath.Dir(file), filepath.Base(file), false)
		swrpp(file, att, nil)

		if argsTags {
			att.tags.set("Меняю", newTags(etc...))
			att.tags.write(file)
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
	watcher, err = fsnotify.NewWatcher()
	log.Println("Начал слежку", err)
	if err == nil {
		defer watcher.Close()

		isEmpty := make(chan string, 1000)
		notEmpty := make(chan string, 1000)
		remove := make(chan string, 1000)

		go func() {
			// isEmpty

			// Список проверяемых файлов
			var (
				files  sync.Map
				filesR = func(f func(file string, empty bool) bool) {
					files.Range(func(key, value any) bool {
						return f(key.(string), value.(bool))
					})
				}
			)

			t := time.NewTimer(time.Second)
			defer t.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-remove:
					if !ok {
						return
					}
					files.Delete(file)
					log.Println("Не жду", file)
				case file, ok := <-isEmpty:
					if !ok {
						return
					}
					if _, ok := files.Load(file); ok {
						// дубликаты
						continue
					}
					if Ext(file) == dotWAV {
						log.Println("DR пишет .wav файл так, что не понятно когда закончит.")
						log.Println("Для автоматического ожидания используй вместо .wav файла файл .mov c аудио частью в pcm")
						log.Println("Когда DR завершит запись .wav файла нажми <Enter>")
						// Чтоб это сообщение появлялось не чаще раз в минуту
						files.Store(file, false)
						time.AfterFunc(time.Minute, func() { remove <- file })
						continue
					}
					log.Println("Жду DR", file)
					files.Store(file, true)
					t.Reset(time.Second) // подождём твою маму
				case <-t.C:
					filesR(func(file string, empty bool) bool {
						if !empty {
							return true
						}
						if f, err := open(file); err == nil {
							f.Close()
							// Не пуст
							l := flock.New(file)
							ok, err := l.TryRLockContext(ctx, time.Second)
							fmt.Print(".")
							l.Close()
							if err != nil || !ok {
								t.Reset(time.Second) // подождём твою маму
								return true
							}
							fmt.Println()
							base := filepath.Base(file)
							dir := filepath.Dir(file)
							switch e := Ext(file); e {
							case ".dpx", ".cin", ".tif", ".ppm", ".bmp", ".xpm":
								jpg := fixDRd8(base, e)
								exported := base != jpg
								jpg = trimExt(jpg) + dotJPG
								args := []string{
									"-hide_banner",
									"-v", "error",
									"-i", base,
									"-q:v", "1", jpg,
								}
								rs, err := run(ctx, os.Stdout, "ffmpeg", dir, args...)
								log.Println(base, "~>", jpg, err)
								if err == nil && rs == 0 {
									if exported {
										// Удаляем только эксортированные
										os.Remove(file)
									}
									file = filepath.Join(dir, jpg)
								} else {
									log.Println("Не удалось создать файл", jpg, err, "код завершения", rs)
								}
							case ".jpg", ".png", dotFLAC:
								fileXXXXXXXX := file
								file = fixDRd8(file, e)
								if file != fileXXXXXXXX {
									err := os.Rename(fileXXXXXXXX, file)
									log.Println(base, "~>", filepath.Base(file), err)
									if err == nil {
										files.Delete(fileXXXXXXXX)
									} else {
										file = fileXXXXXXXX
									}
								}
							}
							files.Store(file, false)
							log.Println("Дождался", file)
							notEmpty <- file

							return true
						}
						t.Reset(time.Second) // подождём твою маму
						return true
					})
				}
			}
		}()

		go func() {
			// notEmpty
			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-notEmpty:
					if !ok {
						return
					}

					fileCSV := ""
					switch Ext(file) {
					case dotCSV:
						fileCSV = file
					default:
						if att, _ := sourcesL(file); att != nil {
							fileCSV = att.parent
						}
						woe := trimExt(file)
						if fileCSV == "" {
							fileCSV, _ = futuresL(woe)
						}
						if fileCSV == "" {
							continue
						}

						futures.Delete(woe)
					}
					// time.Sleep(time.Second)
					log.Println("Обрабатываю", fileCSV)
					sources.LoadOrStore(fileCSV, &ATT{})
					out, album, _, _ := oaet(fileCSV)
					drCSV(album, out, fileCSV)
					remove <- file
					doCSV(fileCSV)
					fmt.Println(prompt)
				}
			}
		}()

		// Start listening for events.
		go func() {
			defer log.Println("Закончил слежку", watcher.WatchList())
			for {
				select {
				case <-ctx.Done():
					log.Println("Context Done")
					return
				case event, ok := <-watcher.Events:
					if !ok {
						log.Println("Events Done")
						return
					}
					file := event.Name
					// log.Println(event.Op.String(), file)
					e := Ext(file)
					if event.Has(fsnotify.Remove) {
						log.Println("Пропал", file)
						continue
					}
					// }
					if event.Has(fsnotify.Create) {
						log.Println("Появился", file)
					} else if !event.Has(fsnotify.Write) {
						continue
					}
					switch e {
					case dotCSV, dotJPG, ".dpx", ".cin", ".tif", ".ppm", ".bmp", ".xpm", ".png":
						// Любой csv или картинка для конвертации или а png и jpg для переименования после экспорта
						isEmpty <- file
					case dotFLAC:
						file = fixDRd8(file, dotFLAC)
						fallthrough
					default:
						// Остальные если это результаты
						fileCSV := ""
						if att, _ := sourcesL(file); att != nil {
							fileCSV = att.parent
						}
						// Или будущие результаты
						if fileCSV == "" {
							fileCSV, _ = futuresL(trimExt(file))
						}
						if fileCSV != "" {
							// Но csv должен уже быть
							if f, err := open(fileCSV); err == nil {
								f.Close()
								isEmpty <- event.Name
							}
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						log.Println("Errors Done")
						return
					}
					log.Println("Ошибка слежки", err)
				}
			}
		}()
		for _, file := range dirs {
			log.Println("Слежу за", file, watcher.Add(file))
		}
	}

	const (
		src = "Исходные медиафайлы------------------------------"
		trg = "Результирующие медиафайлы------------------------"
	)
	log.Println(src)
	for _, file := range mapKeys("*", false) {
		out, album, ext, _ := oaet(file)
		if ext == dotCSV {
			drCSV(album, out, file)
			continue
		}
		if att, _ := sourcesL(file); att != nil {
			att.tags.print(2, file, true)
		}
	}

	results := mapKeys("*", true)
	if len(results) > 0 {
		log.Println(trg)
		for _, file := range results {
			if att, _ := sourcesL(file); att != nil {
				att.tags.print(2, file, true)
			}
		}
	}

	for {
		printHT("*")
		fmt.Println(prompt)

		etc = nil
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
			if s == `""` || s == `''` {
				log.Println("Очистил список исходных медиафайлов")
				sources = *new(sync.Map)
				continue
			}
			// drag-n-drop?
			files, err := SplitCommandLine(s)
			if err != nil {
				log.Println("drag-n-drop?", files, err)
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
				file, err := filepath.Abs(file)
				if err != nil {
					continue
				}
				f, err := open(file)
				if err == nil {
					f.Close()
					switch Ext(file) {
					case dotCSV, dotMOV, dotMP4, dotFLAC, dotMP3:
					default:
						continue
					}

					if _, ok := sources.Load(file); !ok {
						// Новые
						out, album, ext, title := oaet(file)
						att := &ATT{album, title, newTags(), false, "", ""}
						sources.Store(file, att)
						if ext == dotCSV {
							drCSV(album, out, file)
							doCSV(file)
						} else {
							att.audio, _ = probe(filepath.Dir(file), filepath.Base(file), false)
							swrpp(file, att, nil)
							printHT("*")
						}
					}
				} else if err.Error() == "isDir" {
					// dir
					if watcher != nil {
						err := watcher.Add(file)
						if err != nil {
							log.Println("Слежу за", file, err)
						}
					}
				}
			}
			if watcher != nil && len(watcher.WatchList()) > 0 {
				log.Println("Слежу за", watcher.WatchList())
			}
			fmt.Println(prompt)
		}
		if in.Err() != nil || eof {
			return
		}
		// Закончил ввод тэгов
		tags := newTags(etc...)
		if _, ok := tags["=="]; ok {
			delete(tags, "==")
			for _, file := range mapKeys("*", false) {
				att, _ := sourcesL(file)
				if att == nil {
					continue
				}

				a, probes := probe(filepath.Dir(file), filepath.Base(file), false)
				att.audio = a
				swrpp(file, att, tags)
				// добавляет в sources
				if slices.Contains(probes, "format_name=mpegts") {
					mov := file + dotMOV
					if f, err := open(mov); err == nil {
						f.Close()
						sources.Store(mov, att)
						sources.Delete(file)
						file = mov
						probe(filepath.Dir(file), filepath.Base(file), false)
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
							sources.Store(mov, att)
							sources.Delete(file)
							file = mov
							probe(filepath.Dir(file), filepath.Base(file), false)
						} else {
							log.Println("Не удалось создать файл", mov, err, "код завершения", rs)
						}
					}
				} else {
					att.tags.timeLine(att.album, filepath.Dir(file), file, a, "", probes...)
				}
			}
		}
		if lenM(&sources)+len(dirs) == 0 {
			// help
			break
		}
		if lenM(&sources) == 0 {
			continue
		}
		log.Println(src)
		for _, file := range mapKeys("*", false) {
			out, album, ext, _ := oaet(file)
			if ext == dotCSV {
				drCSV(album, out, file)
				continue
			}
			att, _ := sourcesL(file)
			swrpp(file, att, tags)
		}
		results := mapKeys("*", true)
		if len(results) > 0 {
			log.Println(trg)
			for _, file := range results {
				att, _ := sourcesL(file)
				swrpp(file, att, tags)
			}
		}
	}
	help()
}

// Set tags Write file Read file Print tags Parse att
func swrpp(file string, att *ATT, tags Tags) {
	if att == nil || file == "" {
		return
	}
	if tags != nil {
		att.tags.set("", tags)
		att.tags.write(file)
	}

	att.tags = readTags(file)
	att.tags.print(2, file, false)
	att.tags.parse(att.album, att.title)
}

// Упорядочим цикл по m.
// etc, flac, mp3
// out[0]=true только результаты
// out[0]=false только исходные
// parent==att.parent
func mapKeys(parent string, out ...bool) (keys []string) {
	var all []string
	sourcesR(func(file string, att *ATT) bool {
		if att == nil {
			return true
		}
		if len(out) > 0 {
			if out[0] != att.out {
				return true
			}
		}
		if parent != "*" && parent != att.parent {
			return true
		}
		all = append(all, file)
		return true
	})
	slices.Sort(all)
	for _, e := range dots {
		for _, key := range all {
			if Ext(key) == e {
				keys = append(keys, key)
			}
		}
	}

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

func drCSV(album, out, fileCSV string) {
	f, err := open(fileCSV)
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
			resTags.timeLine(album, out, fileName, "", fileCSV)
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
func lenM(m *sync.Map) int {
	var count int
	m.Range(func(k, v interface{}) bool {
		// log.Println(k, v)
		count++
		return true
	})
	return count
}

func doCSV(parent string) {
	sourcesR(func(file string, att *ATT) bool {
		if att != nil && att.parent == parent {
			// log.Println(file, att)
			swrpp(file, att, nil)
		}
		return true
	})

	printHT(parent)
}

// Вывожу хэштэги фильтр по parent.
// Если "*" не вывожу csv.
func printHT(parent string) {
	for _, file := range mapKeys(parent) {
		// log.Println(file)
		e := Ext(file)
		if e == dotCSV && parent == "*" {
			continue
		}
		if att, _ := sourcesL(file); att != nil && att.tags != nil && (parent == "*" || parent == att.parent) {
			if ht := att.tags[HT]; len(ht) > 0 {
				log.Println(e, ht[0])
			}
		}
	}
}

// DR вместо bar.flac пишет bar00047491.flac.
// DR вместо foo1.flac пишет foo1_00047491.flac.
func fixDRd8(file, ext string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(_?\d{8})(\%s)`, ext))
	return re.ReplaceAllString(file, "${2}")
}

func TestFixDRd8() {
	tests := []struct {
		input string
		want  string
	}{
		{"bar00047491.flac", "bar.flac"},
		{"foo1_00047491.flac", "foo1.flac"},
		{"1bar00047491.flac", "1bar.flac"},
		{"1foo1_00047491.flac", "1foo1.flac"},
		{"20250618 Экзамен 01 Чайковский Шесть пьес на одну тему для фортепиано части 1 2_00001137.flac", "20250618 Экзамен 01 Чайковский Шесть пьес на одну тему для фортепиано части 1 2.flac"},
		{"20250618 Экзамен 03 Шостакович Двадцать четыре прелюдии и фуги часть 14 ми-бемоль минор00047491.flac", "20250618 Экзамен 03 Шостакович Двадцать четыре прелюдии и фуги часть 14 ми-бемоль минор.flac"},
	}

	for _, tt := range tests {
		got := fixDRd8(tt.input, dotFLAC)
		if got != tt.want {
			log.Printf("fixDRd8(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
