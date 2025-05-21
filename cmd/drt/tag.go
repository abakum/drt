package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/abakum/go-taglib"
)

type Tags map[string][]string

// Выведем t с title.
func (t Tags) print(calldepth int, title string, slash bool) {
	if title == "" || len(t) == 0 {
		return
	}
	if title != " " {
		// log.Println(title)
		log.Output(calldepth, title)
	}
	for k, vals := range t {
		if slash {
			fmt.Println(k + "=" + strings.Join(vals, "/"))
		} else {
			for _, val := range vals {
				fmt.Println(k + "=" + val)
			}
		}
	}
	if len(t) > 0 {
		fmt.Println()
		fmt.Println()
	}
}

// Читаем из файла fName.
func readTags(fName string) (tags Tags) {
	tags, err := taglib.ReadTags(fName)
	if err != nil {
		log.Output(3, fmt.Sprintln("Ошибка чтения тэгов", err))
	}
	return
}

// Выведем tags с title.
// Добавим из tags в t
func (t *Tags) add(title string, tags Tags) {
	tags.fixKey()
	tags.fixVal()
	tags.print(4, title, true)
	for k, vals := range *t {
		if v, ok := tags[k]; ok && len(v) == 0 {
			// tag=
			continue
		}
		tags[k] = append(tags[k], vals...)
	}
	tags.fixVal()
	*t = tags
}

// Выведем tags с title.
// Установим из tags в t
func (t *Tags) set(title string, tags Tags) {
	tags.fixKey()
	tags.fixVal()
	tags.print(3, title, true)
	for k, vals := range tags {
		t.setVals(k, vals...)
	}
	t.fixVal()
}

// Убираем дубликаты.
// Значения чувствительны к регистру.
// Значения не чувствительны к начальным и конечным пробелам.
func (t *Tags) fixVal() {
	tags := newTags()
	for k, vals := range *t {
		k := strings.ToLower(k)
		k = strings.TrimSpace(k)
		has := make(map[string]bool)
		uniqs := []string{}
		for _, val := range vals {
			val = strings.TrimSpace(val)
			if has[val] {
				continue
			}
			has[val] = true
			uniqs = append(uniqs, val)
		}
		tags[k] = uniqs
	}
	*t = tags
}

// Убираем пустышки.
func (t *Tags) delEmpty() {
	tags := newTags()
	for k, vals := range *t {
		if len(vals) == 0 {
			continue
		}
		tags[k] = vals
	}
	*t = tags
}

// Ключи не чувствительны к регистру.
// Ключи не чувствительны к начальным и конечным пробелам.
// tag =a Tag=b ~> tag=a/b
// tag= Tag=b ~> tag=
func (t *Tags) fixKey() {
	var block = map[string]bool{
		"encoding":          true,
		"encoder":           true,
		"compatible_brands": true,
		"minor_version":     true,
		"major_brand":       true,
		"creation_time":     true,
	}
	tags := newTags()
	for k, vals := range *t {
		k := strings.ToLower(k)
		k = strings.TrimSpace(k)
		if block[k] {
			continue
		}
		if v, ok := tags[k]; ok && len(v) == 0 {
			// tag=
			continue
		}
		tags[k] = append(tags[k], vals...)
	}
	*t = tags
}

// Из строк в tags.
// В строке может быть несколько тэгов как csv.
// Чтоб очистить tag=
func newTags(ss ...string) (tags Tags) {
	tags = make(Tags)
	for _, s := range ss {
		// kvs := strings.Split(s, ",")
		// "a=b,c",d=e
		kvs, _ := csv.NewReader(strings.NewReader(s)).Read()
		for _, kv := range kvs {
			kva := strings.Split(kv, "=")
			k := strings.ToLower(kva[0])
			k = strings.TrimSpace(k)
			if len(kva) > 1 {
				if len(kva) > 2 {
					log.Println("Пропустил", kva[2:])
				}
				if kva[1] == "" {
					tags[k] = nil
					continue
				}
				vals := strings.Split(kva[1], "/")
				for k, v := range vals {
					vals[k] = strings.TrimSpace(v)
				}
				tags[k] = append(tags[k], vals...)
			} else {
				log.Println("Пропустил", kva[0])
			}
		}
	}
	return
}

func (t *Tags) write(args1 string) {
	t.delEmpty()
	for _, key := range []string{"description", "comment"} {
		vals, ok := t.vals(key)
		if ok {
			t.setVals(key, strings.Join(vals, "\n"))
		}
	}
	t.setVals("encoder", "drTags")

	t.print(3, "Пишем тэги в "+args1, false)
	err := taglib.WriteTags(args1, *t, taglib.DiffBeforeWrite|taglib.Clear)
	if err != nil {
		log.Println("Ошибка записи тэгов", err)
	}
}

