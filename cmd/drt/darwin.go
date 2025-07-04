//go:build darwin

package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
	"github.com/xlab/closer"
)

var (
	met = map[string]string{
		dotCSV:  "public.comma-separated-values-text",
		dotMOV:  "com.apple.quicktime-movie",
		dotMP4:  "public.mpeg-4",
		dotFLAC: "org.xiph.flac",
		dotMP3:  "public.mp3",
	}
	DRS = []string{
		filepath.Join(xdg.DataHome,
			"..",
			"Containers",
			"com.blackmagic-design.DaVinciResolveLite",
			"Data",
			"Library",
			"Application Support",
		),
		filepath.Join(xdg.DataHome, "Blackmagic Design", "DaVinci Resolve"),
		filepath.Join(xdg.DataDirs[0], "Blackmagic Design", "DaVinci Resolve"),
	}
	setShellCmd = func() string {
		ex := drTags
		if _, err := exec.LookPath(ex); err != nil {
			//Если не в путёвом
			ex = filepath.Join(dir, drTags)
		}
		return `
set shellCmd to "` + ex + `"
if (count of input) > 0 then
	set quotedPaths to {}
	repeat with anItem in input
		set end of quotedPaths to quoted form of (POSIX path of anItem)
	end repeat
	
	set AppleScript's text item delimiters to " "
	set shellCmd to shellCmd & " " & (quotedPaths as text)
	set AppleScript's text item delimiters to ""
end if
`
	}
	// Запуск shellCmd в терминале
	tellTerminal = `
set bundleID to "` + repo + `"
tell application "Terminal"
	--When the script starts, Terminal sometimes creates an empty first window.
	--The idea is to open the script in the first window if it doesn't already contain this script,
	--and if it does contain this script, to open a new window instead.
	
	if (exists window 1) and not (custom title of first tab of window 1 is bundleID) then
		try
			do script shellCmd in window 1
		on error
			do script shellCmd
		end try
	else
		do script shellCmd
	end if

	set custom title of first tab of front window to bundleID
	activate
end tell`
)

//go:embed drTags.app
var app embed.FS

func onMain() {
	if strings.ToLower(args0) != droplet {
		return
	}
	exec.Command("osascript", "-e", `
if not (application "Terminal" exists) then
	set commandToRun to "open -a drTags.app"
	set the clipboard to commandToRun
	display dialog "Paste" & commandToRun & "by press Cmd+V" buttons {"OK"} default button 1 with icon note
	return
end if

tell application "Finder"
    if exists window 1 then
		activate
`+setShellCmd()+
		`
	end
end tell
`+tellTerminal).Start()
	closer.Close()
}

// https://github.com/RichardBronosky/AppleScript-droplet
func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// /Applications/drTags.app dir/drTags
	workflow(oldname, drTags)

	if oldname == "" {
		//uninstall
		log.Println(adr, "~> /dev/null", os.RemoveAll(adr))
		log.Println(link, "~> /dev/null", os.Remove(link))
		return
	}
	ln(oldname, link, true, false)

	applications := filepath.Dir(adr)

	// Walk through the embedded directory and copy files/dirs.
	fs.WalkDir(app, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(applications, path)

		if d.IsDir() {
			// Create destination directory if it doesn't exist.
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				err = os.MkdirAll(destPath, 0755)
				if err != nil {
					log.Println("Error creating directory:", err)
					return err
				}
			}
			return nil
		}

		// Copy file.
		srcFile, err := app.Open(path)
		if err != nil {
			log.Println("Error opening embedded file:", err)
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			log.Println("Error creating destination file:", err)
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			log.Println("Error copying file:", err)
			return err
		}
		log.Println(path, "~>", destPath)
		return nil
	})
	main := filepath.Join(adr, "Contents")

	drtlet := filepath.Join(main, "MacOS", droplet)
	ln(oldname, drtlet, true, false)
	log.Println("chmod +x", drtlet, os.Chmod(drtlet, 0755))

	return
	main = filepath.Join(main, "Resources", "Scripts")
	scpt := filepath.Join(main, "main.scpt")
	applescript := filepath.Join(main, "main.applescript")
	if _, err := exec.LookPath(drTags); err != nil {
		data, err := os.ReadFile(applescript)
		log.Println("<~", applescript, err)
		if err == nil {
			data = bytes.Replace(data, []byte(`"`+drTags+`"`), []byte(`"`+link+`"`), 1)
			err = os.WriteFile(applescript, data, 0644)
			log.Println("~>", applescript, err)
		}
	}
	log.Println(applescript, "~>", scpt, OSACompile(applescript, scpt))
}

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}

