//go:build !windows

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
)

func createAppLock(appName string) (*os.File, error) {
	// Создаем временный файл
	file, err := os.CreateTemp("", appName+"_*"+dotLock)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %v", err)
	}

	// Записываем PID
	_, err = file.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, fmt.Errorf("failed to write PID: %v", err)
	}

	// Устанавливаем блокировку файла
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, fmt.Errorf("failed to lock file: %v (is app already running?)", err)
	}

	return file, nil
}

func checkProcessExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func cleanupLock(lockFile *os.File) {
	if lockFile != nil {
		err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		if err != nil {
			log.Println(lockFile.Name(), "unLock", err)
		}
		lockFile.Close()
		os.Remove(lockFile.Name())
	}
}

func isGUI() bool {
	return strings.ToLower(args0) != drt
}

// safeWriteFile_Linux записывает данные в файл с эксклюзивной блокировкой (flock).
// Пакет: syscall (Linux/Unix) или golang.org/x/sys/unix.
func safeWriteFile(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Блокировка файла (эксклюзивная)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN) // Разблокировать при выходе

	// Запись данных
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
