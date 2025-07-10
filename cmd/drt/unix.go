//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func createAppLock(appName string) (*os.File, error) {
	// Создаем временный файл
	file, err := os.CreateTemp("", appName+"_*.lock")
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
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		os.Remove(lockFile.Name())
	}
}
