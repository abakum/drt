// Монстр 30 мегабайт
// sudo apt-get install gcc-mingw-w64-x86-64
//
// -go:generate bash -c "GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -tags=opengl -ldflags \"-s -w -H=windowsgui\""
//
//go:generate bash -c "GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -tags=opengl -ldflags \"-s -w\""

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver"
)

func main() {
	// Создаём приложение
	myApp := app.New()
	myWindow := myApp.NewWindow("dr&Tags")
	myWindow.Resize(fyne.NewSize(100, 80))
	myWindow.SetFixedSize(true) // Запрещаем изменение размера

	// Обработка перетаскивания файлов
	myWindow.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}

		var fileList []string
		for _, uri := range uris {
			filePath := uri.Path()
			fileList = append(fileList, filePath)
		}

		fmt.Println("Перетащены файлы:\n" + strings.Join(fileList, "\n"))
	})

	myWindow.Show()
	callSome := func() {
		// we will not have a window pointer until it is shown,
		// so use lifecycle to wait until then
		nativeWin, ok := myWindow.(driver.NativeWindow)
		if !ok {
			panic("this will never happen for a top-level window")
		}
		nativeWin.RunNative(func(ctx any) {
			switch runtime.GOOS {
			case "windows":
				hwnd := ctx.(driver.WindowsWindowContext).HWND
				log.Printf("HWND is %x\n", hwnd)
				callSomeWinAPICGoWrapper(hwnd)
			case "darwin":
				nsWindow := ctx.(driver.MacWindowContext).NSWindow
				log.Printf("NSWindow ptr is %x", nsWindow)
				//callSomeCocoaAPICGoWrapper(nsWindow)
			case "linux":
				if wayland, ok := ctx.(driver.WaylandWindowContext); ok {
					wlSurface := wayland.WaylandSurface
					log.Printf("Wayland window ptr is %x", wlSurface)
					// callSomeWaylandAPICGoWrapper(wlSurface)
				} else {
					x11Window := ctx.(driver.X11WindowContext).WindowHandle
					log.Printf("X11 window ptr is %x", x11Window)
					//callSomeX11APICGoWrapper(x11Window)
				}
				callSomeGlfwAPICGoWrapper(myWindow)
			}
		})
	}
	callSome()
	// myApp.Lifecycle().SetOnEnteredForeground(callSome())
	myWindow.ShowAndRun()
}
