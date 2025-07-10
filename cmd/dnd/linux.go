//go:build linux

// Долго первый раз запрягает зато всего 5 мегабайт

// sudo apt install libgtk-3-dev
// go get github.com/gotk3/gotk3@master
// https://github.com/gotk3/gotk3/issues/343
package main

import (
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func showDroplet(title string) {
	// Обработчик DnD для окна
	handleDrop := func(
		win *gtk.Window,
		context *gdk.DragContext,
		x, y int,
		data *gtk.SelectionData,
		info, time uint,
	) {
		// Получаем данные (URI-список)
		uriList := string(data.GetData())

		paths := []string{}
		// Разбиваем строку на отдельные URI
		for _, uri := range strings.Split(uriList, "\r\n") {
			if uri == "" {
				continue
			}

			// Убираем "file://" и декодируем URI
			path, err := url.PathUnescape(strings.TrimPrefix(uri, "file://"))
			if err != nil {
				log.Printf("Ошибка декодирования URI: %v", err)
				continue
			}
			paths = append(paths, path)
			// Выводим чистый путь
			// fmt.Println("Принят файл:", path)
		}
		logPaths(strings.Join(paths, "\n"))
	}

	// Инициализация GTK
	gtk.Init(&os.Args)

	// Создание окна
	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal(err)
	}
	window.SetTitle(title)
	window.Connect("destroy", gtk.MainQuit)
	window.SetKeepAbove(true)
	window.SetSizeRequest(100, dY)
	window.SetResizable(false)
	target, err := gtk.TargetEntryNew("text/uri-list", gtk.TARGET_OTHER_APP, 0)
	if err != nil {
		log.Fatal(err)
	}
	targets := []gtk.TargetEntry{*target}

	window.DragDestSet(
		gtk.DEST_DEFAULT_ALL,
		targets,
		gdk.ACTION_COPY,
	)
	window.Connect("drag-data-received", handleDrop)
	applyOppositeTheme(window)
	window.ShowAll()
	gtk.Main()
}

func getSystemTheme() string {
	// Попробуем определить тему через настройки GTK
	settings, err := gtk.SettingsGetDefault()
	if err != nil {
		return "light" // По умолчанию
	}

	themeName, err := settings.GetProperty("gtk-theme-name")
	if err != nil {
		return "light"
	}

	// Проверяем популярные названия тёмных тем
	theme := strings.ToLower(themeName.(string))
	if strings.Contains(theme, "dark") || strings.Contains(theme, "black") {
		return "dark"
	}
	return "light"
}

func applyOppositeTheme(window *gtk.Window) {
	currentTheme := getSystemTheme()
	cssProvider, _ := gtk.CssProviderNew()

	var css string
	if currentTheme == "dark" {
		// Если система использует тёмную тему - ставим светлую
		css = `
			window, .window {
				background-color: #f5f5f5;
				color: #333333;
			}
		`
	} else {
		// Если система использует светлую тему - ставим тёмную
		css = `
			window, .window {
				background-color: #333333;
				color: #f5f5f5;
			}
		`
	}

	_ = cssProvider.LoadFromData(css)
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
