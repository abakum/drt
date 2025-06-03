//go:build !windows

package main

import (
	"log"
	"os"
)

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
	name := trimExt(newname)
	err = os.WriteFile(name+".sh", []byte(oldname+" %*"), 0744)
	if err != nil {
		log.Println("Error write .sh:", err)
	}
	return
}
