//go:build windows

package main

import (
	"log"
	"syscall"

	"fyne.io/fyne/v2"
)

func callSomeGlfwAPICGoWrapper(window fyne.Window) {}

// setWindowTopMostAndDisableMinimize
// Для дроплета важен маленький фиксированный размер, быть сверху и не минимизироваться
func callSomeWinAPICGoWrapper(hwnd uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getWindowLongPtr := user32.NewProc("GetWindowLongPtrW")
	setWindowLongPtr := user32.NewProc("SetWindowLongPtrW")
	setWindowPos := user32.NewProc("SetWindowPos")

	// Константы Windows API
	const (
		GWL_STYLE uintptr = 0xFFFFFFFFFFFFFFF0
		// GWL_EXSTYLE    uintptr = 0xFFFFFFFFFFFFFFEC
		// WS_EX_TOPMOST  uintptr = 0x00000008
		WS_MINIMIZEBOX uintptr = 0x00020000
		HWND_TOPMOST   uintptr = ^uintptr(0)
		SWP_NOSIZE     uintptr = 0x0001
		SWP_NOMOVE     uintptr = 0x0002
		// SWP_NOZORDER   uintptr = 0x0004
		// SWP_NOACTIVATE   uintptr = 0x0010
		// SWP_FRAMECHANGED uintptr = 0x0020
		// AfterSet         uintptr = SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED
		For0000 uintptr = SWP_NOMOVE | SWP_NOSIZE
	)

	// // 1. Основной способ - через SetWindowPos
	ret, _, err := setWindowPos.Call(
		hwnd,
		HWND_TOPMOST,
		0, 0, 0, 0,
		For0000,
	)
	if ret == 0 {
		log.Printf("SetWindowPos failed: %v\n", err)
	}

	// 2. Альтернативный способ не работает
	// exStyle, _, err := getWindowLongPtr.Call(
	// 	hwnd,
	// 	GWL_EXSTYLE,
	// )
	// if err != nil && err != syscall.Errno(0) {
	// 	log.Printf("Warning: GetWindowLong(GWL_EXSTYLE) may have failed: %v\n", err)
	// } else {
	// 	setWindowLongPtr.Call(
	// 		hwnd,
	// 		GWL_EXSTYLE,
	// 		exStyle|WS_EX_TOPMOST,
	// 	)
	// }

	// 3. Отключаем кнопку минимизации
	style, _, err := getWindowLongPtr.Call(
		hwnd,
		GWL_STYLE,
	)
	if err != nil && err != syscall.Errno(0) {
		log.Printf("Warning: getWindowLongPtr(GWL_STYLE) may have failed: %v\n", err)
	} else {
		setWindowLongPtr.Call(
			hwnd,
			GWL_STYLE,
			style&^WS_MINIMIZEBOX,
		)
		// Обновляем окно
		// setWindowPos.Call(
		// 	hwnd,
		// 	0,
		// 	0, 0, 0, 0,
		// 	For0000,
		// )
	}
}
