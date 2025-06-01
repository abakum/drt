package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

const (
	ffprobe = "ffprobe"
	ffmpeg  = "ffmpeg"
)

func run(ctx context.Context, writer io.Writer, bin, root string, args ...string) (rc uint32, err error) {
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
				cmd.Stdout = writer
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

func probe(dir, base string, video bool) (audio string, lines []string) {
	log.Println(filepath.Join(dir, base))
	args := []string{
		"-hide_banner",
		"-v", "error",
	}
	if video {
		args = append(args,
			"-select_streams", "v:0",
		)
	}
	args = append(args,
		"-show_entries", "stream=codec_name,bit_rate,sample_fmt,width,height,r_frame_rate,profile,level",
		"-of", "default=noprint_wrappers=1",
		base,
	)
	var spy bytes.Buffer

	rc, err := run(ctx, bufio.NewWriter(&spy), ffprobe, dir, args...)
	if err != nil {
		log.Println("ошибка", err, "код завершения", ffprobe, rc)
		return
	}
	br := "bit_rate="
	i := 1
	for {
		line, err := spy.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "codec_name="):
			switch i {
			case 1:
				//Видео
			case 2:
				//Аудио
				audio = strings.Split(line, "=")[1]
			default:
				return
			}
			i++
		case strings.Contains(line, "=0"):
			continue
		case strings.Contains(line, "=N/A"):
			continue
		case strings.Contains(line, "=unknown"):
			continue
		case strings.HasPrefix(line, br):
			val := strings.TrimPrefix(line, br)
			if i, err := strconv.Atoi(val); err == nil {
				line = fmt.Sprintf(br+"%d kb/s", i/1000)
			}
		}
		// if strings.HasPrefix(line, "codec_name=") {
		// 	if i > 2 {
		// 		return
		// 	}
		// 	i++
		// }
		// if strings.Contains(line, "=0") {
		// 	continue
		// }
		// if strings.Contains(line, "=N/A") {
		// 	continue
		// }
		// if strings.Contains(line, "=unknown") {
		// 	continue
		// }
		// if strings.HasPrefix(line, br) {
		// 	val := strings.TrimPrefix(line, br)
		// 	if i, err := strconv.Atoi(val); err == nil {
		// 		line = fmt.Sprintf(br+"%d kb/s", i/1000)
		// 	}
		// }
		lines = append(lines, line)
	}
}
