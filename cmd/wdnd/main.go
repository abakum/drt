// Всего 2 мегабайта
//go:build windows
// +build windows

//go:generate cmd /c go build -o %GOPATH%\bin\wdnd.exe -ldflags "-s -w -H=windowsgui"

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win/co"
)

func main() {
	runtime.LockOSThread() // Важно для однопоточного GUI Windows

	// Создаем и запускаем окно
	NewDragDropWindow().wnd.RunAsMain()
}

type DragDropWindow struct {
	wnd *ui.Main
}

func NewDragDropWindow() *DragDropWindow {
	wnd := ui.NewMain(
		ui.OptsMain().
			Title("dr&Tags").
			Size(ui.Dpi(100, 80)).
			Style(co.WS_SYSMENU | co.WS_CAPTION).             // Минимальный стиль окна
			ExStyle(co.WS_EX_ACCEPTFILES | co.WS_EX_TOPMOST). // Прием файлов + поверх всех окон
			ClassIconId(101),                                 // Иконка из ресурсов
	)

	ddw := &DragDropWindow{wnd: wnd}
	ddw.setupEventHandlers()
	return ddw
}

func (ddw *DragDropWindow) setupEventHandlers() {
	// Обработчик перетаскивания файлов
	ddw.wnd.On().WmDropFiles(func(p ui.WmDropFiles) {
		hDrop := p.HDrop()
		defer hDrop.DragFinish() // Важно: освобождаем ресурсы

		if files, err := hDrop.DragQueryFile(); err == nil {
			// Логируем файлы
			for i, file := range files {
				log.Printf("[%d] %s", i, file)
			}

			// Показываем уведомление (первые 10 файла)
			displayFiles := files
			if len(files) > 10 {
				displayFiles = files[:10]
			}
			ddw.wnd.Hwnd().MessageBox(
				strings.Join(displayFiles, "\n"),
				fmt.Sprintf("Принято %d файлов", len(files)),
				co.MB_ICONINFORMATION)
		} else {
			log.Printf("Ошибка получения файлов: %v", err)
		}
	})
}
