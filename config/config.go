package config

import (
	"fmt"
	"time"
)

const (
	MinTerminalWidth  = 120
	MinTerminalHeight = 70
	MaxBattleRounds   = 10
	DefaultPlayerName = "Герой"
)

type GameConfig struct {
	ScreenWidth     int
	ScreenHeight    int
	BattleRounds    int
	TypewriterSpeed time.Duration

	AutoSave       bool
	Language       string
	Difficulty     string
	PlayerName     string
	LoggingEnabled bool
	LogLevel       string
}

func Load() *GameConfig {
	cfg := DefaultConfig()
	_ = cfg.Validate()
	return cfg
}

func DefaultConfig() *GameConfig {
	return &GameConfig{
		ScreenWidth:     MinTerminalWidth,
		ScreenHeight:    MinTerminalHeight,
		BattleRounds:    MaxBattleRounds,
		TypewriterSpeed: 30 * time.Millisecond,
		AutoSave:        false,
		Language:        "ru",
		Difficulty:      "normal",
		PlayerName:      DefaultPlayerName,
		LoggingEnabled:  false,
		LogLevel:        "info",
	}
}

func (c *GameConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Language == "" {
		c.Language = "ru"
	}
	if c.Difficulty == "" {
		c.Difficulty = "normal"
	}
	if c.PlayerName == "" {
		c.PlayerName = DefaultPlayerName
	}
	if c.BattleRounds <= 0 {
		c.BattleRounds = MaxBattleRounds
	}
	if c.ScreenWidth <= 0 {
		c.ScreenWidth = MinTerminalWidth
	}
	if c.ScreenHeight <= 0 {
		c.ScreenHeight = MinTerminalHeight
	}
	return nil
}
