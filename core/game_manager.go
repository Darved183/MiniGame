package core

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"MyGame/Struct/Character"
	"MyGame/config"
	"MyGame/utils"
)

type InternalGameConfig struct {
	AutoSave         bool
	AutoSaveInterval time.Duration
	MaxStateHistory  int
	EnableEvents     bool
}

type GameManager struct {
	mu            sync.RWMutex
	State         GameState
	Player        *Character.Character
	running       bool
	StartTime     time.Time
	PlayTime      time.Duration
	eventHandlers map[GameEventType][]func(*GameEvent)
	stateHistory  []GameState
	config        *InternalGameConfig
}

func NewGameManager() *GameManager {
	return &GameManager{
		State:         StateMenu,
		running:       false,
		StartTime:     time.Now(),
		PlayTime:      0,
		eventHandlers: make(map[GameEventType][]func(*GameEvent)),
		stateHistory:  make([]GameState, 0),
		config: &InternalGameConfig{
			AutoSave:         true,
			AutoSaveInterval: 5 * time.Minute,
			MaxStateHistory:  100,
			EnableEvents:     true,
		},
	}
}

func (gm *GameManager) Start() error {
	gm.mu.Lock()
	gm.running = true
	gm.StartTime = time.Now()
	gm.State = StateMenu
	gm.mu.Unlock()
	gm.emitEvent(EventStateChange, StateMenu, "GameManager")
	utils.Info("Игра запущена")
	return nil
}

func (gm *GameManager) Stop() {
	gm.mu.Lock()
	gm.running = false
	gm.PlayTime = time.Since(gm.StartTime)
	gm.mu.Unlock()
	gm.emitEvent(EventStateChange, StateGameOver, "GameManager")
	utils.Info("Игра остановлена. Время игры: %v", gm.PlayTime)
}

func (gm *GameManager) GetState() GameState {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.State
}

func (gm *GameManager) SetState(state GameState) {
	if !IsValidGameState(state) {
		utils.Warning("Попытка установить невалидное состояние игры: %d", state)
		return
	}
	gm.mu.Lock()
	if gm.State == state {
		gm.mu.Unlock()
		utils.Debug("Попытка повторно установить то же состояние игры: %s", state)
		return
	}
	oldState := gm.State
	gm.State = state
	gm.stateHistory = append(gm.stateHistory, oldState)
	if gm.config != nil && len(gm.stateHistory) > gm.config.MaxStateHistory {
		gm.stateHistory = gm.stateHistory[1:]
	}
	gm.mu.Unlock()
	gm.emitEvent(EventStateChange, state, "GameManager")
	utils.Debug("Смена состояния игры: %s -> %s", oldState, state)
}

func (gm *GameManager) IsRunning() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.running
}

func (gm *GameManager) IsValidState() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return IsValidGameState(gm.State)
}

func (gm *GameManager) SetPlayer(player *Character.Character) {
	gm.mu.Lock()
	gm.Player = player
	gm.mu.Unlock()
	if player != nil {
		utils.Info("Установлен игрок: %s", player.GetName())
	}
}

func (gm *GameManager) GetPlayer() *Character.Character {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.Player
}

func (gm *GameManager) GetPlayTime() time.Duration {
	gm.mu.RLock()
	running := gm.running
	start := gm.StartTime
	playTime := gm.PlayTime
	gm.mu.RUnlock()
	if running {
		return time.Since(start)
	}
	return playTime
}

