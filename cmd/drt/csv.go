package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type ATT struct {
	album string
	title string
	tags  Tags
}

var (
	results = make(map[string]*ATT)
)

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
		fmt.Println(k, v)
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

func (r Row) print(key string) (kv string) {
	if val := r.val(key); val != "" {
		kv = key + "=" + val
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

func (t *Tags) timeLine(album, in, title, a string) {
	flacMp3 := ".mp3"
	if Ext(title) == flacMp3 {
		log.Println(flacMp3)
		return
	}
	res, err := open(title)
	if err == nil {
		// Не csv
		title = filepath.Base(title)
		// title = strings.TrimSuffix(title, filepath.Ext(title))
		title = trimExt(title)
	}
	var (
		mp3    = title + flacMp3
		inMp3  = filepath.Join(in, mp3)
		flac   = title + ".flac"
		inFlac = filepath.Join(in, flac)
		mp4    = title + ".mp4"
		inMp4  = filepath.Join(in, mp4)
		mov    = title + ".mov"
		inMov  = filepath.Join(in, mov)
	)
	if err != nil {
		res, err = open(inMp4) //flac.mp4 alac.mp4
		if err != nil {
			res, err = open(inMov)
			if err != nil {
				log.Println("Нет результата в mp4 c flac или alac", inMp4)
				log.Println("Нет результата в mov c lpcm", inMov)
				return
			}
		}
	}
	defer res.Close()
	// Заменяю!
	base := filepath.Base(res.Name())
	timeline := a == "csv"
	if timeline {
		var probes []string
		a, probes = probe(in, base, false)
		fmt.Println(append(probes, probeA(res.Name(), true)...))
		sources[res.Name()] = &ATT{album, title, *t}
	}
	// lpcm := !strings.HasSuffix(res.Name(), alac) && !strings.HasSuffix(res.Name(), mp4)
	lpcm := false
	xlac := false
	outs := []string{res.Name()}
	switch a {
	case "pcm_f32le", "pcm_s16le", "pcm_s24le", "pcm_s32le":
		lpcm = true
		xlac = true
		flacMp3 = ".mp4, .flac, .mp3"
		outs = append(outs, inMp4, inFlac)
	case "flac", "alac":
		xlac = true
		flacMp3 = ".flac, .mp3"
		outs = append(outs, inFlac)
	}
	outs = append(outs, inMp3)
	if isFirstAfterSecond(res.Name(), inMp3) ||
		xlac && isFirstAfterSecond(res.Name(), inFlac) ||
		lpcm && isFirstAfterSecond(res.Name(), inMp4) {
		args := []string{
			"-hide_banner",
			"-v", "error",
			"-i", base,
		}
		if lpcm {
			args = append(args,
				"-c:v", "copy", "-c:a", "alac", "-y", mp4,
			)
		}
		if xlac {
			if a == "flac" {
				args = append(args,
					"-vn", "-c:a", "copy", "-y", flac,
				)
			} else {
				args = append(args,
					"-vn", "-compression_level", "12", "-y", flac,
				)
			}
		}
		if a == "mp3" {
			args = append(args,
				"-vn", "-c:a", "copy", "-y", mp3,
			)
		} else {
			args = append(args,
				"-vn", "-q", "0", "-joint_stereo", "0", "-y", mp3,
			)
		}
		log.Println(filepath.Ext(base), "~>", flacMp3)
		rs, err := run(ctx, os.Stdout, "ffmpeg", in, args...)
		if err == nil && rs == 0 {
			if lpcm {
				res.Close()
				// log.Println(res.Name(), "~>", inMp4, os.Remove(res.Name()))
				log.Println(res.Name(), "~>", inMp4)
			}
		} else {
			log.Println("Не удалось создать файлы", flacMp3, err, "код завершения", rs)
		}
	} else {
		log.Println("Файлы", flacMp3, "моложе чем", res.Name())
	}

	if timeline {
		t.print(2, "TimeLine "+title, false)
	}
	t.parse(album, title)
	if argsTags {
		t.set("Из командной строки", newTags(etc...))
	}

	for i, args1 := range outs {
		f, err := open(args1)
		if err == nil {
			if i > 0 {
				// Кроме исходного
				_, probes := probe(filepath.Dir(args1), filepath.Base(args1), false)
				fmt.Println(append(probes, probeA(res.Name(), true)...))
			}
			if argsTags {
				t.write(args1)
				readTags(args1).print(2, args1, false)
			} else {
				// Пригодится после консольного ввода тэгов
				if _, ok := results[args1]; !ok && i > 0 {
					// Кроме исходного
					results[args1] = &ATT{album, title, *t}
				}
			}
			f.Close()
		}
	}
}
