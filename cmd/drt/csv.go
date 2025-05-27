package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	// albums,
	// titles,
	results []string
	tlTags  []Tags
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

func (r Row) print(key string) {
	if val := r.val(key); val != "" {
		fmt.Println(key + "=" + val)
	}
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

func (t *Tags) timeLine(album, in, file string) {
	var (
		mp3    = file + ".mp3"
		inMp3  = filepath.Join(in, mp3)
		flac   = file + ".flac"
		inFlac = filepath.Join(in, flac)
		mp4    = file + ".mp4"
		inMp4  = filepath.Join(in, mp4)
		mov    = file + ".mov"
		inMov  = filepath.Join(in, mov)
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
	// slices.SortFunc(yourSlice, func(a, b T) int { return a.Date.Compare(b.Date) })
	if isFirstAfterSecond(res.Name(), inMp3) ||
		isFirstAfterSecond(res.Name(), inFlac) ||
		isFirstAfterSecond(res.Name(), inAlac) {
		// .mov
		// .mp4
		// .alac.mov
		if strings.HasSuffix(res.Name(), alac) {
			// .alac.mov
			log.Println("Результат в", alac, ". Создаём mp3, flac")
			rs, err := run(ctx, "ffmpeg", in,
				"-hide_banner",
				"-v", "error",
				"-i", alac,
				"-vn", "-compression_level", "12", "-y", flac,
				"-vn", "-q", "0", "-joint_stereo", "0", "-y", mp3,
			)
			if err != nil || rs != 0 {
				log.Println("Не удалось создать файлы flac, mp3", err, "код завершения", rs)
			}
		} else {
			// .mov
			// .mp4
			base := filepath.Base(res.Name())
			log.Println("Результат в", base, "Создаём alac.mov, mp3, flac")
			rs, err := run(ctx, "ffmpeg", in,
				"-hide_banner",
				"-v", "error",
				"-i", base,
				"-c:v", "copy", "-c:a", "alac", "-y", alac,
				"-vn", "-compression_level", "12", "-y", flac,
				"-vn", "-q", "0", "-joint_stereo", "0", "-y", mp3,
			)
			if err == nil && rs == 0 {
				res.Close()
				log.Println("Удаляем", res.Name(), os.Remove(res.Name()))
			} else {
				log.Println("Не удалось создать файлы alac.mov, flac, mp3", err, "код завершения", rs)
			}
		}
	} else {
		log.Println("Файлы flac, mp3 моложе чем", res.Name())
	}

	t.parse(album, file)
	if argsTags {
		t.set("Из командной строки", newTags(etc...))
	}

	for i, args1 := range []string{inAlac, inFlac, inMp3} {
		f, err := open(args1)
		if err == nil {
			if i == 0 {
				probe(filepath.Dir(args1), filepath.Base(args1))
			} else {
				log.Println(args1)
				probeA(args1, true)
			}
			if argsTags {
				t.write(args1)
			} else {
				// Пригодится после консольного ввода тэгов
				// albums = append(albums, album)
				// titles = append(titles, file)
				results = append(results, args1)
				tlTags = append(tlTags, *t)
			}
			f.Close()
		}
	}
}
