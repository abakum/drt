//go:build windows

package main

import (
	"fmt"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/adrg/xdg"
	"github.com/jxeng/shortcut"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win/co"
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	met = map[string]string{
		dotCSV:  "text/csv",
		dotMOV:  "video/quicktime",
		dotMP4:  "video/mp4",
		dotFLAC: "audio/x-flac",
		dotMP3:  "audio/mpeg",
	}
	DRS = []string{
		filepath.Join(xdg.DataDirs[0], "Blackmagic Design", "DaVinci Resolve", "Support"),
		filepath.Join(xdg.DataDirs[1], "Blackmagic Design", "DaVinci Resolve"),
	}
)

// Эта функция amAdmin() проверяет, запущена ли программа с правами администратора в Windows.
func amAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// Эта функция реализует вызов Windows API-функции ShellExecute,
// которая запускает файл или выполняет операцию (например, открытие/печать) с указанным файлом.
func ShellExecute(verb, file, cwd string, showCmd int32, args ...string) (err error) {
	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	filePtr, _ := syscall.UTF16PtrFromString(file)
	argPtr, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)

	err = windows.ShellExecute(0, verbPtr, filePtr, argPtr, cwdPtr, showCmd)
	return
}

// Эта функция mkLink реализует создание символических или жёстких ссылок на файлы в Windows,
// с учётом необходимости запуска с правами администратора и возможностью создания .cmd-файла как альтернативы.
func mkLink(oldname, newname string, link, hard bool) (err error) {
	if link {
		opt := ""
		osLink := os.Symlink
		m := "symbolic"
		if hard {
			osLink = os.Link
			opt = "/h"
			m = "hard"
		}
		err = osLink(oldname, newname)
		if err == nil {
			log.Println("mklink", opt, newname, oldname)
			return
		}
		if !amAdmin() {
			wd, _ := os.Getwd()
			err2 := ShellExecute("runas", "cmd", wd, 0, "/c", fmt.Sprintf(`mklink %s "%s" "%s"`, opt, newname, oldname))
			if err2 == nil {
				log.Println("mklink", opt, newname, oldname)
				return err2
			}
			log.Println("Error run mklink as Administrator:", err2)
		}
		log.Printf(`Error creating %s link: %v
`, m, err)
		return
	}
	name := trimExt(newname)
	err = os.WriteFile(name+".cmd", []byte(oldname+" %*"), 0755)
	if err != nil {
		log.Println("Error write .cmd:", err)
	}
	return
}

