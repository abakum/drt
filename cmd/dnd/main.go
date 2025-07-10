package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	title   = "dr&Tags"
	appName = "droplet"
	dX      = 100
	dY      = 80
)

func main() {
	log.SetFlags(log.Lshortfile)

	// Проверяем, не запущен ли дроплет
	cleanup, err := initializeAppLock(appName)
	if err != nil {
		log.Fatalf("Application lock failed: %v", err)
	}
	defer cleanup()

	onMain(title)
}
func logPaths(paths string) {
	fmt.Println(paths)
	return
	for _, path := range strings.Split(paths, "\n") {
		fmt.Println(path)
	}
}

func initializeAppLock(appName string) (func(), error) {
	// Проверить существующие lock-файлы
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), appName+"_*.lock"))
	if err != nil {
		return nil, fmt.Errorf("failed to search lock files: %v", err)
	}

	for _, filename := range matches {
		content, err := os.ReadFile(filename)
		if err != nil {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil {
			continue
		}

		if checkProcessExists(pid) {
			return nil, fmt.Errorf("application already running (PID %d)", pid)
		}

		// Удаляем устаревший lock-файл

		log.Println(filename, "~> nul", os.Remove(filename))

	}

	// Создаем новый lock
	lockFile, err := createAppLock(appName)
	if err != nil {
		return nil, err
	}

	// Возвращаем функцию очистки
	return func() {
		cleanupLock(lockFile)
	}, nil
}
