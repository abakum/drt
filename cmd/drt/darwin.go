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

	"github.com/adrg/xdg"
	"github.com/google/shlex"
)

var DRS = filepath.Join(xdg.DataHome, "Blackmagic Design", "DaVinci Resolve")

//go:embed drTags.app
var app embed.FS

//https://github.com/RichardBronosky/AppleScript-droplet

func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// ~/Applications/drTags.app dir/drTags
	if oldname == "" {
		//uninstall
		for _, lnk := range lnks {
			log.Println(lnk, "~> /dev/null", os.RemoveAll(lnk))
		}
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
	files := []string{filepath.Join(main, "MacOS", "droplet")}
	main = filepath.Join(main, "Resources", "Scripts", "main.scpt")
	files = append(files, main)
	for _, f := range files {
		log.Println("chmod +x", f, os.Chmod(f, 0755))
	}

	// https://github.com/abbeycode/AppleScripts/blob/master/Services/Convert%20Script%20to%20Text.applescript
	if _, err := exec.LookPath(drTags); err == nil {
		return
	}
	data, err := os.ReadFile(main)
	log.Println("Читаю скрипт", main, err)
	if err != nil {
		return
	}
	data = bytes.Replace(data, []byte(drTags), []byte(link), 1)
	log.Println("Пишу скрипт", main, os.WriteFile(main, data, 0755))
}

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}
