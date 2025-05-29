//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func amAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func runElevated(exe, cwd string, showCmd int32, args ...string) (err error) {
	verb := "runas"
	// exe, _ := os.Executable()
	// cwd, _ := os.Getwd()
	arg := strings.Join(args, " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(arg)

	// var showCmd int32 = 1 //SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
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
			err = runElevated("cmd", wd, 1, "/c", fmt.Sprintf(`mklink %s "%s" "%s"`, opt, newname, oldname))
			if err == nil {
				return
			}
			log.Println("Error run mklink as Administrator:", err)
		}
		return
	}
	name := strings.TrimSuffix(newname, filepath.Ext(newname))
	err = os.WriteFile(name+".cmd", []byte(oldname+" %*"), 0744)
	if err != nil {
		log.Println("Error write .cmd:", err)
	}
	return
}
