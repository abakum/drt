//go:build darwin

package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func install_(oldname string, lnks ...string) {
	// ~/Applications/drTags.app dir/drTags
	adr, link := lnks[0], lnks[1]
	if oldname == "" {
		//uninstall
		for _, lnk := range lnks {
			os.RemoveAll(lnk)
		}
		return
	}
	mkLink(oldname, link, true, false)

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
	os.Remove(main)
	mkLink(oldname, main, true, false)
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	if link {
		opt := "/s"
		osLink := os.Symlink
		m := "symbolic"
		if hard {
			osLink = os.Link
			opt = ""
			m = "hard"
		}
		err = osLink(oldname, newname)
		log.Println("ln", opt, oldname, newname, err)
		if err == nil {
			return
		}
		log.Printf("Error creating %s link: %v\n", m, err)
		return
	}
	err = os.WriteFile(newname, []byte(`#!/usr/bin/env bash

set -o nounset
set -o errexit
`+oldname+` "${@}"`), 0744)
	if err != nil {
		log.Println("Error write .sh:", err)
	}
	return
}
