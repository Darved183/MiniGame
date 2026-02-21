package core

import (
	"fmt"
	"time"

	"MyGame/Struct/Character"
	"MyGame/config"
)

type Core struct {
	GameManager         Game
	ExtendedGameManager *ExtendedGameManager
	Config              *config.GameConfig
	Deps                *Dependencies
}

func NewCore() *Core {
	extendedManager := NewExtendedGameManager()

	return &Core{
		GameManager:         extendedManager,
		ExtendedGameManager: extendedManager,
		Config:              extendedManager.Config,
		Deps:                extendedManager.Deps,
	}
}

func (c *Core) Initialize() error {
	if c.Deps == nil {
		return WrapError(ErrDepsNotReady, nil, map[string]interface{}{
			"reason": "Deps is nil in Core.Initialize",
		})
	}

	if err := c.Deps.Validate(); err != nil {
		return WrapError(ErrDepsNotReady, err, map[string]interface{}{
			"stage": "Deps.Validate",
		})
	}

	if err := c.GameManager.Start(); err != nil {
		return fmt.Errorf("ошибка запуска игры: %w", err)
	}

	if c.Deps.Logger != nil {
		c.Deps.Logger.Info("Ядро игры инициализировано")
	}

	return nil
}

func (c *Core) Shutdown() {
	if c.ExtendedGameManager != nil {
		c.ExtendedGameManager.SafeShutdown()
	}

	if c.Deps != nil && c.Deps.Logger != nil {
		c.Deps.Logger.Info("Ядро игры завершило работу")
	}
}

type CoreError struct {
	Code    string
	Message string
	Context map[string]interface{}
	Err     error
}

func (e *CoreError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CoreError) Unwrap() error {
	return e.Err
}

var (
	ErrDepsNotReady = &CoreError{Code: "DEPS_NOT_READY", Message: "зависимости не готовы"}
)

func WrapError(baseError *CoreError, err error, context map[string]interface{}) error {
	return &CoreError{
		Code:    baseError.Code,
		Message: baseError.Message,
		Context: context,
		Err:     err,
	}
}

type GameState int

const (
	StateMenu GameState = iota
	StateBattle
	StateInventory
	StateEULA
	StateSettings
	StateStory
	StateLoading
	StateGameOver
	StateVictory
	StatePaused
)

type GameEventType int

const (
	EventBattleStart GameEventType = iota
	EventBattleEnd
	EventPlayerLevelUp
	EventItemAcquired
	EventItemUsed
	EventCharacterDeath
	EventGameSave
	EventGameLoad
	EventStateChange
)

func (gs GameState) String() string {
	names := []string{
		"Меню", "Бой", "Инвентарь", "Лицензия", "Настройки",
		"Сюжет", "Загрузка", "Конец игры", "Победа", "Пауза",
	}
	if int(gs) < len(names) {
		return names[gs]
	}
	return "Неизвестно"
}

func (get GameEventType) String() string {
	names := []string{
		"Начало боя", "Конец боя", "Повышение уровня", "Получение предмета",
		"Использование предмета", "Смерть персонажа", "Сохранение игры",
		"Загрузка игры", "Смена состояния",
	}
	if int(get) < len(names) {
		return names[get]
	}
	return "Неизвестно"
}

func IsValidGameState(state GameState) bool { return state >= StateMenu && state <= StatePaused }
func IsCombatState(state GameState) bool    { return state == StateBattle }

func CanSaveInState(state GameState) bool {
	unsavableStates := []GameState{StateBattle, StateLoading, StatePaused}
	for _, s := range unsavableStates {
		if state == s {
			return false
		}
	}
	return true
}

type GameEvent struct {
	Type      GameEventType
	Data      interface{}
	Timestamp int64
	Source    string
}

func NewGameEvent(eventType GameEventType, data interface{}, source string) *GameEvent {
	return &GameEvent{Type: eventType, Data: data, Timestamp: time.Now().Unix(), Source: source}
}

type Game interface {
	Start() error
	Stop()
	GetState() GameState
	SetState(state GameState)
	IsRunning() bool
	IsValidState() bool
	GetPlayer() *Character.Character
	SetPlayer(player *Character.Character)
	GetPlayTime() time.Duration
	GetFormattedPlayTime() string
	Pause()
	Resume()
}