func install(oldname string, lnks ...string) {
	bin := drt
	prog := drTags
	vendor := "Abakum"
	FriendlyAppName := "Tagger for DaVinci Resolve"
	path := qq(exe)
	command := path + " " + qq("%1")
	head := `Windows Registry Editor Version 5.00

`
	if oldname == "" {
		// uninstall
		for _, lnk := range lnks {
			log.Println(lnk, "~> nul", os.Remove(lnk))
		}
	} else {
		sc := shortcut.Shortcut{
			// ShortcutPath:     "",
			Target:       oldname,
			IconLocation: oldname,
			// Arguments:        "",
			Description: FriendlyAppName,
			// Hotkey:           "",
			WindowStyle: "1",
			// WorkingDirectory: "",
		}
		for _, lnk := range lnks {
			sc.ShortcutPath = lnk
			log.Println(oldname, "~>", sc.ShortcutPath, shortcut.Create(sc))
		}
	}
	// Создаю меню Открыть с помощью
	reg, err := os.CreateTemp("", drt+"*.reg")
	log.Println(reg.Name(), err)
	if err != nil {
		return
	}
	defer os.Remove(reg.Name())
	defer reg.Close()
	var sb strings.Builder
	if oldname == "" {
		// Разрегистрируем
		sb.WriteString(head)
		appPaths(&sb, drt)
		applications(&sb, bin)
		progIDs(&sb, vendor, prog)
		registeredApplications(&sb, vendor, prog)
	} else {
		// Регистрируем
		// https://learn.microsoft.com/en-us/windows/win32/shell/app-registration
		sb.WriteString(head)
		appPaths(&sb, drt, path)
		applications(&sb, bin, prog, command, FriendlyAppName)
		progIDs(&sb, vendor, prog, command)
		registeredApplications(&sb, vendor, prog, FriendlyAppName)

		TypeByExtension(&sb)
	}
	// regS := strings.Replace(sb.String(), qq(`C:\Users\user_\go\bin\drt.exe`), qq(exe), -1)

	fmt.Println(sb.String())

	// Пишем .reg с UTF-16 LE BOM
	encoding := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	encoder := encoding.NewEncoder()
	writer := transform.NewWriter(reg, encoder)
	writer.Write([]byte(sb.String()))
	writer.Close()
	reg.Close()

	// wd, _ := os.Getwd()
	// Окошко
	// log.Println(ShellExecute("open", reg.Name(), wd, 1))

	// Требует админа
	// cmd := exec.CommandContext(ctx, "regedit", "/s", reg.Name())
	cmd := exec.CommandContext(ctx, "cmd", "/c", "regedit /s "+reg.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	log.Println(cmd, err)
	if err != nil {
		return
	}
	cmd.Wait()
	NotifySystemOfNewRegistration()
}

func sl3(o ...string) (s0, s1, s2 string) {
	if len(o) > 0 {
		s0 = o[0]
	}
	if len(o) > 1 {
		s1 = o[1]
	}
	if len(o) > 2 {
		s2 = o[2]
	}
	return
}

func applications(sb *strings.Builder, bin string, prog_command_FriendlyAppName ...string) {
	if len(prog_command_FriendlyAppName) < 3 {
		sb.WriteString(`[-HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\` + bin + `.exe]
`)
		return
	}
	prog, command, FriendlyAppName := sl3(prog_command_FriendlyAppName...)
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\` + bin + `.exe]
@="` + prog + `"
"FriendlyAppName"="` + FriendlyAppName + `"
`)
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\` + bin + `.exe\SupportedTypes]
`)
	for _, e := range Keys(met) {
		sb.WriteString(fmt.Sprintf(`"%s"=""
`, e))
	}
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\` + bin + `.exe\shell\open\command]
@="` + command + `"

`)
}

// progIDs(&sb,"Abakum.drTags", qq(exe))
func progIDs(sb *strings.Builder, vendor, prog string, ShellOpenCommand ...string) {
	soc, _, _ := sl3(ShellOpenCommand...)
	for _, e := range Keys(met) {
		if len(ShellOpenCommand) > 0 {
			sb.WriteString(fmt.Sprintf(`[HKEY_CURRENT_USER\SOFTWARE\Classes\%s.%s%s\Shell\Open\Command]
`, vendor, prog, e))
			sb.WriteString(`@="` + soc + `"
`)
		} else {
			sb.WriteString(fmt.Sprintf(`[-HKEY_CURRENT_USER\SOFTWARE\Classes\%s.%s%s]
`, vendor, prog, e))
		}
	}
}

// appPaths(&sb,"drt", qq(exe))
func appPaths(sb *strings.Builder, bin string, path ...string) {
	if len(path) < 1 {
		sb.WriteString(`[-HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\` + bin + `.exe]
`)
		return
	}
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\` + bin + `.exe]
`)
	sb.WriteString(`@="` + path[0] + `"

`)
}

func registeredApplications(sb *strings.Builder, vendor, prog string, description ...string) {
	if len(description) < 1 {
		sb.WriteString(fmt.Sprintf(`
[-HKEY_CURRENT_USER\SOFTWARE\%s\%s]
[HKEY_CURRENT_USER\SOFTWARE\RegisteredApplications]
"%s"=-

`, vendor, prog, prog))
		return
	}
	sb.WriteString(fmt.Sprintf(`
