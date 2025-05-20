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

func run(ctx context.Context, bin, root string, args ...string) (rc uint32, err error) {
	path, err := exec.LookPath(bin)
	if err == nil {
		path = filepath.Join(path, bin)
	} else {
		path = bin
	}
	log.Output(3, path+" "+strings.Join(args, " "))
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
		// "-show_entries", "stream=codec_name,bit_rate,sample_fmt,coded_width,coded_height,duration,sample_rate",
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
