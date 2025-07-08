//go:build !windows

package main

import (
	"reflect"
	"unsafe"

	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func callSomeWinAPICGoWrapper(hwnd uintptr) {}

func callSomeGlfwAPICGoWrapper(window fyne.Window) {
	if w := getGlfwWindow(window); w != nil {
		w.SetAttrib(glfw.Floating, glfw.True)
	}
}

func getGlfwWindow(window fyne.Window) *glfw.Window {
	rv := reflect.ValueOf(window)
	if rv.Type().String() != "*glfw.window" {
		return nil
	}
	rv = rv.Elem()
	var glfwWindowPtr uintptr = rv.FieldByName("viewport").Pointer()
	// uncomment following code to wait until window is displayed
	// for glfwWindowPtr == 0 {
	// 	 glfwWindowPtr = rv.FieldByName("viewport").Pointer()
	// }
	return (*glfw.Window)(unsafe.Pointer(glfwWindowPtr))
}
