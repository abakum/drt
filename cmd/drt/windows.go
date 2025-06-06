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
		log.Printf("Error creating %s link: %v\n", m, err)
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

func install(oldname string, lnks ...string) {
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
			Description: "Tagger for DaVinci Resolve",
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
	es := []string{".csv", ".mov", ".mp4", ".flac", ".mp3"}
	mes := map[string]string{
		".csv":  "text/csv",
		".mov":  "video/quicktime",
		".mp4":  "video/mp4",
		".flac": "audio/x-flac",
		".mp3":  "audio/mpeg",
	}
	var sb strings.Builder
	if oldname == "" {
		sb.WriteString(`Windows Registry Editor Version 5.00

[-HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\drt.exe]

[-HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\drt.exe]

`)
		removeIDs(&sb, es...)
		// [-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.csv]
		// [-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mov]
		// [-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mp4]
		// [-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.flac]
		// [-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mp3]
		sb.WriteString(`
[-HKEY_CURRENT_USER\SOFTWARE\Abakum\drTags]
[HKEY_CURRENT_USER\SOFTWARE\RegisteredApplications]
"drTags"=-
`)
	} else {
		// https://learn.microsoft.com/en-us/windows/win32/shell/app-registration
		sb.WriteString(`Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\drt.exe]
@="\"C:\\Users\\user_\\go\\bin\\drt.exe\""

[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\drt.exe]
@="drTags"
"FriendlyAppName"="Tagger for DaVinci Resolve"
`)
		// mknv(`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`+base, map[string]string{
		// 	"": `"` + exe + `"`,
		// })
		// mknv(`SOFTWARE\Classes\Applications\`+base, map[string]string{
		// 	"":                "drTags",
		// 	"FriendlyAppName": "Tagger for DaVinci Resolve",
		// })
		SupportedTypes(&sb, es...)
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\drt.exe\SupportedTypes]
		// ".csv"=""
		// ".mov"=""
		// ".mp4"=""
		// ".flac"=""
		// ".mp3"=""
		sb.WriteString(`
[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\drt.exe\shell\open\command]
@="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""

`)
		progIDs(&sb, es...)
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.csv\Shell\Open\Command]
		// @="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mov\Shell\Open\Command]
		// @="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mp4\Shell\Open\Command]
		// @="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.flac\Shell\Open\Command]
		// @="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""
		// [HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags.mp3\Shell\Open\Command]
		// @="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""
		sb.WriteString(`
[HKEY_CURRENT_USER\SOFTWARE\Abakum\drTags\Capabilities]
"ApplicationName"="drTags"
"ApplicationDescription"="Tagger for DaVinci Resolve"
`)
		FileAssociations(&sb, es...)
		// [HKEY_CURRENT_USER\SOFTWARE\Abakum\drTags\Capabilities\FileAssociations]
		// ".csv"="Abakum.drTags.csv"
		// ".mov"="Abakum.drTags.mov"
		// ".mp4"="Abakum.drTags.mp4"
		// ".flac"="Abakum.drTags.flac"
		// ".mp3"="Abakum.drTags.mp3"
		sb.WriteString(`
[HKEY_CURRENT_USER\SOFTWARE\RegisteredApplications]
"drTags"="Software\\Abakum\\drTags\\Capabilities"
`)
		TypeByExtension(&sb, mes, es...)
	}
	regS := strings.Replace(sb.String(), qq(`C:\Users\user_\go\bin\drt.exe`), qq(exe), -1)

	log.Println(reg.Name())
	fmt.Println(regS)

	// Пишем .reg с UTF-16 LE BOM
	encoding := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	encoder := encoding.NewEncoder()
	writer := transform.NewWriter(reg, encoder)
	writer.Write([]byte(regS))
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

func SupportedTypes(sb *strings.Builder, es ...string) {
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Classes\Applications\drt.exe\SupportedTypes]` + "\n")
	for _, e := range es {
		sb.WriteString(fmt.Sprintf(`"%s"=""`+"\n", e))
	}
}

func progIDs(sb *strings.Builder, es ...string) {
	for _, e := range es {
		sb.WriteString(fmt.Sprintf(`[HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags%s\Shell\Open\Command]`+"\n", e))
		sb.WriteString(`@="\"C:\\Users\\user_\\go\\bin\\drt.exe\" \"%1\""` + "\n")
	}
}

func removeIDs(sb *strings.Builder, es ...string) {
	for _, e := range es {
		sb.WriteString(fmt.Sprintf(`[-HKEY_CURRENT_USER\SOFTWARE\Classes\Abakum.drTags%s]`+"\n", e))
	}
}

func FileAssociations(sb *strings.Builder, es ...string) {
	sb.WriteString(`[HKEY_CURRENT_USER\SOFTWARE\Abakum\drTags\Capabilities\FileAssociations]` + "\n")
	for _, e := range es {
		sb.WriteString(fmt.Sprintf(`"%s"="Abakum.drTags%s"`+"\n", e, e))
	}
}

func TypeByExtension(sb *strings.Builder, m map[string]string, es ...string) {
	for _, e := range es {
		if t, ok := m[e]; ok && mime.TypeByExtension(e) == "" {
			sb.WriteString(fmt.Sprintf(`[HKEY_CURRENT_USER\SOFTWARE\Classes\%s]`+"\n", e))
			sb.WriteString(fmt.Sprintf(`@="Abakum.drTags%s"`+"\n", e))
			sb.WriteString(fmt.Sprintf(`"Content Type"="%s"`+"\n", t))
			sb.WriteString(fmt.Sprintf(`"PerceivedType"="%s"`+"\n", strings.Split(t, "/")[0]))
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