func OSACompile(src, trg string) error {
	// Проверяем существование исходного файла
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("исходный файл не существует: %s", src)
	}

	// Определяем язык по расширению
	ext := strings.ToLower(filepath.Ext(src))
	var lang string

	switch ext {
	case ".applescript":
		lang = "AppleScript"
	case ".js":
		lang = "JavaScript"
	default:
		return fmt.Errorf("неподдерживаемый формат скрипта: %s", ext)
	}

	// Проверяем соответствие расширений
	targetExt := filepath.Ext(trg)
	if (lang == "AppleScript" && targetExt != ".scpt") ||
		(lang == "JavaScript" && targetExt != ".jsc") {
		return fmt.Errorf("несоответствие расширений: исходный %s -> целевой %s", ext, targetExt)
	}

	// Создаем директорию для целевого файла, если нужно
	if err := os.MkdirAll(filepath.Dir(trg), 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию: %v", err)
	}

	// Вызываем osacompile
	cmd := exec.Command("osacompile", "-l", lang, "-o", trg, src)

	// Захватываем вывод ошибок
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка компиляции: %v\n%s", err, string(output))
	}

	return nil
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return ln(oldname, newname, link, hard)
}

func workflow(oldName, serviceName string) {
	servicePath := filepath.Join(xdg.DataHome, "..", "Library", "Services", serviceName+".workflow") // dir
	if oldName == "" {
		log.Println(servicePath, "~> /dev/null", os.RemoveAll(servicePath))
		return
	}

	serviceScript := `
on run {input, parameters}
    -- Проверяем, есть ли выделенные файлы
    if (count of input) = 0 then
        display alert "Выбери" message " файлы или каталоги и повтори" as warning
        return
    end if
` + setShellCmd() +
		tellTerminal + `
end run
`
	os.MkdirAll(filepath.Join(servicePath, "Contents"), 0755)

	// 1. Создаем Info.plist с заполненными данными
	infoPlistContent := fillInfoPlistTemplate(serviceName, repo)
	if createFile(filepath.Join(servicePath, "Contents", "Info.plist"), infoPlistContent) != nil {
		return
	}

	// 2. Создаем document.wflow
	documentContent := fillDocumentWflowTemplate(serviceName, serviceScript)
	if createFile(filepath.Join(servicePath, "Contents", "document.wflow"), documentContent) != nil {
		return
	}

	// 3. Обновляем сервисы
	cmd := exec.CommandContext(ctx, "/System/Library/CoreServices/pbs", "-flush")
	err := cmd.Start()
	log.Println(cmd, err)
	if err != nil {
		return
	}
	cmd.Wait()
}

func createFile(path string, content string) (err error) {
	err = os.WriteFile(path, []byte(content), 0644)
	log.Println("~>", path, err)
	return
}

func fillInfoPlistTemplate(serviceName, BundleId string) string {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>NSServices</key>
    <array>
        <dict>
            <key>NSMenuItem</key>
            <dict>
                <key>default</key>
                <string>{{.Name}}</string>
            </dict>
            <key>NSMessage</key>
            <string>runWorkflowAsService</string>
            <key>NSSendFileTypes</key>
            <array>
                <string>public.item</string>
            </array>
            <key>NSRequiredContext</key>
            <dict>
                <key>NSTextContext</key>
                <string>FilePath</string>
            </dict>
        </dict>
    </array>
    <key>CFBundleIdentifier</key>
    <string>{{.ID}}</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
</dict>
</plist>`

	t := template.Must(template.New("infoplist").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, struct {
		Name string
		ID   string
	}{
		Name: serviceName,
		ID:   BundleId,
	})
	return buf.String()
}

func fillDocumentWflowTemplate(serviceName, script string) string {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>AMAccepts</key>
    <dict>
        <key>Container</key>
        <string>List</string>
        <key>Optional</key>
        <true/>
        <key>Types</key>
        <array>
            <string>public.item</string>
        </array>
    </dict>
    <key>AMAction</key>
    <array>
        <dict>
            <key>AMAccepts</key>
            <dict>
                <key>Container</key>
                <string>List</string>
                <key>Optional</key>
                <true/>
                <key>Types</key>
                <array>
                    <string>public.item</string>
                </array>
            </dict>
            <key>AMActionType</key>
            <string>AppleScript</string>
            <key>AMAppleScript</key>
            <string>{{.Script}}</string>
            <key>AMCanShowWhenRun</key>
            <true/>
            <key>AMCategory</key>
            <string>AMCategoryUtilities</string>
            <key>AMIconName</key>
            <string>Script</string>
            <key>AMName</key>
            <string>{{.Name}}</string>
            <key>AMVersion</key>
            <string>1.0</string>
        </dict>
    </array>
    <key>AMApplication</key>
    <array>
        <string>com.apple.Finder</string>
    </array>
    <key>AMDoc</key>
    <dict>
        <key>version</key>
        <string>1.0</string>
    </dict>
</dict>
</plist>
`

	t := template.Must(template.New("wflow").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, struct {
		Name   string
		Script string
	}{
		Name:   serviceName,
		Script: script,
	})
	return buf.String()
}
