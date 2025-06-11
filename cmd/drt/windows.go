//go:build windows

package main

import (
	"fmt"
	"log"
	"mime"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/jxeng/shortcut"
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	met = map[string]string{
		".csv":  "text/csv",
		".mov":  "video/quicktime",
		".mp4":  "video/mp4",
		".flac": "audio/x-flac",
		".mp3":  "audio/mpeg",
	}
)

func amAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func ShellExecute(verb, file, cwd string, showCmd int32, args ...string) (err error) {
	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	filePtr, _ := syscall.UTF16PtrFromString(file)
	argPtr, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)

	err = windows.ShellExecute(0, verbPtr, filePtr, argPtr, cwdPtr, showCmd)
	return
}

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
			return
		}
		log.Printf(`Error creating %s link: %v
`, m, err)
		if !amAdmin() {
			wd, _ := os.Getwd()
			err = ShellExecute("runas", "cmd", wd, 1, "/c", fmt.Sprintf(`mklink %s "%s" "%s"`, opt, newname, oldname))
			if err == nil {
				return
			}
			log.Println("Error run mklink as Administrator:", err)
		}
		return
	}
	name := trimExt(newname)
	err = os.WriteFile(name+".cmd", []byte(oldname+" %*"), 0744)
	if err != nil {
		log.Println("Error write .cmd:", err)
	}
	return
}

func install_(oldname string, lnks ...string) {
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
			WindowStyle: "3",
			// WorkingDirectory: "",
		}
		for _, lnk := range lnks {
			sc.ShortcutPath = lnk
			log.Println(oldname, "~>", sc.ShortcutPath, shortcut.Create(sc))
		}
	}
	reg, err := os.CreateTemp("", drt+"*.reg")
	log.Println(reg.Name(), err)
	if err != nil {
		return
	}
	defer os.Remove(reg.Name())
	defer reg.Close()
	var sb strings.Builder
	if oldname == "" {
		// uninstall
		sb.WriteString(head)
		appPaths(&sb, drt)
		applications(&sb, bin)
		progIDs(&sb, vendor, prog)
		registeredApplications(&sb, vendor, prog)
	} else {
		// install
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