func (gm *GameManager) GetFormattedPlayTime() string {
	total := gm.GetPlayTime()
	hours := int(total.Hours())
	minutes := int(total.Minutes()) % 60
	seconds := int(total.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func (gm *GameManager) RegisterEventHandler(eventType GameEventType, handler func(*GameEvent)) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	if gm.config != nil && gm.config.EnableEvents {
		gm.eventHandlers[eventType] = append(gm.eventHandlers[eventType], handler)
	}
}

func (gm *GameManager) emitEvent(eventType GameEventType, data interface{}, source string) {
	gm.mu.RLock()
	if gm.config == nil || !gm.config.EnableEvents {
		gm.mu.RUnlock()
		return
	}
	handlers := make([]func(*GameEvent), len(gm.eventHandlers[eventType]))
	copy(handlers, gm.eventHandlers[eventType])
	gm.mu.RUnlock()

	event := NewGameEvent(eventType, data, source)
	for _, h := range handlers {
		func(handler func(*GameEvent)) {
			defer func() {
				if r := recover(); r != nil {
					utils.Error("Паника в обработчике события %s: %v", eventType, r)
				}
			}()
			handler(event)
		}(h)
	}
}

func (gm *GameManager) GetStateHistory() []GameState {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	if len(gm.stateHistory) == 0 {
		return nil
	}
	cp := make([]GameState, len(gm.stateHistory))
	copy(cp, gm.stateHistory)
	return cp
}

func (gm *GameManager) CanSave() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.running && CanSaveInState(gm.State) && gm.Player != nil
}

type GameSaveDTO struct {
	PlayerName         string    `json:"player_name"`
	PlayerCurrentHP    int       `json:"player_current_hp"`
	PlayerMaxHP        int       `json:"player_max_hp"`
	PlayerStrength     int       `json:"player_strength"`
	PlayerAgility      int       `json:"player_agility"`
	PlayerIntelligence int       `json:"player_intelligence"`
	GameState          GameState `json:"game_state"`
	PlayTimeNs         int64     `json:"play_time_ns"`
	SaveTime           string    `json:"save_time"`
	Version            string    `json:"version"`
}

func (gm *GameManager) Save() ([]byte, error) {
	if !gm.CanSave() {
		gm.mu.RLock()
		state := gm.State
		gm.mu.RUnlock()
		return nil, fmt.Errorf("невозможно сохранить игру в текущем состоянии: %s", state)
	}
	gm.mu.RLock()
	player := gm.Player
	state := gm.State
	gm.mu.RUnlock()
	if player == nil {
		return nil, fmt.Errorf("игрок не установлен")
	}
	playTime := gm.GetPlayTime()
	saveTime := time.Now()
	dto := &GameSaveDTO{
		PlayerName:         player.GetName(),
		PlayerCurrentHP:    player.GetHP(),
		PlayerMaxHP:        player.GetMaxHP(),
		PlayerStrength:     player.GetStrength(),
		PlayerAgility:      player.GetAgility(),
		PlayerIntelligence: player.GetIntelligence(),
		GameState:          state,
		PlayTimeNs:         int64(playTime),
		SaveTime:           saveTime.Format(time.RFC3339),
		Version:            "1.0.0",
	}
	data, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации данных сохранения: %w", err)
	}
	gm.emitEvent(EventGameSave, dto, "GameManager")
	utils.Info("Игра сохранена")
	return data, nil
}

func (gm *GameManager) Load(data []byte) error {
	var dto GameSaveDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return fmt.Errorf("ошибка десериализации данных сохранения: %w", err)
	}
	player, err := Character.New(dto.PlayerName, dto.PlayerMaxHP, dto.PlayerStrength, dto.PlayerAgility, dto.PlayerIntelligence)
	if err != nil {
		return fmt.Errorf("создание персонажа при загрузке: %w", err)
	}
	player.CurrentHP = dto.PlayerCurrentHP
	if player.CurrentHP > player.MaxHP {
		player.CurrentHP = player.MaxHP
	}
	playTime := time.Duration(dto.PlayTimeNs)
	gm.mu.Lock()
	gm.Player = player
	gm.State = dto.GameState
	gm.StartTime = time.Now().Add(-playTime)
	gm.PlayTime = playTime
	gm.mu.Unlock()
	gm.emitEvent(EventGameLoad, &dto, "GameManager")
	utils.Info("Игра загружена")
	return nil
}

func (gm *GameManager) Pause() {
	if gm.State != StatePaused {
		gm.SetState(StatePaused)
		utils.Info("Игра приостановлена")
	}
}

