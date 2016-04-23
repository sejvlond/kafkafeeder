package main

import (
	"sync"
	"time"
)

type LogCleaner struct {
	lgr          LOGGER
	wg           *sync.WaitGroup
	shutdownChan chan struct{}
	ticker       *time.Ticker
	cfg          *CleanerConfig
}

func NewLogCleaner(lgr LOGGER, cfg *CleanerConfig,
	shutdownChan chan struct{}, wg *sync.WaitGroup) (*LogCleaner, error) {

	cleaner := &LogCleaner{
		lgr:          lgr,
		wg:           wg,
		shutdownChan: shutdownChan,
		ticker:       time.NewTicker(cfg.Interval),
		cfg:          cfg,
	}
	return cleaner, nil
}

func (c *LogCleaner) Run() {
	c.lgr.Infof("started")
	run := true
	for run {
		select {
		case <-c.ticker.C:
			// TODO
			c.lgr.Errorf("NOT IMPLEMENTED walk and remove old logs")
			break
		case <-c.shutdownChan:
			c.lgr.Infof("shutdown accepted")
			run = false
			break
		}
	}
	c.wg.Done()
	c.lgr.Infof("stopped")
}
