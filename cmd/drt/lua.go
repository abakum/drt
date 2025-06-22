// go:build ignore
package main

import (
	"log"
	"path/filepath"

	dr "github.com/abakum/drt/resolve"
)

func Resolve() *dr.Resolve {
	return &dr.Resolve{}
}

// Дипсик, переведи на lua:
var (
	resolve        = Resolve()
	projectManager = resolve.GetProjectManager()
	project        = projectManager.GetCurrentProject()
	mediaPool      = project.GetMediaPool()
	rootFolder     = mediaPool.GetRootFolder()
	clips          = rootFolder.GetClipList()
)

func rootTimeLines() (root string, timeLines []*dr.MediaPoolItem) {
	for _, clip := range clips {
		if filePath := clip.GetClipProperty("File Path"); filePath == "" {
			// Это таймлайн
			timeLines = append(timeLines, clip)
			continue
		} else {
			// В моих проектах все клипы в одном каталоге
			root = filepath.Dir(filePath)
		}
	}
	return
}

func main() {
	// csv
	root, timeLines := rootTimeLines()
	log.Println(root, timeLines)
	tlm := make(map[string][]*dr.MediaPoolItem) // клипы в тайлайне
	tlc := project.GetTimelineCount()
	for i := 1; i <= tlc; i++ {
		tl := project.GetTimelineByIndex(i)
		for _, trackType := range []string{"subtitle", "video", "audio"} {
			tc := tl.GetTrackCount(trackType)
			for j := 1; j <= tc; j++ {
				tlis := tl.GetItemListInTrack(trackType, j)
				for _, tli := range tlis {
					tln := tl.GetName()
					log.Println(i, trackType, j, tln, tli.GetName())
					tlm[tln] = append(tlm[tln], tli.GetMediaPoolItem())
				}
			}
		}
	}
	for _, timeLine := range timeLines {
		name := timeLine.GetName()
		file := filepath.Join(root, name)
		tln := timeLine.GetName()
		log.Println(file, tlm[tln])
		log.Println(tlm[tln])
		mediaPool.ExportMetadata(file+".csv", append(tlm[tln], timeLine)...)
	}
}
