package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type FATT struct {
	file  string
	album string
	title string
	tags  Tags
}

var (
	results = []FATT{}
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

// func (r Row) print(key string) {
// 	if val := r.val(key); val != "" {
// 		fmt.Println(key + "=" + val)
// 	}
// }

func (r Row) sprint(key string) (kv string) {
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

func (t *Tags) timeLine(album, in, file, a string) {
	if a == "" {
		return
	}
	var (
		mp3    = file + ".mp3"
		inMp3  = filepath.Join(in, mp3)
		flac   = file + ".flac"
		inFlac = filepath.Join(in, flac)
		mp4    = file + ".mp4"
		inMp4  = filepath.Join(in, mp4)
		mov    = file + ".mov"
		inMov  = filepath.Join(in, mov)
		gp3    = file + ".3gp"
		inGp3  = filepath.Join(in, gp3)
		alac   = file + ".alac.mov"
		inAlac = filepath.Join(in, alac)
	)
	res, err := open(inMov) // удерживаем
	if err != nil {
		res, err = open(inMp4)
		if err != nil {
			res, err = open(inAlac)
			if err != nil {
				log.Println("Нет результата в mov c lpcm", inMov)
				log.Println("Нет результата в mp4 c flac", inMp4)
				log.Println("Нет результата в mov c alac", inAlac)
				return
			}
		}
	}
	defer res.Close()
	flacMp3 := ""
	// lpcm := !strings.HasSuffix(res.Name(), alac) && !strings.HasSuffix(res.Name(), mp4)
	lpcm := false
	if isFirstAfterSecond(res.Name(), inMp3) ||
		isFirstAfterSecond(res.Name(), inFlac) ||
		lpcm && isFirstAfterSecond(res.Name(), inAlac) {
		base := filepath.Base(res.Name())
		args := []string{
			"-hide_banner",
			"-v", "error",
			"-i", base,
		}
		switch a {
		case "pcm_f32le", "pcm_s16le", "pcm_s24le", "pcm_s32le":
			lpcm = true
			flacMp3 = "alac.mov, flac , mp3"
			args = append(args,
				"-c:v", "copy", "-c:a", "alac", "-y", alac,
				"-vn", "-compression_level", "12", "-y", flac,
			)
		case "flac", "alac":
			flacMp3 = "flac, mp3"
			args = append(args,
				"-vn", "-compression_level", "12", "-y", flac,
			)
		default:
			flacMp3 = "mp3"
		}
		args = append(args,
			"-vn", "-q", "0", "-joint_stereo", "0", "-y", mp3,
		)
		log.Println("Результат в", filepath.Ext(base), "Создаём", flacMp3)
		rs, err := run(ctx, os.Stdout, "ffmpeg", in, args...)
		if err == nil && rs == 0 {
			if lpcm {
				res.Close()
				log.Println(res.Name(), "~>", inGp3, os.Rename(res.Name(), inGp3))
			}
		} else {
			log.Println("Не удалось создать файлы", flacMp3, err, "код завершения", rs)
		}
	} else {
		log.Println("Файлы", flacMp3, "моложе чем", res.Name())
	}

	t.print(2, file, false)
	t.parse(album, file)
	if argsTags {
		t.set("Из командной строки", newTags(etc...))
	}

	for i, args1 := range []string{inGp3, inAlac, inMp4, inFlac, inMp3} {
		f, err := open(args1)
		if err == nil {
			if i < 3 {
				log.Println(probe(filepath.Dir(args1), filepath.Base(args1), true))
			} else {
				log.Println(args1)
				log.Println(sprobeA(args1, true))
			}
			if argsTags {
				t.write(args1)
				readTags(args1).print(2, args1, false)
			} else {
				// Пригодится после консольного ввода тэгов
				results = append(results, FATT{args1, album, file, *t})
			}
			f.Close()
		}
	}
}
