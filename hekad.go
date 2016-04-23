package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type hekadOutputCatcher struct {
	lgr LOGGER
	err bool
}

func (o *hekadOutputCatcher) Write(data []byte) (int, error) {
	for _, msg := range strings.Split(string(data), "\n") {
		if len(msg) > 0 {
			if o.err {
				o.lgr.Warnf("%s", msg)
			} else {
				o.lgr.Infof("%s", msg)
			}
		}
	}
	return len(data), nil
}

type HekadCmd struct {
	cmd             *exec.Cmd
	lgr             LOGGER
	bin, cfg        string
	ShouldBeRunning bool
}

func NewHekadCmd(lgr LOGGER, bin, cfg string) (*HekadCmd, error) {
	hekad := &HekadCmd{
		lgr:             lgr,
		bin:             bin,
		cfg:             cfg,
		ShouldBeRunning: false,
	}
	hekad.initCmd()
	return hekad, nil
}

func (h *HekadCmd) initCmd() {
	h.cmd = exec.Command(h.bin, "-config", h.cfg)
	h.cmd.Stdout = &hekadOutputCatcher{
		lgr: h.lgr.WithField("hekad", "stdout"),
		err: false,
	}
	h.cmd.Stderr = &hekadOutputCatcher{
		lgr: h.lgr.WithField("hekad", "stderr"),
		err: true,
	}
}

func (h *HekadCmd) Start() bool {
	err := h.cmd.Start()
	if err != nil {
		h.lgr.Errorf("Error starting hekad process %q", err)
	}
	h.ShouldBeRunning = true
	return err == nil
}

func (h *HekadCmd) Stop() {
	h.ShouldBeRunning = false
	err := h.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		h.lgr.Errorf("Error killing hekad process %q", err)
		h.cmd.Process.Kill()
	}
	err = h.cmd.Wait()
	if err != nil {
		h.lgr.Errorf("Error stopping hekad process %q", err)
	}
}

func (h *HekadCmd) isOk() bool {
	if h.ShouldBeRunning == false {
		return true
	}
	return h.cmd != nil && h.cmd.Process != nil && h.cmd.ProcessState != nil &&
		!h.cmd.ProcessState.Exited()
}

func (h *HekadCmd) Reload() bool {
	/* TODO when hekad will implement SIGHUP for configs reload, uncoment this
	if err := h.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		h.lgr.Errorf("Error sending HUP signal to hekad process %q", err)
		return false
	}
	h.lgr.Infof("Reload signal sent to hekad proces")
	*/
	h.Stop()
	h.initCmd()
	if ok := h.Start(); !ok {
		h.lgr.Errorf("Error starting hekad process")
		return false
	}
	h.lgr.Infof("Hekad process reloaded")
	return true
}

func (h *HekadCmd) Status() bool {
	if err := h.cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		h.lgr.Errorf("Error sending USR1 signal to hekad process %q", err)
		return false
	}
	h.lgr.Infof("Status signal sent to heka proces")
	return true
}

type Hekad struct {
	lgr          LOGGER
	wg           *sync.WaitGroup
	shutdownChan chan struct{}
	logManager   *LogManager
	cfg          *HekadConfig
	CallShutDown func()
	converter    *Converter
	hekad        *HekadCmd
}

func NewHekad(lgr LOGGER, cfg *HekadConfig, logManager *LogManager,
	shutdownChan chan struct{}, wg *sync.WaitGroup, shutDownFunc func()) (
	*Hekad, error) {

	converter, err := NewConverter(cfg.KafkaBrokers)
	if err != nil {
		return nil, fmt.Errorf("Error initializing converter %q", err)
	}
	hekadCmd, err := NewHekadCmd(lgr, cfg.BinPath, cfg.ConfDir)
	if err != nil {
		return nil, fmt.Errorf("Error initializing hekaCmd %q", err)
	}
	hekad := &Hekad{
		lgr:          lgr,
		cfg:          cfg,
		logManager:   logManager,
		wg:           wg,
		shutdownChan: shutdownChan,
		CallShutDown: shutDownFunc,
		converter:    converter,
		hekad:        hekadCmd,
	}
	if err = hekad.resetConf(); err != nil {
		return nil, fmt.Errorf("Error prepareing conf dir %q", err)
	}
	return hekad, nil
}

func (h *Hekad) resetConf() error {
	matches, err := filepath.Glob(h.cfg.ConfDir + "/*")
	if err != nil {
		return err
	}
	for _, match := range matches {
		if err := os.RemoveAll(match); err != nil {
			return err
		}
	}
	h.lgr.Infof("Conf dir %q clean", h.cfg.ConfDir)

	err = os.Symlink(h.cfg.MainConfPath, h.cfg.ConfDir+"/hekad.toml")
	return err
}

func (h *Hekad) Check() {
	if !h.hekad.isOk() {
		h.lgr.Errorf("OMG hekad process exited prematurely")
		h.CallShutDown()
	}
}

func (h *Hekad) Reload() {
	h.lgr.Infof("Request to reload configuration accepted")
	if err := h.resetConf(); err != nil {
		h.lgr.Errorf("Error reseting old configurations %q", err)
		h.CallShutDown()
		return
	}
	var (
		file *os.File
		err  error
	)
	for path, log := range h.logManager.Logs {
		file, err = os.Create(filepath.Join(
			h.cfg.ConfDir, IdFromString(path)+".toml"))
		if err != nil {
			h.lgr.Errorf("Error creating converted file %q", err)
		}
		err = h.converter.Convert(log, file)
		if err != nil {
			h.lgr.Errorf("Error converting file %q: %q", path, err)
		}
	}

	if ok := h.hekad.Reload(); !ok {
		h.lgr.Errorf("Error reloading hekad - shuting down")
		h.CallShutDown()
	} else {
		h.lgr.Debugf("Loaded logs:")
		for file, log := range h.logManager.Logs {
			h.lgr.Debugf("%v => %v (%+v)", file, log.Directory, log.Topics)
		}
	}
	return
}

func (h *Hekad) Run() {
	h.lgr.Infof("started")
	if h.hekad.Start() {
		run := true
		for run {
			select {
			case <-h.shutdownChan:
				h.lgr.Infof("shutdown accepted")
				run = false
				break
			}
		}
		h.hekad.Stop()
	}
	h.wg.Done()
	h.lgr.Infof("stopped")
}
