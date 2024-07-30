package main

import (
	"errors"
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