// Если есть в title мажор или минор а перед ними диез или бемоль а перед ними нота, то добавит initialkey в tags.
// Соната соль минор ~>Gm.
// Соната соль мажор ~>G.
// Соната соль-бемоль мажор ~>Gb.
// Соната соль бемоль мажор ~>Gb.
// Соната си-бемоль мажор ~>Bb.
// Соната си минор ~>Bm.
func (t *Tags) tKey(title string) {
	if _, ok := (*t)["initialkey"]; ok {
		return
	}
	fields := strings.Fields(strings.ToLower(title))
	minor := ""
	half := ""
	note := ""
loop:
	for i, mm := range fields {
		switch mm {
		case "минор":
			minor = "m"
			fallthrough
		case "мажор":
			before := i - 1
			if before < 0 {
				return
			}
			switch {
			case fields[before] == "диез":
				half = "#"
				before--
				if before < 0 {
					return
				}
				note = fields[before]
			case fields[before] == "бемоль":
				half = "b"
				before--
				if before < 0 {
					return
				}
				note = fields[before]
			case strings.HasSuffix(fields[before], "-дубль-диез"):
				half = "##"
				note = strings.TrimSuffix(fields[before], "-дубль-диез")
			case strings.HasSuffix(fields[before], "-дубль-бемоль"):
				half = "bb"
				note = strings.TrimSuffix(fields[before], "-дубль-бемоль")
			case strings.HasSuffix(fields[before], "-диез"):
				half = "#"
				note = strings.TrimSuffix(fields[before], "-диез")
			case strings.HasSuffix(fields[before], "-бемоль"):
				half = "b"
				note = strings.TrimSuffix(fields[before], "-бемоль")
			default:
				note = fields[before]
			}
			if note == "дубль" {
				before--
				if before < 0 {
					return
				}
				half += half
				note = fields[before]
			}
			break loop
		}
	}
	switch note {
	case "ля":
		note = "A"
	case "си":
		note = "B"
	case "до":
		note = "C"
	case "ре":
		note = "D"
	case "ми":
		note = "E"
	case "фа":
		note = "F"
	case "соль":
		note = "G"
	default:
		return
	}
	t.setVals("initialkey", note+half+minor)
}

func probeA(inFile string, asr bool) {
	f, err := os.Open(inFile)
	if err != nil {
		return
	}
	defer f.Close()

	p, err := taglib.ReadProperties(inFile)
	if err != nil {
		log.Println("a_bit_rate=?", err)
		return
	}
	if p.Bitrate > 0 {
		fmt.Printf("a_bit_rate=%d\r\n", p.Bitrate*1000)
	}
	if p.Length > 0 {
		fmt.Printf("a_duration=%s\r\n", p.Length)
	}
	if asr && p.SampleRate > 0 {
		fmt.Printf("a_sample_rate=%d\r\n", p.SampleRate)
	}
}

func (t *Tags) kvv(key string, val *string) {
	if ss, ok := t.vals(key); ok {
		*val = ""
		if len(ss) > 0 {
			*val = strings.Join(ss, "/")
		}
	} else {
		t.setVals(key, *val)
	}
}

func (t Tags) vals(key string) (ss []string, ok bool) {
	key = strings.ToLower(key)
	key = strings.TrimSpace(key)
	ss, ok = t[key]
	return
}

func (t *Tags) setVals(key string, vals ...string) {
	key = strings.ToLower(key)
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	vs := []string{}
	for _, val := range vals {
		va := strings.Split(val, "/")
		for _, v := range va {
			v = strings.TrimSpace(v)
			if v != "" {
				vs = append(vs, v)
			}
		}
	}
	(*t)[key] = vs
	t.fixVal()
}

// album/album tracknumber composer title
func (t *Tags) parse(album, file string) {
	t.kvv("album", &album)
	if _, ok := t.vals("date"); !ok {
		date := strings.Fields(album)[0]
		if _, err := strconv.Atoi(date); err == nil {
			if len(date) > 7 {
				t.setVals("date", date[:8])
			} else {
				t.setVals("date", date[:4])
			}
		}
	}
	title := strings.TrimPrefix(file, album)
	title = strings.TrimSpace(title)

	//01 Моцарт Соната для фортепиано 9 pе мажор
	tracknumber := strings.Fields(title)[0]
	if _, err := strconv.Atoi(tracknumber); err == nil {
		title = strings.TrimPrefix(title, tracknumber)
		title = strings.TrimSpace(title)
		// log.Println(title)
		//Моцарт Соната для фортепиано 9 pе мажор
		if _, ok := t.vals("tracknumber"); !ok {
			t.setVals("tracknumber", tracknumber)
		}
		composer := strings.Fields(title)[0]
		if composer != title {
			title = strings.TrimPrefix(title, composer)
			title = strings.TrimSpace(title)
			// log.Println(title)
			//Соната для фортепиано 9 pе мажор
			if _, ok := t.vals("composer"); !ok {
				// tags = addTags("", ssTags("composer="+composer), tags)
				t.setVals("composer", composer)
			}
		}
	}

	// kvv("title", &title, tags)
	t.kvv("title", &title)

	// tKey(title, t)
	t.tKey(title)
}

func (t *Tags) csv(file string, row *Row, keys ...string) {
	s := " файла "
	if filepath.Ext(file) == "" {
		s = " клипа "
	}
	s += file
	for _, key := range keys {
		// val := keyVal(key, header, r)
		val := row.val(key)
		if val == "" {
			continue
		}
		switch key {
		case "Description":
			val = strings.ReplaceAll(val, "\n", ",")
			fallthrough
		case "Keywords":
			if !strings.Contains(val, "=") {
				continue
			}
			// t.add("Тэги из "+key+s, newTags(val))
			t.add("", newTags(val))
		case "Comments":
			// log.Output(2, key+" из"+s)
			// fmt.Println("comment=" + val)
			t.setVals("comment", val)
		}
	}
}
