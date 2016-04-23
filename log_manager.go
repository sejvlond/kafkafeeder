package main

import (
	"os"
	"path/filepath"
	"time"
)

const (
	ACK_DISK_WRITE   = -1
	ACK_MEMORY_WRITE = 1
	ACK_DISABLED     = 0
)

type TopicConfig struct {
	Topic     string
	Type      string
	Broker    string
	Retention time.Duration
	Ack       int
}

type LogConfig struct {
	ModTime   time.Time
	Directory string
	Topics    map[string]*TopicConfig
}

type LogManager struct {
	Logs map[string]*LogConfig
}

func NewLogManager() (*LogManager, error) {
	lm := &LogManager{
		Logs: make(map[string]*LogConfig),
	}
	return lm, nil
}

func (lm *LogManager) Add(linkPath string, file string, finfo os.FileInfo) (
	bool, error) {

	logCfg, ok := lm.Logs[file]
	if ok { // already exists
		if finfo.ModTime() == logCfg.ModTime ||
			finfo.ModTime().Before(logCfg.ModTime) { // is older than we have
			return false, nil
		}
	}
	// create new - when parsing fails add anyway
	logCfg, err := ParseFile(file)
	if logCfg == nil {
		logCfg = &LogConfig{}
	}
	logCfg.ModTime = finfo.ModTime()
	logCfg.Directory, _ = filepath.Abs(filepath.Dir(linkPath))
	lm.Logs[file] = logCfg
	return err == nil, err
}

func (lm *LogManager) KeepValid() bool {
	change := false
	for path, _ := range lm.Logs {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			delete(lm.Logs, path)
			change = true
		}
	}
	return change
}
