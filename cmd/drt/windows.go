//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jxeng/shortcut"
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
	name := trimExt(newname)
	err = os.WriteFile(name+".cmd", []byte(oldname+" %*"), 0744)
	if err != nil {
		log.Println("Error write .cmd:", err)
	}
	return
}

func install(oldname string, lnks ...string) {
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

type ST struct {
	userDir  func() (string, error)
	p, m     string
	root, dr string
}

func swap(sts ...ST) {
	stm := make(map[bool]ST, 2)
	var err error
	for i, st := range sts {
		st.root, err = st.userDir()
		if err != nil {
			return
		}
		st.dr = filepath.Join(st.root, st.p, drTags+ext)
		os.Remove(st.dr) // если старая ссылка не на drt то удалится
		stm[i > 0] = st
	}
	i := false
	f, err := open(stm[!i].dr)
	if err == nil {
		f.Close()
	} else {
		i = !i
	}
	if yes(stm[i].m) {
		defer ctrlC()
	} else {
		i = !i
	}
	rename(exe, stm[!i].dr, stm[i].dr)
}

func rename(exe, s, t string) {
	f, err := open(s)
	if err == nil {
		f.Close()
		log.Println(s, "~>", t, os.Rename(s, t))
	} else {
		f, err = open(t)
		if err != nil {
			log.Println(exe, "->", t, mkLink(exe, t, true, true))
		} else {
			f.Close()
		}
	}
}
