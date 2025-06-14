package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abakum/go-taglib"
	"github.com/bogem/id3v2/v2"
)

const (
	HT = "HASHTAGS"
	EC = "ENCODER"
	DS = "DESCRIPTION"
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
		if k == HT {
			continue
		}
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

func dateDash(t *Tags, dash bool) {
	dates, _ := t.vals(taglib.Date)
	d := ""
	for _, date := range dates {
		if len(date) > len(d) {
			d = date
		}
	}
	if d == "" {
		return
	}
	d = strings.ReplaceAll(d, "-", "")
	if dash {
		if len(d) > 7 {
			t.setVals(taglib.Date, d[:4]+"-"+d[4:6]+"-"+d[6:8])
		}
		return
	}
	t.setVals(taglib.Date, d)
}

// Читаем из файла file.
func readTags(file string) (tags Tags) {
	tags, err := taglib.ReadTags(file)

	dateDash(&tags, false)

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
					if len(kva) > 2 && strings.TrimSpace(kva[2]) == "" {
						// ffmpeg
						tags = Tags{"==": nil}
						continue
					}
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

//	func deleteFirstNotLetter(s string) string {
//		re := regexp.MustCompile(`^[^\p{Letter}]*`)
//		return re.ReplaceAllString(s, "")
//	}
func removeFirstNonLatinCyrillic(s string) string {
	re := regexp.MustCompile(`^[^\p{Latin}\p{Cyrillic}]*`)
	return re.ReplaceAllString(s, "")
}
func removeNonAlphaNumSp(s string) string {
	re := regexp.MustCompile(`[^\p{Latin}\p{Cyrillic}\p{N} ]`)
	return re.ReplaceAllString(s, "")
}

func (t *Tags) write(file string) {
	t.delEmpty()
	uniq := make(map[string]bool) // отбросим повторения хэштэгов
	if LL(file) {
		uniq["#LL"] = true
	}
	for key, vs := range *t {
		switch key {
		case HT, EC, DS, taglib.Comment:
			continue
		}
		for _, v := range vs {
			name := removeFirstNonLatinCyrillic(v)

			if name == "" {
				// Числа пригодятся
				switch key {
				case taglib.MovementName:
					if len(v) < 2 {
						uniq["#m0"+v] = true
					} else {
						uniq["#m"+v[:2]] = true
					}
				case taglib.TrackNumber:
					if len(v) < 2 {
						uniq["#t0"+v] = true
					} else {
						uniq["#t"+v[:2]] = true
					}
				case taglib.Date:
					if len(v) > 3 {
						uniq["#y"+v[:4]] = true
					}
					if len(v) > 7 {
						uniq["#d"+v[:8]] = true
					}
				}
				continue
			}
			if key == taglib.InitialKey {
				name = strings.Replace(name, "#", "d", -1) // C#m~>Cdm
			}
			name = removeNonAlphaNumSp(name) // №1~>1
			if len(name) > 0 {
				name = "#" + strings.Replace(name, " ", "_", -1)
				uniq[name] = true
			}
		}
	}

	hashTags := Keys(uniq)

	slices.Sort(hashTags) // упорядочим хэштэги

	if len(hashTags) > 0 {
		t.setVals(HT, strings.Join(hashTags, " "))
	}

	vals, comment := t.vals(taglib.Comment)
	mp3 := Ext(file) == dotMP3
	if comment {
		// \n
		t.setVals(taglib.Comment, vals...)
		if mp3 {
			// Пишет COMMENT и Comment:XXX
			delete(*t, taglib.Comment)
		}
	}
	dateDash(t, true)

	if _, ok := t.vals(EC); ok {
		t.setVals(EC, "drTags")
	}

	err := taglib.WriteTags(file, *t, taglib.DiffBeforeWrite|taglib.Clear)
	if err != nil {
		log.Println("Ошибка записи тэгов", err)
	}

	if !mp3 {
		return
	}
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		log.Fatal("Ошибка чтения тэгов id3v2", err)
	}
	defer tag.Close()
	if len(vals) > 0 {
		comm := id3v2.CommentFrame{
			Encoding: id3v2.EncodingUTF8,
			Language: "XXX",
			Text:     strings.Join(vals, "\n"),
		}
		tag.AddCommentFrame(comm)
	}

	if err = tag.Save(); err != nil {
		log.Fatal("Ошибка записи тэгов id3v2", err)
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

// Ищем в title (val) или `val` или «val» и добавляем их в Subtitle=
func (t *Tags) tSub(title string) {
	// a `b` (c `d`) (e «f») «g»
	vals := parseVals(title, "(", ")")
	// "c `d`" "e «f»"
	vals = append(vals, parseVals(title, "`", "`")...)
	// "b" "d"
	vals = append(vals, parseVals(title, "«", "»")...)
	if len(vals) > 0 {
		t.setVals(taglib.Subtitle, vals...)
	}
	// "f" "g"
}

func parseVal(input, dL, dR string) (val, next string) {
	start := strings.Index(input, dL)
	if start == -1 {
		return
	}
	end := -1
	if len(input) > start+1 {
		end = start + 1 + strings.Index(input[start+1:], dR)
	}

	if end == -1 || end < start {
		return
	}

	l := len(dL)
	val = input[start+l : end]
	if len(input) > end+l {
		next = input[end+l:]
	}
	return
}

func parseVals(input, dL, dR string) (vals []string) {
	for {
		input = strings.TrimSpace(input)
		if input == "" {
			return
		}
		val, next := parseVal(input, dL, dR)
		if val != "" {
			vals = append(vals, val)
		}
		input = next
	}
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
		"фортепианным": "Фортепиано",
		"фортепианное": "Фортепиано",
		// Струнные смычковые: Скрипка, альт, виолончель, контрабас
		"альта":       "Альт",
		"альтов":      "Альт",
		"виолончели":  "Виолончель",
		"виолончелей": "Виолончель",
		"контрабаса":  "Контрабас",
		"контрабасов": "Контрабас",
		"скрипки":     "Скрипка",
		"скрипок":     "Скрипка",
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
		// Вокал
		"голоса":  "Вокал",
		"голосов": "Вокал",
		"баса":    "Вокал",
		"басов":   "Вокал",
		"тенора":  "Вокал",
		"теноров": "Вокал",
		"хора":    "Вокал",
		"стихи":   "Вокал",
		"слова":   "Вокал",
		"капелла": "Вокал",
		// Ансамбли
		"орестром": "Оркестр",
		"трио":     "Трио",
		"квартет":  "Квартет",
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
	if i.IsDir() {
		f.Close()
		return nil, errors.New("isDir")
	}
	if i.Size() == 0 {
		f.Close()
		return nil, errors.New("isEmpty")
	}
	return f, nil
}

func probeA(inFile string, asr bool) (lines []string) {
	switch Ext(inFile) {
	case dotMP3, dotFLAC, dotMOV, dotMP4, ".3gp", ".m4a":
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
		return
	}
	if p.Bitrate > 0 {
		lines = append(lines, fmt.Sprintf("a_bit_rate=%d kb/s", p.Bitrate))
	}
	if p.Length > 0 {
		lines = append(lines, fmt.Sprint("a_duration=", time.Duration(p.Length)))
	}
	if asr && p.SampleRate > 0 {
		lines = append(lines, fmt.Sprintf("a_sample_rate=%d", p.SampleRate))
	}
	return
}

// Если ключа key нет устанавливаем его в tags.
// Если ключ key есть то присваиваем его val.
func (t *Tags) kvv(key string, val *string, tags *Tags) {
	if vals, ok := t.vals(key); ok {
		*val = ""
		if len(vals) > 0 {
			*val = strings.Join(vals, "/")
		}
	} else {
		tags.setVals(key, *val)
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
		val = strings.ReplaceAll(val, "\n", "/")
		va := strings.Split(val, "/")
		for _, v := range va {
			v = strings.TrimSpace(v)
			if v != "" || key == taglib.Comment {
				vs = append(vs, v)
			}
		}
	}
	// log.Println(key, vs)
	if key == taglib.Comment {
		(*t)[key] = []string{strings.Join(vs, "\n")}
		return
	}
	(*t)[key] = vs
	t.fixVal()
}

// album/album tracknumber composer title часть 1
// album/album tracknumber Имя_Фамилия title части 1 Медленно_и_печально
func (t *Tags) parse(album, file string) {
	tags := newTags()
	t.kvv(taglib.Album, &album, &tags)
	if _, ok := t.vals(taglib.Date); !ok {
		Fields := strings.Fields(album)
		if len(Fields) > 0 {
			date := Fields[0]
			if _, err := strconv.Atoi(date); err == nil {
				if len(date) > 7 {
					tags.setVals(taglib.Date, date[:8])
				} else {
					tags.setVals(taglib.Date, date[:4])
				}
			}
		}
	}
	titleSort, _ := t.vals(taglib.TitleSort)
	ts := "Тэги из имени файла="
	if len(titleSort) > 0 {
		ts = "Тэги из TitleSort="
		file = titleSort[0]
	}
	ts += file

	title := strings.TrimPrefix(file, album)
	title = strings.TrimSpace(title)

	//01 Моцарт Соната для фортепиано 9 pе мажор
	Fields := strings.Fields(title)
	if len(Fields) > 0 {
		tracknumber := Fields[0]
		if _, err := strconv.Atoi(tracknumber); err == nil {
			title = strings.TrimPrefix(title, tracknumber)
			title = strings.TrimSpace(title)
			//Моцарт Соната для фортепиано 9 pе мажор
			if _, ok := t.vals(taglib.TrackNumber); !ok {
				tags.setVals(taglib.TrackNumber, tracknumber)
			}
			Fields := strings.Fields(title)
			if len(Fields) > 0 {
				composer := Fields[0]
				if composer != title {
					title = strings.TrimPrefix(title, composer)
					title = strings.TrimSpace(title)
					//Соната для фортепиано 9 pе мажор
					if _, ok := t.vals(taglib.Composer); !ok {
						tags.setVals(taglib.Composer, strings.ReplaceAll(composer, "_", " "))
					}
				}
			}
		}
	}

	if _, ok := t.vals(taglib.InitialKey); !ok {
		tags.tKey(title)
	}
	tags.tGroup(title)
	tags.tSub(title)
	title = tags.tMovement(title)
	t.kvv(taglib.Title, &title, &tags)
	t.add(ts, tags)
}

func LL(file string) (ok bool) {
	att := sources[file]
	if att == nil || att.audio == "" {
		switch Ext(file) {
		case dotFLAC, ".wav", ".ape", ".wv", ".dsf", ".dff", ".tak", ".tta", ".ofr":
			return true
		}
		return
	}
	switch att.audio {
	case "flac", "alac", "ape", "wavpack", "dsd_lsbf":
		return true
	case "aac", "mp3":
		return
	default:
		if strings.HasPrefix(att.audio, "pcm_") {
			return true
		}
	}
	return
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
