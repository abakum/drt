// go:build ignore
package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	dr "github.com/abakum/drt/resolve"
)

func Resolve() *dr.Resolve {
	return &dr.Resolve{}
}

// Переведи на lua объяви resolve глобально:

var (
	resolve         = Resolve()
	projectManager  = resolve.GetProjectManager()
	project         = projectManager.GetCurrentProject()
	mediaPool       = project.GetMediaPool()
	rootFolder      = mediaPool.GetRootFolder()
	clips           = rootFolder.GetClipList()
	jobs            []string
	outputFolder    string
	frameRate       float64
	exported        int
	renderJobsAdded bool
)

const (
	FORMAT  = "png"
	CODEC   = "RGB8"
	FORMATb = "dpx"
	CODECb  = "RGB10"
)

func timeToTimecode(seconds, fps float64) string {
	return fmt.Sprintf("%02d:%02d:%02d:%02d",
		int(seconds/3600),
		int(math.Mod(seconds, 3600)/60),
		int(math.Mod(seconds, 60)),
		int(math.Mod(seconds, 1)*fps),
	)
}

func timecodeToFrames(timecode interface{}, fps float64) int {
	switch v := timecode.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		parts := strings.Split(v, ":")
		if len(parts) != 4 {
			return 0
		}
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		s, _ := strconv.Atoi(parts[2])
		f, _ := strconv.Atoi(parts[3])
		return f + s*int(fps) + m*60*int(fps) + h*3600*int(fps)
	default:
		return 0
	}
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

func exportFrameAsStill(pos int, outputPath string) bool {
	// Попробуем новый метод
	if project.ExportCurrentFrameAsStill(outputPath) {
		fmt.Println("Со вкладки Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
		return true
	}

	// План B через Deliver
	timeline := project.GetCurrentTimeline()
	startFrame := timeline.GetStartFrame()

	renderSettings := map[string]interface{}{
		"MarkIn":      startFrame + pos,
		"MarkOut":     startFrame + pos,
		"CustomName":  strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath)),
		"TargetDir":   outputFolder,
		"ExportVideo": false,
		"ExportAudio": false,
	}

	if !project.SetRenderSettings(renderSettings) {
		fmt.Println("Не удалось установить настройки рендера")
		return false
	}

	if !project.SetCurrentRenderFormatAndCodec(FORMAT, CODEC) {
		if !project.SetCurrentRenderFormatAndCodec(FORMATb, CODECb) {
			fmt.Printf("Не удалось установить формат %s и кодек %s\n", FORMATb, CODECb)
			return false
		}
	}

	jobId := project.AddRenderJob()
	jobs = append(jobs, jobId)
	return true
}

func exportMarkedFrames() {
	timeline := project.GetCurrentTimeline()

	markers := timeline.GetMarkers()
	if len(markers) == 0 {
		return
	}

	// Sort markers
	var positions []int
	for pos := range markers {
		positions = append(positions, pos)
	}
	sort.Ints(positions)

	// Base filename
	timelineName := sanitizeFilename(timeline.GetName())

	// Export frames
	for _, pos := range positions {
		marker := markers[pos]
		name := marker.Name
		if name == "Marker 1" {
			name = ""
		}

		timecode := timeToTimecode(float64(pos)/frameRate, frameRate)
		timeline.SetCurrentTimecode(timecode)

		filename := timelineName + name
		fullPath := filepath.Join(outputFolder, filename+"."+FORMAT)

		if exportFrameAsStill(pos, fullPath) {
			fmt.Printf("Экспортирован %s -> %s\n", timecode, filename)
			exported++
		} else {
			fmt.Println("Ошибка экспорта " + timecode)
		}
	}
}

func exportAllTimelines() {
	timelineCount := project.GetTimelineCount()

	// Save original timeline
	originalTimeline := project.GetCurrentTimeline()

	// Process all timelines
	for i := 1; i <= timelineCount; i++ {
		timeline := project.GetTimelineByIndex(i)
		if markers := timeline.GetMarkers(); len(markers) > 0 {
			fmt.Printf("\n--- Таймлайн: %s ---\n", timeline.GetName())
			project.SetCurrentTimeline(timeline)
			exportMarkedFrames()
		}
	}

	// Restore original timeline
	if timelineCount > 1 {
		project.SetCurrentTimeline(originalTimeline)
	}
}

func rootMarkers() (root string, total int) {
	for _, clip := range clips {
		if filePath := clip.GetClipProperty("File Path"); filePath == "" {
			// Это таймлайн
			total += len(clip.GetMarkers())
			continue
		} else {
			// Допустим исходные и результирующие медиафайлы в одном каталоге
			root = filepath.Dir(filePath)
		}
	}
	return
}

func main() {
	fmt.Println("=== Экспорт кадров с маркерами ===")

	if project == nil {
		log.Println("Проект не найден")
		return
	}
	outputFolder, total := rootMarkers()
	if outputFolder == "" {
		log.Println("Пустой медиапул")
		return
	}
	if total == 0 {
		log.Println("Нет маркеров на таймлайнах")
		return
	}

	fpsStr := project.GetSetting("timelineFrameRate")
	frameRate, _ = strconv.ParseFloat(fpsStr, 64)
	if frameRate == 0 {
		frameRate = 24
	}

	project.DeleteAllRenderJobs()
	exportAllTimelines()

	log.Println(fmt.Sprintf("=== Экспортировано %d из %d ===", exported, total))
	if renderJobsAdded {
		project.StartRendering()
	}
}
