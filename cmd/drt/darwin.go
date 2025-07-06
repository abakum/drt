//go:build darwin

package main

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
	"github.com/xlab/closer"
)

const (
	applescript = "main.applescript"
	Resources   = "Resources"
)

//go:embed drTags.app
var app embed.FS

//go:embed drTags.workflow
var workflow embed.FS

var (
	met = map[string]string{
		dotCSV:  "public.comma-separated-values-text",
		dotMOV:  "com.apple.quicktime-movie",
		dotMP4:  "public.mpeg-4",
		dotFLAC: "org.xiph.flac",
		dotMP3:  "public.mp3",
	}
	DRS = []string{
		filepath.Join(filepath.Dir(xdg.DataHome),
			"Containers",
			"com.blackmagic-design.DaVinciResolveLite",
			"Data",
			"Library",
			"Application Support",
		),
		filepath.Join(xdg.DataHome, "Blackmagic Design", "DaVinci Resolve"),
		filepath.Join(xdg.DataDirs[0], "Blackmagic Design", "DaVinci Resolve"),
	}
)

func onMain() {
	if strings.ToLower(args0) != droplet {
		return
	}
	MacOS := filepath.Dir(os.Args[0])
	Contents := filepath.Dir(MacOS)
	scriptName := filepath.Join(Contents, Resources, applescript)
	script, err := os.ReadFile(scriptName)
	if err == nil {
		cmd := exec.Command("osascript", "-e", string(script))
		// log.Println(string(script))
		output, err := cmd.CombinedOutput()
		log.Println(cmd, err)
		if err != nil {
			log.Println(string(output))
		}
	} else {
		log.Println("<~", scriptName, err)
	}
	closer.Close()
}

// https://github.com/RichardBronosky/AppleScript-droplet
func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// /dest/drTags.app dir/drTags
	servicePath := filepath.Join(filepath.Dir(xdg.DataHome), "Services", drTags+".workflow") // dir
	if oldname == "" {
		//uninstall
		log.Println(adr, "~> /dev/null", os.RemoveAll(adr))
		log.Println(servicePath, "~> /dev/null", os.RemoveAll(servicePath))
		log.Println(link, "~> /dev/null", os.Remove(link))
		return
	}
	ln(oldname, link, true, false)

	// Грязный результат fn
	var script []byte

	// Грязные параметры для fn
	_, err := exec.LookPath(drTags)
	replace := err != nil
	dest := filepath.Dir(servicePath)
	efs := workflow

	// Walk through the embedded directory and copy files/dirs.
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, path)
		// log.Println(path, destPath, d.Name(), "path,destPath,d.Name()")

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

		if filepath.Base(path) == applescript {
			script, err = efs.ReadFile(path)
			log.Println("<~", path, err)
			if err != nil {
				return err
			}
			if replace {
				script = bytes.Replace(script, []byte(`"`+drTags+`"`), []byte(`"`+filepath.Join(dir, drTags)+`"`), 1)
				err = os.WriteFile(destPath, script, 0644)
				log.Println("~>", path, err)
				if err != nil {
					return err
				}
				return nil
			}
		}

		// Copy file.
		srcFile, err := efs.Open(path)
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
		log.Println(path, "~>", destPath, err)
		if err != nil {
			return err
		}
		return nil
	}

	fs.WalkDir(efs, ".", fn)
	pbs(servicePath, string(script))

	dest = filepath.Dir(adr)
	efs = app
	fs.WalkDir(efs, ".", fn)
	Contents := filepath.Join(adr, "Contents")
	MacOS := filepath.Join(Contents, "MacOS")
	drtlet := filepath.Join(MacOS, droplet)
	os.MkdirAll(MacOS, 0755)
	ln(oldname, drtlet, true, false)
	log.Println("chmod +x", drtlet, os.Chmod(drtlet, 0755))

}

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return ln(oldname, newname, link, hard)
}

func pbs(servicePath, workflowScript string) {
	documentFile := filepath.Join(servicePath, "Contents", "document.wflow")

	// Чтение файла с логгированием
	documentBytes, err := os.ReadFile(documentFile)
	log.Println("<~", documentFile, err)
	if err != nil {
		return
	}

	documentContent := string(documentBytes)
	log.Printf(documentContent)
	log.Printf(workflowScript)

	// Улучшенное регулярное выражение для поиска блока ActionParameters
	re := regexp.MustCompile(`(?s)<key>ActionParameters</key>\s*<dict>\s*<key>source</key>\s*<string>(.*?)</string>`)

	// Заменяем содержимое CDATA
	if re.MatchString(documentContent) {
		// Заменяем существующий скрипт
		documentContent = re.ReplaceAllString(documentContent,
			`<key>ActionParameters</key>
                <dict>
                    <key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
	} else {
		// Если не нашли стандартную структуру, вставляем в более общем месте
		re = regexp.MustCompile(`(?s)<key>source</key>\s*<string>(.*?)</string>`)
		if re.MatchString(documentContent) {
			documentContent = re.ReplaceAllString(documentContent,
				`<key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
		} else {
			log.Println("Не удалось найти место для вставки скрипта в document.wflow")
			return
		}
	}

	// Запись обратно с логгированием
	err = os.WriteFile(documentFile, []byte(documentContent), 0644)
	log.Println("~>", documentFile, err)
	if err != nil {
		return
	}

	// Обновляем сервисы
	cmd := exec.Command("/System/Library/CoreServices/pbs", "-flush")
	err = cmd.Start()
	log.Println(cmd, err)
	if err != nil {
		return
	}
	cmd.Wait()
}

// tell application "Finder"
//     activate
//     -- Снимаем выделение во всех окнах
//     repeat with win in windows
//         set selection of win to {}
//     end repeat
// end tell

//osascript -e 'on run {input, parameters} ... end run' "/path/to/file1" "/path/to/file2"
