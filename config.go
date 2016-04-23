package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type LoggingConfig struct {
	Component string `yaml:"component"`
	Dir       string `yaml:"dir"`
	Interval  uint   `yaml:"interval"`
}

type HekadConfig struct {
	MainConfPath string              `yaml:"main_conf_path"`
	BinPath      string              `yaml:"bin_path"`
	ConfDir      string              `yaml:"conf_dir"`
	KafkaBrokers map[string][]string `yaml:"kafka_brokers"`
}

type CleanerConfig struct {
	Interval time.Duration `yaml:"interval"`
}

type WatcherConfig struct {
	Interval time.Duration `yaml:"interval"`
}

type Config struct {
	Logging LoggingConfig `yaml:"logging"`

	CheckpointDir string `yaml:"checkpoint_dir"`
	JournalDir    string `yaml:"journal_dir"`
	LogDir        string `yaml:"log_dir"`

	Hekad   HekadConfig   `yaml:"hekad"`
	Cleaner CleanerConfig `yaml:"cleaner"`
	Watcher WatcherConfig `yaml:"watcher"`
}

func NewConfig(filename string) (cfg *Config, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	cfg = &Config{}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return
	}

	// validators --
	if cfg.Logging.Component == "" {
		return nil, fmt.Errorf("logging.component can not be empty")
	}
	if cfg.Logging.Dir == "" {
		return nil, fmt.Errorf("logging.dir can not be empty")
	}

	if cfg.CheckpointDir == "" {
		return nil, fmt.Errorf("checkpoint_dir can not be empty")
	}
	if cfg.JournalDir == "" {
		return nil, fmt.Errorf("journal_dir can not be empty")
	}
	if cfg.LogDir == "" {
		return nil, fmt.Errorf("log_dir can not be empty")
	}

	if cfg.Hekad.MainConfPath == "" {
		return nil, fmt.Errorf("Hekad main_conf_path can not be empty")
	}
	if cfg.Hekad.BinPath == "" {
		return nil, fmt.Errorf("Hekad bin_path can not be empty")
	}
	if cfg.Hekad.ConfDir == "" {
		return nil, fmt.Errorf("Hekad conf_dir can not be empty")
	}

	cfg.Cleaner.Interval *= time.Second
	if cfg.Cleaner.Interval <= 0 {
		return nil, fmt.Errorf("Cleaner interval has to be positive value in"+
			" seconds, not %q", cfg.Cleaner.Interval)
	}

	cfg.Watcher.Interval *= time.Second
	if cfg.Watcher.Interval <= 0 {
		return nil, fmt.Errorf("Watcher interval has to be positive value in"+
			" seconds, not %q", cfg.Watcher.Interval)
	}

	return
}
