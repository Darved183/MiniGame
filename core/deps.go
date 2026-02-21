package core

import (
	"fmt"

	"MyGame/config"
	"MyGame/utils"
)

type Dependencies struct {
	Terminal *utils.TerminalManager
	Logger   *utils.Logger
	Config   *config.GameConfig
}

func NewDependencies(cfg *config.GameConfig) *Dependencies {
	if cfg == nil {
		cfg = config.Load()
	}
	log := utils.NewLogger(cfg.LoggingEnabled)
	if cfg.LoggingEnabled {
		log.SetLevel(parseLogLevel(cfg.LogLevel))
	}
	d := &Dependencies{
		Terminal: utils.NewTerminalManager(),
		Logger:   log,
		Config:   cfg,
	}
	_ = d.Validate()
	return d
}

func (d *Dependencies) Validate() error {
	if d == nil {
		return fmt.Errorf("deps is nil")
	}
	if d.Config == nil {
		d.Config = config.Load()
	}
	if err := d.Config.Validate(); err != nil {
		return err
	}
	if d.Terminal == nil {
		d.Terminal = utils.NewTerminalManager()
	}
	w, h := d.Terminal.GetSize()
	d.Config.ScreenWidth = w
	d.Config.ScreenHeight = h
	return nil
}

func (d *Dependencies) SafeShutdown() {
	if d == nil {
		return
	}
	if d.Logger != nil {
		d.Logger.Close()
	}
}

func (d *Dependencies) GetLogger() *utils.Logger {
	if d == nil {
		return nil
	}
	return d.Logger
}

func (d *Dependencies) GetTerminalManager() *utils.TerminalManager {
	if d == nil {
		return nil
	}
	return d.Terminal
}

func parseLogLevel(level string) utils.LogLevel {
	switch level {
	case "debug":
		return utils.LogLevelDebug
	case "info":
		return utils.LogLevelInfo
	case "warning":
		return utils.LogLevelWarning
	case "error":
		return utils.LogLevelError
	case "fatal":
		return utils.LogLevelFatal
	default:
		return utils.LogLevelInfo
	}
}
