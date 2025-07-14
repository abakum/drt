//go:build !windows && !darwin

package main

import (
	"fmt"
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
		dotCSV:  "text/csv",
		dotMOV:  "video/quicktime",
		dotMP4:  "video/mp4",
		dotFLAC: "audio/flac",
		dotMP3:  "audio/mpeg",
	}
	DRS = []string{
		filepath.Join(xdg.DataHome, "DaVinciResolve"),
		filepath.Join("/opt", "resolve"),
		filepath.Join("/home", "resolve"),
	}
)

func install(oldname string, lnks ...string) {
	desktop, sh, link, applications, adr, xdgDesktopIcon, verb := lnks[0], lnks[1], lnks[2], lnks[3], lnks[4], lnks[5], lnks[6]
	ln(oldname, link, true, false)

	ex := LookPath(drTags)
	if _, err := exec.LookPath(nautilus); err == nil {
		shString := fmt.Sprintf(`#!/usr/bin/env bash

x-terminal-emulator -e %s`, ex)
		if hasGtkLaunch() {
			shString = fmt.Sprintf(`#!/usr/bin/env bash

%s %s`, launch, drTags)
		}
		log.Println("Меню для", nautilus, sh,
			os.WriteFile(sh, []byte(shString), 0744))
	}
	deskTop(desktop, ex)
	cmd := exec.CommandContext(ctx, "desktop-file-install", "--rebuild-mime-info-cache", desktop, "--dir="+applications) //
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
Name=`+drTags+`
Comment=`+FriendlyAppName+`
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

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return ln(oldname, newname, link, hard)
}

func onMain(consoleTitle string) {
	fmt.Printf("\033]0;%s\007", consoleTitle)
}
