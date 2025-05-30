package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

const (
	ffprobe = "ffprobe"
	ffmpeg  = "ffmpeg"
)

func run(ctx context.Context, bin, root string, args ...string) (rc uint32, err error) {
	qArgs := []string{bin}
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			arg = `"` + arg + `"`
		}
		qArgs = append(qArgs, arg)
	}
	out := func(wasm bool) {
		if wasm {
			log.Output(3, "Использую встроенный "+bin+" v5.1.6 wasm")
			// } else {
			// 	log.Output(3, "Надеюсь установлен "+bin+" новей и быстрей чем встроенный v5.1.6 wasm")
		}
		if bin != ffprobe {
			log.Output(3, strings.Join(qArgs, " "))
		}
	}
	if path, err := exec.LookPath(bin); err == nil {
		// log.Println(path, err, "path, err")
		if exe, err := os.Executable(); err == nil {
			// log.Println(exe, err, "exe, err")
			if resolved, err := filepath.EvalSymlinks(path); err == nil && resolved != exe {
				qArgs[0] = path
				out(false)

				cmd := exec.CommandContext(ctx, path, args...)
				cmd.Dir = root
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				return uint32(cmd.ProcessState.ExitCode()), err
			}
		}
	}
	out(true)
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
	log.Println(filepath.Join(dir, base))

	rc, err := run(ctx, ffprobe, dir,
		"-hide_banner",
		"-v", "error",
		"-show_entries", "stream=codec_name,bit_rate,sample_fmt,coded_width,coded_height",
		"-of", "default=noprint_wrappers=1",
		base,
	)
	if err != nil {
		log.Println("ошибка", err, "код завершения", ffprobe, rc)
	}

}

func probeV(dir, base string) {
	log.Println(filepath.Join(dir, base))

	rc, err := run(ctx, ffprobe, dir,
		"-hide_banner",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate,duration",
		"-of", "default=noprint_wrappers=1",
		base,
	)
	if err != nil {
		log.Println("bit_rate=?", err, "код завершения", ffprobe, rc)
	}
}
