package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	workingDir     string
	workingDirOnce sync.Once
)

func getPalaceDir() string {
	workingDirOnce.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logFatal(err)
		}
		workingDir = filepath.Join(homeDir, ".moonpalace")
		stat, err := os.Stat(workingDir)
		switch {
		case err == nil:
			if !stat.IsDir() {
				logFatal(errors.New(workingDir + " is not a directory"))
			}
		case os.IsNotExist(err):
			if err = os.MkdirAll(workingDir, 0755); err != nil {
				logFatal(err)
			}
		default:
			logFatal(err)
		}
	})
	return workingDir
}

func getPalaceSqlite() string {
	palaceDir := getPalaceDir()
	sqlitePath := filepath.Join(palaceDir, "moonpalace.sqlite")
	if _, err := os.Stat(sqlitePath); err != nil {
		if os.IsNotExist(err) {
			if _, err = os.Create(sqlitePath); err != nil {
				logFatal(err)
			}
		} else {
			logFatal(err)
		}
	}
	return sqlitePath
}

func getPalaceServerErrorLog() io.Writer {
	palaceDir := getPalaceDir()
	serverErrorLogPath := filepath.Join(palaceDir, "server_error.log")
	file, err := os.OpenFile(serverErrorLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		logFatal(err)
	}
	return file
}

func getConfig() io.Reader {
	palaceDir := getPalaceDir()
	configPath := filepath.Join(palaceDir, "config.yaml")
open:
	file, err := os.Open(configPath)
	if err != nil {
		if filepath.Ext(configPath) == ".yaml" && os.IsNotExist(err) {
			configPath = filepath.Join(palaceDir, "config.yml")
			goto open
		}
		return nil
	}
	return file
}
