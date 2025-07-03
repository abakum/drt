//go:build darwin

package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
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
)

//go:embed drTags.app
var app embed.FS

//https://github.com/RichardBronosky/AppleScript-droplet

func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// ~/Applications/drTags.app dir/drTags
	if oldname == "" {
		//uninstall
		log.Println(adr, "~> /dev/null", os.RemoveAll(adr))
		for _, lnk := range lnks[1:] {
			log.Println(lnk, "~> /dev/null", os.Remove(lnk))
		}
		return
	}
	for _, lnk := range lnks[1:] {
		ln(oldname, lnk, true, false)
	}

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
	files := []string{filepath.Join(main, "MacOS", "droplet")}
	for _, f := range files {
		log.Println("chmod +x", f, os.Chmod(f, 0755))
	}
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
