//go:build !windows && !darwin

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	met = map[string]string{
		".csv":  "text/csv",
		".mov":  "video/quicktime",
		".mp4":  "video/mp4",
		".flac": "audio/flac",
		".mp3":  "audio/mpeg",
	}
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
	err = os.WriteFile(newname, []byte(`#!/usr/bin/env bash

set -o nounset
set -o errexit
`+oldname+` "${@}"`), 0744)
	if err != nil {
		log.Println("Error write .sh:", err)
	}
	return
}

func install_(oldname string, lnks ...string) {
	desktop, sh, link, application, adr, xdgDesktopIcon, verb := lnks[0], lnks[1], lnks[2], lnks[3], lnks[4], lnks[5], lnks[6]
	if f, err := open(link); err == nil {
		f.Close()
	} else {
		mkLink(oldname, link, true, true)
	}

	ex := drTags
	if _, err := exec.LookPath(ex); err != nil {
		//Если не в путёвом
		ex = link
	}

	if _, err := exec.LookPath("nautilus"); err == nil {
		log.Println("Меню для nautilus", sh,
			os.WriteFile(sh, []byte(fmt.Sprintf(`#!/bin/bash
gnome-terminal --title %s -- %s`, drTags, drTags)), 0744))
	}
	deskTop(desktop, ex)
	cmd := exec.CommandContext(ctx, "desktop-file-install", "--rebuild-mime-info-cache", desktop, "--dir="+application) //
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println(cmd.Args, cmd.Run())
	if f, err := open(adr); err == nil {
		f.Close()
	} else {
		deskTop(adr, ex)
	}
	cmd = exec.CommandContext(ctx, xdgDesktopIcon, verb, "--novendor", adr)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println(cmd.Args, cmd.Run())
}

func deskTop(desktop, ex string) {
	MimeType := strings.Join(Values(met), ";")
	log.Println("Ярлык на рабочем столе", desktop,
		os.WriteFile(desktop, []byte(`[Desktop Entry]
Name=drTags
Type=Application
Exec=`+ex+` %F
Terminal=true
Icon=edit-find-replace
NoDisplay=false
MimeType=`+MimeType+`;
Categories=AudioVideo;AudioVideoEditing;
Keywords=media info;metadata;tag;video;audio;codec;davinci;resolve;
`), 0644))
}
