// go:build ignore
package main

import (
	"log"
	"path/filepath"
	"regexp"

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

func sanitizeFilename(filename string) string {
	// Example: Remove common invalid Windows characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := invalidChars.ReplaceAllString(filename, "_") // Replace with underscore
	return sanitized
}

func main() {
	// csv
	root, timeLines := rootTimeLines()
	log.Println(root, timeLines)
	tlm := make(map[string]map[string]*dr.MediaPoolItem) // клипы в тайлайне
	tlc := project.GetTimelineCount()
	for i := 1; i <= tlc; i++ {
		tl := project.GetTimelineByIndex(i)
		for _, trackType := range []string{"subtitle", "video", "audio"} {
			tc := tl.GetTrackCount(trackType)
			for j := 1; j <= tc; j++ {
				tlis := tl.GetItemListInTrack(trackType, j)
				for _, tli := range tlis {
					tln := tl.GetName()
					mpi := tli.GetMediaPoolItem()
					mi := mpi.GetMediaId()
					tlm[tln][mi] = mpi
				}
			}
		}
	}
	for _, timeLine := range timeLines {
		name := sanitizeFilename(timeLine.GetName())
		file := filepath.Join(root, name) + ".csv"
		var clips []*dr.MediaPoolItem
		clips = append(clips, timeLine)
		tln := timeLine.GetName()
		for mi, mpi := range tlm[tln] {
			clips = append(clips, mpi)
			log.Println(tln)
			log.Println(mi, " ", mpi.GetName())
		}
		mediaPool.ExportMetadata(file, clips...)
	}
}
