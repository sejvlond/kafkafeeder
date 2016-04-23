package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
	"github.com/sejvlond/kafkalog-logrus"
)

type LOGGER logrus.FieldLogger

type KafkaFeeder struct {
	cfg          *Config
	lgr          LOGGER
	workerWG     sync.WaitGroup
	shutdownChan chan struct{}
	signalWG     sync.WaitGroup
	logManager   *LogManager
}

func (k *KafkaFeeder) restoreCheckpoints() (err error) {
	k.lgr.Infof("Restoring checkpoints")
	var cmd *exec.Cmd
	matches, err := filepath.Glob(k.cfg.CheckpointDir + "/*")
	if err != nil {
		return
	}
	for _, match := range matches {
		// TODO possible use shutil-go library
		cmd = exec.Command("cp", match, k.cfg.JournalDir)
		if err = cmd.Run(); err != nil {
			return
		}
	}
	return nil
}

func (k *KafkaFeeder) Start() {
	k.shutdownChan = make(chan struct{})
	var (
		err            error
		hekad          *Hekad
		watcher        *LogWatcher
		cleaner        *LogCleaner
		signal         *SignalHandler
		signalChan     = make(chan os.Signal)
		signalDispatch = make(map[os.Signal]SignalHandlerFunc)
	)

	// restore checkpoints
	if err := k.restoreCheckpoints(); err != nil {
		k.lgr.Errorf("Error copying checkpoints to journals %q", err)
		goto shutdown
	}

	// init log manager
	if k.logManager, err = NewLogManager(); err != nil {
		k.lgr.Infof("Log Manager initialization error: %q", err)
		goto shutdown
	}

	// init hekad
	hekad, err = NewHekad(k.lgr.WithField("name", "HEKAD"),
		&k.cfg.Hekad, k.logManager, k.shutdownChan, &k.workerWG,
		k.ShutDown)
	if err != nil {
		k.lgr.Infof("Hekad initialization error: %q", err)
		goto shutdown
	}
	k.workerWG.Add(1)
	go hekad.Run()

	// init signal handler
	signalDispatch[os.Interrupt] = k.ShutDown
	signalDispatch[os.Kill] = k.ShutDown
	signalDispatch[syscall.SIGTERM] = k.ShutDown
	signalDispatch[syscall.SIGUSR1] = hekad.Reload
	signalDispatch[syscall.SIGCHLD] = hekad.Check
	signal, err = NewSignalHandler(k.lgr.WithField("name", "SIGNAL HANDLER"),
		signalChan, &k.signalWG, signalDispatch)
	if err != nil {
		k.lgr.Infof("Signal handler initialization error: %q", err)
		goto shutdown
	}
	k.signalWG.Add(1)
	go signal.Run()

	// init log watcher
	watcher, err = NewLogWatcher(k.lgr.WithField("name", "WATCHER"),
		k.cfg.LogDir, &k.cfg.Watcher, k.logManager, hekad,
		k.shutdownChan, &k.workerWG)
	if err != nil {
		k.lgr.Infof("Watcher initialization error: %q", err)
		goto shutdown
	}
	k.workerWG.Add(1)
	go watcher.Run()

	// init log cleaner
	cleaner, err = NewLogCleaner(k.lgr.WithField("name", "CLEANER"),
		&k.cfg.Cleaner, k.shutdownChan, &k.workerWG)
	if err != nil {
		k.lgr.Infof("Cleaner initialization error: %q", err)
		goto shutdown
	}
	k.workerWG.Add(1)
	go cleaner.Run()

	goto stopping // validly here - skip shutdown
shutdown:
	k.ShutDown()
stopping:
	k.workerWG.Wait()
	close(signalChan)
	k.signalWG.Wait()
	return
}

func (k *KafkaFeeder) ShutDown() {
	k.lgr.Infof("Shutdown initialized")
	close(k.shutdownChan)
}

func (k *KafkaFeeder) Stop() {
	k.lgr.Infof("Bye")
}

func main() {
	lgr := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	usage := `kafkafeeder

Usage:
    kafkafeeder -c <config_file>
    kafkafeeder -h | --help

Options:
    -c --config         configuration file
    -h --help           Show this screen.`

	var err error
	args, err := docopt.Parse(usage, nil, true, "", false)
	if err != nil {
		lgr.Fatalf("Error parsing arguments %q", err)
	}

	var cfg *Config
	if useCfg := args["--config"].(bool); useCfg {
		configFile, ok := args["<config_file>"].(string)
		if !ok {
			lgr.Fatalf("Error config argument %q", err)
		}
		cfg, err = NewConfig(configFile)
		if err != nil {
			lgr.Fatalf("Error loading config file %q", err)
		}
	} else {
		lgr.Fatalf("Config was not loaded")
	}

	kafkalog_hook, err := kafkalog_logrus.NewKafkalogHook(
		cfg.Logging.Component, cfg.Logging.Interval, cfg.Logging.Dir)
	if err != nil {
		lgr.Fatalf("Could not create kafkalog hook: '%v'", err)
	}
	lgr.Hooks.Add(kafkalog_hook)

	app := KafkaFeeder{
		lgr: lgr.WithField("name", "MAIN"),
		cfg: cfg,
	}
	app.Start()
	app.Stop()
	return
}
