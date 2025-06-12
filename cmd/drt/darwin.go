//go:build darwin

package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

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
					fmt.Println("Error creating directory:", err)
					return err
				}
			}
			return nil
		}

		// Copy file.
		srcFile, err := app.Open(path)
		if err != nil {
			fmt.Println("Error opening embedded file:", err)
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			fmt.Println("Error creating destination file:", err)
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			fmt.Println("Error copying file:", err)
			return err
		}
		fmt.Println(path, "~>", destPath)
		return nil
	})
	main := filepath.Join(adr, "Contents", "Resources", "Scripts", "main")
	log.Println(main, "~> /dev/null", os.Remove(main))
	ln(oldname, main, true, false)
}
func evtp() {
}
