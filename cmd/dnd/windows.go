//go:build windows
// +build windows

// Всего 2 мегабайта

//go:generate cmd /c go build -o %GOPATH%\bin\wdnd.exe -ldflags "-s -w -H=windowsgui"

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win/co"
)

func onMain(title string) {
	runtime.LockOSThread() // Важно для однопоточного GUI Windows

	// Создаем и запускаем окно
	NewDragDropWindow(title).wnd.RunAsMain()
}

type DragDropWindow struct {
	wnd *ui.Main
}

func NewDragDropWindow(title string) *DragDropWindow {
	wnd := ui.NewMain(
		ui.OptsMain().
			// ClassIconId(101). // Иконка из ресурсов
			Title(title).
			Size(ui.Dpi(dX, dY)).
			Style(co.WS_SYSMENU | co.WS_CAPTION).             // Минимальный стиль окна
			ExStyle(co.WS_EX_ACCEPTFILES | co.WS_EX_TOPMOST), // Прием файлов + поверх всех окон
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
			logPaths(strings.Join(files, "\n"))
			// Показываем уведомление (первые 10 файла)
			// displayFiles := files
			// if len(files) > 10 {
			// 	displayFiles = files[:10]
			// }
			// ddw.wnd.Hwnd().MessageBox(
			// 	strings.Join(displayFiles, "\n"),
			// 	fmt.Sprintf("Принято %d файлов", len(files)),
			// 	co.MB_ICONINFORMATION)
		} else {
			log.Printf("Ошибка получения файлов: %v", err)
		}
	})
}

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
	procOpenProcess = kernel32.NewProc("OpenProcess")
	procCloseHandle = kernel32.NewProc("CloseHandle")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	ERROR_ALREADY_EXISTS              = 183
)

func createAppLock(appName string) (*os.File, error) {
	// Создаем именованный мьютекс
	mutexName, err := syscall.UTF16PtrFromString(fmt.Sprintf("Global\\%s", appName))
	if err != nil {
		return nil, fmt.Errorf("failed to create mutex name: %v", err)
	}

	handle, _, err := procCreateMutex.Call(
		0, // default security attributes
		0, // initially not owned
		uintptr(unsafe.Pointer(mutexName)),
	)

	if handle == 0 {
		return nil, fmt.Errorf("failed to create mutex: %v", err)
	}

	// Проверяем, не существует ли уже мьютекс
	if errno, ok := err.(syscall.Errno); ok && errno == ERROR_ALREADY_EXISTS {
		syscall.CloseHandle(syscall.Handle(handle))
		return nil, fmt.Errorf("application already running")
	}

	// Создаем временный файл для записи PID
	file, err := os.CreateTemp("", appName+"_*.lock")
	if err != nil {
		syscall.CloseHandle(syscall.Handle(handle))
		return nil, fmt.Errorf("failed to create lock file: %v", err)
	}

	_, err = file.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		syscall.CloseHandle(syscall.Handle(handle))
		file.Close()
		os.Remove(file.Name())
		return nil, fmt.Errorf("failed to write PID: %v", err)
	}

	file.Sync()
	return file, nil
}

func checkProcessExists(pid int) bool {
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_LIMITED_INFORMATION),
		uintptr(0),
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	procCloseHandle.Call(handle)
	return true
}

func cleanupLock(lockFile *os.File) {
	if lockFile != nil {
		lockFile.Close()
		os.Remove(lockFile.Name())
	}
}
