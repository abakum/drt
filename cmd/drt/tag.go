package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
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
	keys := []string{}
	for k := range t {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		vals := t[k]
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
	}
}

// Читаем из файла fName.
func readTags(fName string) (tags Tags) {
	tags, err := taglib.ReadTags(fName)
	if err != nil {
		log.Output(3, fmt.Sprintln("Ошибка чтения тэгов", err))
	}
	tags.fixKey()
	tags.fixVal()
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
	if _, ok := tags["="]; ok {
		// С чистого листа
		*t = newTags()
		delete(tags, "=")
	}
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
		k := strings.ToUpper(k)
		k = strings.TrimSpace(k)
		if k == taglib.Comment {
			// Комментарии не трогаем
			tags[k] = vals
			continue
		}
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
		"ENCODING":          true,
		"ENCODER":           true,
		"COMPATIBLE_BRANDS": true,
		"MINOR_VERSION":     true,
		"MAJOR_BRAND":       true,
		"CREATION_TIME":     true,
	}
	tags := newTags()
	for k, vals := range *t {
		k := strings.ToUpper(k)
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
	old := taglib.Comment
	for _, s := range ss {
		// kvs := strings.Split(s, ",")
		// "a=b,c",d=e
		kvs, _ := csv.NewReader(strings.NewReader(s)).Read()
		for _, kv := range kvs {
			kva := strings.Split(kv, "=")
			key := strings.ToUpper(kva[0])
			key = strings.TrimSpace(key)
			if len(kva) < 2 {
				key = old
				kva = []string{key, kva[0]}
			} else {
				if key == "" && strings.TrimSpace(kva[1]) == "" {
					// С чистого листа
					tags = Tags{"=": nil}
					continue
				}
				old = key
			}
			for _, val := range kva[1:] {
				// if val == "" {
				// 	tags[k] = nil
				// 	continue
				// }
				vals := strings.Split(val, "/")
				for k, v := range vals {
					vals[k] = strings.TrimSpace(v)
				}
				tags[key] = append(tags[key], vals...)
			}
		}
	}
	return
}

func (t *Tags) write(args1 string) {
	t.delEmpty()
	for _, key := range []string{"DESCRIPTION", taglib.Comment} {
		vals, ok := t.vals(key)
		if ok {
			t.setVals(key, strings.Join(vals, "\n"))
		}
	}
	if _, ok := t.vals("ENCODER"); ok {
		t.setVals("ENCODER", "drTags")
	}

	// t.print(3, "Пишу тэги в "+args1, false)
	err := taglib.WriteTags(args1, *t, taglib.DiffBeforeWrite|taglib.Clear)
	if err != nil {
		log.Println("Ошибка записи тэгов", err)
	}
}

// Убираем точки и запятые.
// Убираем в title хвост с частью.
// Если в title есть часть или части то добавим MovementName=.
// часть 1
// часть Медлено и печально
// части 1 2
// части Адажио Медлено_и_печально
func (t *Tags) tMovement(in string) (title string) {
	title = strings.ReplaceAll(in, ".", "")
	title = strings.ReplaceAll(title, ",", "")
	_, ok := t.vals(taglib.MovementName)

	fields := strings.Fields(title)
	found := false
	for i, field := range fields {
		field = strings.ToLower(field)
		if field != "часть" && field != "части" {
			continue
		}
		if len(fields) < i+2 {
			return
		}
		found = true
		if field == "часть" {
			if !ok {
				t.addVals(taglib.MovementName, strings.Join(fields[i+1:], " "))
			}
			break
		}

		for _, field := range fields[i+1:] {
			if !ok {
				t.addVals(taglib.MovementName, strings.ReplaceAll(field, "_", " "))
			}
		}
		break
	}
	if found {
		// Рубим хвост
		if tail := strings.Index(strings.ToLower(title), "част"); tail > 0 {
			title = strings.TrimSpace(title[:tail])
		}
	}
	return
}

