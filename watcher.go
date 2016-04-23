package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogWatcher struct {
	lgr          LOGGER
	wg           *sync.WaitGroup
	shutdownChan chan struct{}
	logDir       string
	ticker       *time.Ticker
	cfg          *WatcherConfig
	logManager   *LogManager
	hekad        *Hekad
	change       bool
}

func NewLogWatcher(lgr LOGGER, logDir string, cfg *WatcherConfig,
	logManager *LogManager, hekad *Hekad, shutdownChan chan struct{},
	wg *sync.WaitGroup) (*LogWatcher, error) {

	watcher := &LogWatcher{
		lgr:          lgr,
		wg:           wg,
		shutdownChan: shutdownChan,
		logDir:       logDir,
		cfg:          cfg,
		ticker:       time.NewTicker(cfg.Interval),
		logManager:   logManager,
		hekad:        hekad,
	}
	return watcher, nil
}

func (w *LogWatcher) foundKafkafeeder(path string, info os.FileInfo) error {
	var (
		realPath = path
		realInfo = info
		err      error
	)
	if IsSymlink(info) {
		realPath, realInfo, err = ReadSymlink(path, info)
		if err != nil {
			return err
		}
	}
	added, err := w.logManager.Add(path, realPath, realInfo)
	if err != nil {
		return fmt.Errorf("Error parsing %v: %v", path, err)
	}
	if added {
		w.change = true
		w.lgr.Infof("New kafkafeeder found %q", path)
	}
	return nil
}

func (w *LogWatcher) checkPath(path string, info os.FileInfo, err error) error {
	if err != nil {
		w.lgr.Warnf("Error walking %q", err)
		return nil // continue
	}
	if info.IsDir() {
		return nil
	}
	if info.Name() == "kafkafeeder.yaml" {
		err = w.foundKafkafeeder(path, info)
		if err != nil {
			w.lgr.Warnf("Error adding kafkafeeder: %q", err)
		}
		return nil
	}
	if IsSymlink(info) {
		path, info, err = ReadSymlink(path, info)
		if err != nil {
			w.lgr.Warnf("Error reading symlink %q", err)
			return nil
		}
		if info.IsDir() {
			return w.lookForKafkafeeders(path)
		}
	}
	return nil
}

func (w *LogWatcher) lookForKafkafeeders(root string) error {
	return filepath.Walk(root, w.checkPath)
}

func (w *LogWatcher) Run() {
	w.lgr.Infof("start scaning %s", w.logDir)
	run := true
	for run {
		select {
		case <-w.ticker.C:
			w.change = false
			if err := w.lookForKafkafeeders(w.logDir); err != nil {
				w.lgr.Errorf("Walk log dir error %q", err)
			}
			if w.logManager.KeepValid() {
				w.change = true
			}
			if w.change {
				w.hekad.Reload()
			}
			break
		case <-w.shutdownChan:
			w.lgr.Infof("shutdown accepted")
			run = false
			break
		}
	}
	w.wg.Done()
	w.lgr.Infof("stopped")
}
