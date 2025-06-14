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
	out   bool
	audio string
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

func (t *Tags) timeLine(album, in, title, a string) {
	flacMp3 := dotMP3
	if Ext(title) == flacMp3 {
		log.Println(flacMp3)
		return
	}
	res, err := open(title)
	if err == nil {
		// Не csv
		title = filepath.Base(title)
		title = trimExt(title)
	}
	var (
		mp3    = title + dotMP3
		inMp3  = filepath.Join(in, mp3)
		flac   = title + dotFLAC
		inFlac = filepath.Join(in, flac)
		mp4    = title + dotMP4
		inMp4  = filepath.Join(in, mp4)
		mov    = title + dotMOV
		inMov  = filepath.Join(in, mov)
		probes []string
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
	timeline := a == dotCSV
	if timeline {
		a, probes = probe(in, base, false)
		fmt.Println(append(probes, probeA(res.Name(), true)...))
	}
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
			source, ok := sources[file]
			if !ok {
				source = &ATT{}
			}
			source.album = album
			source.title = title
			source.out = i > 0
			if i > 0 {
				source.audio, probes = probe(filepath.Dir(file), filepath.Base(file), false)
				fmt.Println(append(probes, probeA(res.Name(), true)...))
			}
			sources[file] = source

			t.write(file)

			sources[file].tags = readTags(file)

		}
	}
}
