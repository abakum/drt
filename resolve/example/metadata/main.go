// go:build ignore
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	dr "github.com/abakum/drt/resolve"
)

func Resolve() *dr.Resolve {
	return &dr.Resolve{}
}

// Переведи на lua объяви resolve глобально:
var (
	resolve        = Resolve()
	projectManager = resolve.GetProjectManager()
	project        = projectManager.GetCurrentProject()
	mediaPool      = project.GetMediaPool()
	rootFolder     = mediaPool.GetRootFolder()
	clips          = rootFolder.GetClipList()
	exported       int
	outputFolder   string
)

func rootTimeLines() (root string, timeLines []*dr.MediaPoolItem) {
	for _, clip := range clips {
		if filePath := clip.GetClipProperty("File Path"); filePath == "" {
			// Это таймлайн
			timeLines = append(timeLines, clip)
			continue
		} else {
			// Допустим исходные и результирующие медиафайлы в одном каталоге
			root = filepath.Dir(filePath)
		}
	}
	return
}

func sanitizeFilename(name string) string {
	if name == "" {
		return "frame"
	}

	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.Join(strings.Fields(name), " ")
	return strings.TrimSpace(name)
}

func main() {
	log.Println("=== Экспорт метаданных таймлайнов ===")

	total := project.GetTimelineCount()
	if total == 0 {
		log.Println("Нет таймлайнов")
		return
	}

	root, timeLines := rootTimeLines()
	if root == "" {
		log.Println("Пустой медиапул")
		return
	}

	log.Println("Экспорт в ", root)
	tlm := make(map[string]map[string]*dr.MediaPoolItem) // клипы в таймлайне
	for i := 1; i <= total; i++ {
		tl := project.GetTimelineByIndex(i)
		tln := tl.GetName()
		log.Println(i, tln)
		for _, trackType := range []string{"video", "audio"} {
			tc := tl.GetTrackCount(trackType)
			for j := 1; j <= tc; j++ {
				tlis := tl.GetItemListInTrack(trackType, j)
				for _, tli := range tlis {
					mpi := tli.GetMediaPoolItem()
					mi := mpi.GetMediaId()
					tlm[tln][mi] = mpi
				}
			}
		}
	}

	for _, timeLine := range timeLines {
		tln := timeLine.GetName()
		name := sanitizeFilename(tln)
		file := filepath.Join(root, name) + ".csv"
		log.Println("Экспорт в ", file)

		var clips []*dr.MediaPoolItem
		clips = append(clips, timeLine)
		for _, mpi := range tlm[tln] {
			clips = append(clips, mpi)
			log.Println(mpi.GetName())
		}
		if mediaPool.ExportMetadata(file, clips...) {
			exported++
		} else {
			log.Println("Ошибка экспорта ", file)
		}
	}
	log.Println(fmt.Sprintf("=== Экспортировано %d из %d ===", exported, total))
}
