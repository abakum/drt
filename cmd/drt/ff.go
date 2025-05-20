package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

func run(ctx context.Context, bin, root string, args ...string) (rc uint32, err error) {
	qArgs := []string{bin}
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			arg = `"` + arg + `"`
		}
		qArgs = append(qArgs, arg)
	}
	// Может установлен ffmpeg новей и быстрей чем wasm version n5.1.6
	if path, err := exec.LookPath(bin); err == nil {
		log.Println(path, err, "path, err")
		if exe, err := os.Executable(); err == nil {
			log.Println(exe, err, "exe, err")
			if resolved, err := filepath.EvalSymlinks(path); err == nil && resolved != exe {
				log.Println(resolved, err, "resolved, err")
				qArgs[0] = path
				log.Output(3, strings.Join(qArgs, " "))

				cmd := exec.CommandContext(ctx, path, args...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				return uint32(cmd.ProcessState.ExitCode()), err
			}
		}
	}
	log.Output(3, strings.Join(qArgs, " "))
	cacheDir := os.TempDir()
	if ucd, err := os.UserCacheDir(); err == nil {
		cacheDir = ucd
	}
	os.Setenv("WAZERO_COMPILATION_CACHE", cacheDir)
	rc, err = ffmpreg.Run(ctx, wasm.Args{
		Name:   bin,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Args:   args,
		Config: func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
			for _, kv := range os.Environ() {
				i := strings.IndexByte(kv, '=')
				if i < 1 {
					continue
				}
				cfg = cfg.WithEnv(kv[:i], kv[i+1:])
			}
			fscfg := wazero.NewFSConfig()

			fscfg = fscfg.WithDirMount(root, "/")
			return cfg.WithFSConfig(fscfg)
		},
	})
	return
}

func probe(dir, base string) {
	rc, err := run(ctx, "ffprobe", dir,
		"-hide_banner",
		"-v", "error",
		"-show_entries", "stream=codec_name,bit_rate,sample_fmt,coded_width,coded_height",
		"-of", "default=noprint_wrappers=1",
		base,
	)
	if err != nil {
		log.Println("ошибка", err, "код завершения", "ffprobe", rc)
	}

}

func probeV(dir, base string) {
	rc, err := run(ctx, "ffprobe", dir,
		"-hide_banner",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate,duration",
		"-of", "default=noprint_wrappers=1", //:nokey=1
		base,
	)
	if err != nil {
		log.Println("bit_rate=?", err, "код завершения", "ffprobe", rc)
	}
}

const (
	DIRMODE  = 0755
	FILEMODE = 0644
)

func isFileExist(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}

func AntiLoop() (cleanUp func()) {
	e, err := os.Executable()
	// path/type.exe
	if err != nil {
		e = "started"
		// type
	} else {
		e = filepath.Base(e)
		// type.exe
		e = strings.Split(e, ".")[0]
		// type
	}
	tmp := os.TempDir()
	antiLoop := filepath.Join(tmp, e)
	if isFileExist(antiLoop) {
		return
	}
	os.MkdirAll(tmp, DIRMODE)
	if os.WriteFile(antiLoop, []byte{}, FILEMODE) == nil {
		return func() { os.Remove(antiLoop) }
	}
	return
}