func (gm *GameManager) Resume() {
	gm.mu.Lock()
	if gm.State != StatePaused {
		gm.mu.Unlock()
		return
	}
	var previousState GameState
	if len(gm.stateHistory) > 0 {
		previousState = gm.stateHistory[len(gm.stateHistory)-1]
		gm.stateHistory = gm.stateHistory[:len(gm.stateHistory)-1]
	} else {
		previousState = StateMenu
	}
	gm.mu.Unlock()
	if !IsValidGameState(previousState) {
		previousState = StateMenu
	}
	gm.mu.Lock()
	gm.State = previousState
	gm.mu.Unlock()
	gm.emitEvent(EventStateChange, previousState, "GameManager")
	utils.Info("Игра возобновлена")
}

func (gm *GameManager) GetInternalConfig() *InternalGameConfig {
	return gm.config
}

func (gm *GameManager) SetInternalConfig(cfg *InternalGameConfig) {
	if cfg != nil {
		gm.config = cfg
	}
}

func (gm *GameManager) GetSaveID() string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	if gm.Player != nil {
		return fmt.Sprintf("save_%s_%d", gm.Player.GetName(), gm.StartTime.Unix())
	}
	return fmt.Sprintf("save_%d", gm.StartTime.Unix())
}

func (gm *GameManager) GetSaveVersion() int {
	return 1
}

type ExtendedGameManager struct {
	*GameManager

	Config *config.GameConfig
	Deps   *Dependencies
}

func NewExtendedGameManager() *ExtendedGameManager {
	cfg := config.Load()
	deps := NewDependencies(cfg)

	gm := &ExtendedGameManager{
		GameManager: NewGameManager(),
		Config:      cfg,
		Deps:        deps,
	}

	player, err := Character.NewWarrior(config.DefaultPlayerName)
	if err != nil {
		if gm.Deps != nil && gm.Deps.Logger != nil {
			gm.Deps.Logger.Error("Ошибка создания игрока: %v", err)
		}
		player, err = Character.NewWarrior("Герой")
		if err != nil && gm.Deps != nil && gm.Deps.Logger != nil {
			gm.Deps.Logger.Error("Ошибка создания резервного игрока: %v", err)
		}
	}

	gm.GameManager.SetPlayer(player)
	gm.registerEventHandlers()
	return gm
}

func (gm *ExtendedGameManager) UpdatePlayer(player *Character.Character) {
	gm.GameManager.SetPlayer(player)
	if gm.Deps != nil && gm.Deps.Logger != nil && player != nil {
		gm.Deps.Logger.Info("Игрок обновлен: %s", player.GetName())
	}
}

func (gm *ExtendedGameManager) GetPlayerStats() map[string]interface{} {
	player := gm.GetPlayer()
	if player == nil {
		return map[string]interface{}{"error": "Игрок не установлен"}
	}
	return map[string]interface{}{
		"name":      player.GetName(),
		"hp":        player.GetHP(),
		"max_hp":    player.GetMaxHP(),
		"strength":  player.GetStrength(),
		"attack":    player.GetAttack(),
		"defense":   player.GetDefense(),
		"play_time": gm.GetFormattedPlayTime(),
	}
}

func (gm *ExtendedGameManager) registerEventHandlers() {
	gm.RegisterEventHandler(EventStateChange, func(event *GameEvent) {
		if gm.Deps != nil && gm.Deps.Logger != nil {
			gm.Deps.Logger.Info("Смена состояния игры: %v", event.Data)
		}
	})
	gm.RegisterEventHandler(EventGameSave, func(*GameEvent) {
		if gm.Deps != nil && gm.Deps.Logger != nil {
			gm.Deps.Logger.Info("Игра сохранена")
		}
	})
	gm.RegisterEventHandler(EventGameLoad, func(*GameEvent) {
		if gm.Deps != nil && gm.Deps.Logger != nil {
			gm.Deps.Logger.Info("Игра загружена")
		}
	})
}

func (gm *ExtendedGameManager) GetConfig() *config.GameConfig { return gm.Config }

func (gm *ExtendedGameManager) ValidateConfig() error {
	if gm.Config == nil {
		return fmt.Errorf("конфигурация не установлена")
	}
	return gm.Config.Validate()
}

func (gm *ExtendedGameManager) SafeShutdown() {
	if gm.Deps != nil {
		gm.Deps.SafeShutdown()
	}
	if gm.GameManager != nil {
		gm.GameManager.Stop()
	}
}