// Если есть в title мажор или минор а перед ними диез или бемоль а перед ними нота, то добавит InitialKey в tags.
// Соната соль минор ~>Gm.
// Соната соль мажор ~>G.
// Соната соль-бемоль мажор ~>Gb.
// Соната соль бемоль мажор ~>Gb.
// Соната си-бемоль мажор ~>Bb.
// Соната си минор ~>Bm.
func (t *Tags) tKey(title string) {
	if _, ok := t.vals(taglib.InitialKey); ok {
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
	t.setVals(taglib.InitialKey, note+half+minor)
}

// Ищем в title инструмент и добавляем его в Grouping=
// Ищем в title № и всё, что до него добавляем в Grouping=
func (t *Tags) tGroup(title string) {
	// для скрипки и фортепиано РПЕЧ
	// для двух скрипок и фортепиано РПМЧ
	instruments := map[string]string{
		// Клавишные: фортепиано, орган, синтезатор
		"органа":       "Орган",
		"синтезатора":  "Синтезатор",
		"синтезаторов": "Синтезатор",
		"фортепиано":   "Фортепиано",
		// Струнные смычковые: Скрипка, альт, виолончель, контрабас
		"альта":       "Альт",
		"альтов":      "Альт",
		"виолончели":  "Виолончель",
		"виолончелей": "Виолончель",
		"контрабаса":  "Контрабас",
		"контрабасов": "Контрабас",
		"cкрипки":     "Скрипка",
		"cкрипок":     "Скрипка",
		// Струнные щипковые:  арфа, балалайка, домра, гитара, лютня, мандолина
		"арфы":      "Арфа",
		"арф":       "Арфа",
		"балалайки": "Балалайка",
		"балалаек":  "Балалайка",
		"гитары":    "Гитара",
		"гитар":     "Гитара",
		"домры":     "Домра",
		"домр":      "Домра",
		"лютни":     "Лютня",
		"лютней":    "Лютня",
		"лютен":     "Лютня",
		"мандолины": "Мандолина",
		"мандолин":  "Мандолина",
		// Деревянные духовые: Флейта
		// Язычковые духовые: кларнет, саксофон
		// Духовые с двойным язычком: гобой, фагот
		"гобоя":      "Гобой",
		"гобоев":     "Гобой",
		"кларнета":   "Кларнет",
		"кларнетов":  "Кларнет",
		"саксофона":  "Саксофон",
		"саксофонов": "Саксофон",
		"фагота":     "фагот",
		"фаготов":    "фагот",
		"флейты":     "Флейта",
		"флейт":      "Флейта",
		// Медные духовые: Труба, тромбон, валторна
		"валторны":  "Валторна",
		"валторн":   "Валторна",
		"тромбона":  "Тромбон",
		"тромбонов": "Тромбон",
		"трубы":     "Труба",
		"труб":      "Труба",
		// Язычковые Гармонь Аккордеон Баян
		"аккордеона":  "Аккордеон",
		"аккордеонов": "Аккордеон",
		"баяна":       "Баян",
		"баянов":      "Баян",
		"гармони":     "Гармонь",
		"гармоней":    "Гармонь",
	}
	fields := strings.Fields(strings.ToLower(title))
	for _, field := range fields {
		if v, ok := instruments[field]; ok {
			t.addVals(taglib.Grouping, v)
		}
	}
	ss := strings.Split(title, "№")
	if ss[0] != title {
		t.addVals(taglib.Grouping, strings.TrimSpace(ss[0]))
	}
}

func open(name string) (*os.File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if i.IsDir() || i.Size() == 0 {
		return nil, errors.New("not media file")
	}
	return f, nil
}

func probeA(inFile string, asr bool) {
	switch strings.ToLower(filepath.Ext(inFile)) {
	case ".mp3", ".flac", ".mov", "mp4", "m4a":
	default:
		return
	}
	f, err := open(inFile)
	if err != nil {
		return
	}
	defer f.Close()

	p, err := taglib.ReadProperties(f.Name())
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

// Если ключа key нет устанавливаем его в val.
// Если ключ key есть то присваиваем его val.
func (t *Tags) kvv(key string, val *string) {
	if vals, ok := t.vals(key); ok {
		*val = ""
		if len(vals) > 0 {
			*val = strings.Join(vals, "/")
		}
	} else {
		t.setVals(key, *val)
	}
}

func (t Tags) vals(key string) (ss []string, ok bool) {
	key = strings.ToUpper(key)
	key = strings.TrimSpace(key)
	ss, ok = t[key]
	return
}
func (t *Tags) addVals(key string, vals ...string) {
	tags := newTags()
	tags.setVals(key, vals...)
	t.add("", tags)
}

func (t *Tags) setVals(key string, vals ...string) {
	key = strings.ToUpper(key)
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	vs := []string{}
	for _, val := range vals {
		va := strings.Split(val, "/")
		for _, v := range va {
			v = strings.TrimSpace(v)
			if v != "" || key == taglib.Comment {
				vs = append(vs, v)
			}
		}
	}
	// log.Println(key, vs)
	(*t)[key] = vs
	t.fixVal()
}

// album/album tracknumber composer title часть 1
// album/album tracknumber Имя_Фамилия title части 1 Медленно_и_печально
func (t *Tags) parse(album, file string) {
	t.kvv(taglib.Album, &album)
	if _, ok := t.vals(taglib.Date); !ok {
		date := strings.Fields(album)[0]
		if _, err := strconv.Atoi(date); err == nil {
			if len(date) > 7 {
				t.setVals(taglib.Date, date[:4], date[:8])
			} else {
				t.setVals(taglib.Date, date[:4])
			}
		}
	}
	titleSort, _ := t.vals(taglib.TitleSort)
	if len(titleSort) > 0 {
		file = titleSort[0]
	}
	title := strings.TrimPrefix(file, album)
	title = strings.TrimSpace(title)

	//01 Моцарт Соната для фортепиано 9 pе мажор
	tracknumber := strings.Fields(title)[0]
	if _, err := strconv.Atoi(tracknumber); err == nil {
		title = strings.TrimPrefix(title, tracknumber)
		title = strings.TrimSpace(title)
		//Моцарт Соната для фортепиано 9 pе мажор
		if _, ok := t.vals(taglib.TrackNumber); !ok {
			t.setVals(taglib.TrackNumber, tracknumber)
		}
		composer := strings.Fields(title)[0]
		if composer != title {
			title = strings.TrimPrefix(title, composer)
			title = strings.TrimSpace(title)
			//Соната для фортепиано 9 pе мажор
			if _, ok := t.vals(taglib.Composer); !ok {
				t.setVals(taglib.Composer, strings.ReplaceAll(composer, "_", " "))
			}
		}
	}

	t.tKey(title)
	t.tGroup(title)
	title = t.tMovement(title)
	t.kvv(taglib.Title, &title)
}

func (t *Tags) csv(file string, row *Row, keys ...string) {
	s := " файла "
	if filepath.Ext(file) == "" {
		s = " клипа "
	}
	s += file
	for _, key := range keys {
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
			t.set("", newTags(val))
		case "Comments":
			t.setVals(taglib.Comment, val)
		}
	}
}