[HKEY_CURRENT_USER\SOFTWARE\RegisteredApplications]
"%s"="SOFTWARE\\%s\\%s\\Capabilities"
`, prog, vendor, prog))

	d, _, _ := sl3(description...)
	sb.WriteString(fmt.Sprintf(`
[HKEY_CURRENT_USER\SOFTWARE\%s\%s\Capabilities]
"ApplicationName"="%s"
"ApplicationDescription"="%s"
`, vendor, prog, prog, d))
	sb.WriteString(fmt.Sprintf(`[HKEY_CURRENT_USER\SOFTWARE\%s\%s\Capabilities\FileAssociations]
`, vendor, prog))
	for _, e := range Keys(met) {
		sb.WriteString(fmt.Sprintf(`"%s"="%s.%s%s"
`, e, vendor, prog, e))
	}
}

func TypeByExtension(sb *strings.Builder) {
	for e := range met {
		t := met[e]
		tbe := mime.TypeByExtension(e)
		if tbe == "" {
			// Создадим с правильным flac
			// https://mimetype.io/audio/x-flac#:~:text=audio/x%2Dflac%20%2D%20mimetype,when%20it's%20known%20as%20OggFLAC).
			sb.WriteString(fmt.Sprintf(`[HKEY_CURRENT_USER\SOFTWARE\Classes\%s]
`, e))
			sb.WriteString(fmt.Sprintf(`@="Abakum.drTags%s"
`, e))
			tTrue := strings.Replace(t, "/x-", "/", 1)
			sb.WriteString(fmt.Sprintf(`"Content Type"="%s"
`, tTrue))
			sb.WriteString(fmt.Sprintf(`"PerceivedType"="%s"
`, strings.Split(t, "/")[0]))
			met[e] = tTrue
		} else {
			met[e] = tbe
		}
	}
}

// Эта функция уведомляет операционную систему Windows о смене ассоциаций файлов или программ,
// чтобы обновить кэшированные данные и визуальные элементы системы.
func NotifySystemOfNewRegistration() {
	// https://learn.microsoft.com/en-us/windows/win32/shell/default-programs
	const (
		SHCNE_ASSOCCHANGED = 0x08000000
		SHCNF_DWORD        = 0x0003
		SHCNF_FLUSH        = 0x1000
		nullptr            = 0
	)
	windows.NewLazyDLL("shell32.dll").NewProc("SHChangeNotify").Call(
		SHCNE_ASSOCCHANGED,
		SHCNF_DWORD|SHCNF_FLUSH,
		nullptr, nullptr)
	time.Sleep(1000)
}

func qq(exe string) (path string) {
	// a\b
	path = fmt.Sprintf("%q", exe)
	// "a\\b"
	path = strings.Trim(path, `"`)
	// a\\b
	path = `\"` + path + `\"`
	// \"a\\b\"
	return
}

// Настраивает терминал Windows для поддержки ANSI-экранирования (цвета, стили текста) через виртуальный терминал (Virtual Terminal Processing).
// В Windows 10, 11 и 2004 (Build 22000 и выше) поддерживается только для консоли, а не для всех приложений.
// В Windows 7 и 8, поддерживается только для консоли.
// В Windows 8.1 и 10, поддерживается только для консоли и приложения, запущенных с помощью командной строки.
// В Windows 10, поддерживается только для консоли и приложений, запущенных с помощью командной строки.
// В Windows 11, поддерживается только для консоли и приложений, запущенных с помощью командной строки.
// В Windows 2004 (Build 22000) и выше, поддерживается только для консоли и приложений, запущенных с помощью командной строки.
func evtp() {
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32
	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}

func SplitCommandLine(command string) ([]string, error) {
	return windows.DecomposeCommandLine(command)
}
func onMain() {

}
func showDroplet(title string) {
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
			ClassIconId(1). // Иконка из ресурсов
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
