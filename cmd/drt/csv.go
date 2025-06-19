package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type ATT struct {
	album  string
	title  string
	tags   Tags
	out    bool
	audio  string
	parent string
}

type Row struct {
	head map[string]int
	vals []string
}

func newRow(vals []string) *Row {
	if len(vals) == 0 {
		return nil
	}
	row := Row{head: make(map[string]int, len(vals))}
	for k, v := range vals {
		// log.Println(k, v)
		row.head[v] = k
	}
	return &row
}

func (r Row) val(key string) (val string) {
	if i, ok := r.head[key]; ok && len(r.vals) > i {
		return r.vals[i]
	}
	return
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

func (t *Tags) timeLine(album, in, title, a, fileCSV string) {
	var (
		probes []string
	)

	// flacMp3 := dotMP3
	// if Ext(title) == flacMp3 {
	// 	log.Println(flacMp3)
	// 	return
	// }
	res, err := open(title)
	if err == nil {
		// Не csv
		title = filepath.Base(title)
		title = trimExt(title)
	}
	b := func(e string) string {
		return title + e
	}
	p := func(e string) string {
		return filepath.Join(in, b(e))
	}
	if err != nil {
		// csv
		if watcher != nil {
			defer func() {
				if !slices.Contains(watcher.WatchList(), in) {
					err := watcher.Add(in)
					if err != nil {
						log.Println("Слежу за", in, err)
					}
				}
			}()
			if slices.Contains(watcher.WatchList(), in) {
				err := watcher.Remove(in)
				if err != nil {
					log.Println("Не слежу за", in, err)
				}
			}
		}
		for _, e := range dots {
			res, err = open(p(e))
			if err == nil {
				break
			}
		}
		if err != nil {
			// for _, e := range dots {
			// 	log.Println("Жду", p(e))
			// }
			log.Println("Жду появления", p(".*"))
			futures.Store(p(""), fileCSV)
			// log.Println(lenM(&futures))
			return
		} else {
			futures.Delete(p(""))
			// log.Println(lenM(&futures))
		}
	}
	resName := res.Name()
	res.Close()
	// Заменяю!
	base := filepath.Base(resName)
	e := Ext(resName)
	timeline := fileCSV != ""
	if timeline {
		a, probes = probe(in, base, false)
		fmt.Println(append(probes, probeA(resName, true)...))
	}
	pcm := false
	xlac := false
	outs := []string{resName}
	switch a {
	case "pcm_f32le", "pcm_s16le", "pcm_s24le", "pcm_s32le":
		xlac = true
		switch e {
		case dotMOV, dotMXF, dotAVI:
			pcm = true
			outs = append(outs, p(dotMP4), p(dotFLAC))
		default:
			outs = append(outs, p(dotFLAC))
		}
	case "flac", "alac":
		if e != dotFLAC {
			xlac = true
			outs = append(outs, p(dotFLAC))
		}
	}
	if e != dotMP3 {
		outs = append(outs, p(dotMP3))
	}

	if len(outs) > 1 {
		exts := []string{}
		for _, file := range outs[1:] {
			exts = append(exts, Ext(file))
		}
		if isFirstAfterSecond(resName, p(dotMP3)) ||
			xlac && isFirstAfterSecond(resName, p(dotFLAC)) ||
			pcm && isFirstAfterSecond(resName, p(dotMP4)) {
			args := []string{
				"-hide_banner",
				"-v", "error",
				"-i", base,
			}
			if pcm {
				args = append(args,
					"-c:v", "copy", "-c:a", "alac", "-y", b(dotMP4),
				)
			}
			if xlac {
				if a == "flac" {
					args = append(args,
						"-vn", "-c:a", "copy", "-y", b(dotFLAC),
					)
				} else {
					args = append(args,
						"-vn", "-compression_level", "12", "-y", b(dotFLAC),
					)
				}
			}
			if a == "mp3" {
				args = append(args,
					"-vn", "-c:a", "copy", "-y", b(dotMP3),
				)
			} else {
				args = append(args,
					"-vn", "-q", "0", "-joint_stereo", "0", "-y", b(dotMP3),
				)
			}
			log.Println(filepath.Ext(base), "~>", exts)
			rs, err := run(ctx, os.Stdout, "ffmpeg", in, args...)
			if err == nil && rs == 0 {
				if pcm {
					// res.Close()
					// log.Println(res.Name(), "~>", p(dotMP4), os.Remove(res.Name()))
					log.Println(resName, "~>", p(dotMP4))
					defer sources.Delete(resName)
				}
			} else {
				log.Println("Не удалось создать файлы", exts, err, "код завершения", rs)
			}
		} else {
			log.Println("Файлы", exts, "моложе чем", resName)
		}
	}

	if timeline {
		t.print(2, title+dotCSV, false)
	}
	t.parse(album, title)
	if argsTags {
		t.set("Из командной строки", newTags(etc...))
	}
	for i, file := range outs {
		f, err := open(file)
		if err == nil {
			f.Close()
			v, _ := sources.LoadOrStore(file, &ATT{})
			att := v.(*ATT)

			att.album = album
			att.title = title
			att.out = i > 0
			if i > 0 {
				att.audio, probes = probe(filepath.Dir(file), filepath.Base(file), false)
				fmt.Println(append(probes, probeA(resName, true)...))
			} else {
				att.audio = a
			}
			att.parent = fileCSV
			sources.Store(file, att)

			t.write(file)
			if i > 0 {
				now := time.Now()
				os.Chtimes(file, now, now)
			}

			att.tags = readTags(file)
			sources.Store(file, att)

		}
	}
}
