package utils

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

var (
	levelNames = map[LogLevel]string{
		LogLevelDebug:   "DEBUG",
		LogLevelInfo:    "INFO",
		LogLevelWarning: "WARN",
		LogLevelError:   "ERROR",
		LogLevelFatal:   "FATAL",
	}
	defaultLogger *Logger
	once          sync.Once
)

type Logger struct {
	mu      sync.RWMutex
	level   LogLevel
	enabled bool
	closed  bool
}

func GetDefaultLogger() *Logger {
	once.Do(func() { defaultLogger = NewLogger(true) })
	return defaultLogger
}

func NewLogger(enabled bool) *Logger {
	return &Logger{
		enabled: enabled,
		level:   LogLevelInfo,
	}
}

func (l *Logger) shouldSkipLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.enabled || l.closed || level < l.level
}

func (l *Logger) createLogEntry(levelName, message string) string {
	return fmt.Sprintf("[%s] [%s] %s", time.Now().Format("2006-01-02 15:04:05"), levelName, message)
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if l.shouldSkipLog(level) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	entry := l.createLogEntry(levelNames[level], fmt.Sprintf(format, args...))
	fmt.Println(entry)
}

func (l *Logger) Debug(format string, args ...interface{})   { l.log(LogLevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...interface{})    { l.log(LogLevelInfo, format, args...) }
func (l *Logger) Warning(format string, args ...interface{}) { l.log(LogLevelWarning, format, args...) }
func (l *Logger) Error(format string, args ...interface{})   { l.log(LogLevelError, format, args...) }
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelFatal, format, args...)
	l.Close()
	os.Exit(1)
}
func (l *Logger) GameEvent(eventType, details string) {
	l.Info("СОБЫТИЕ [%s]: %s", eventType, details)
}
func (l *Logger) BattleEvent(round int, attacker, defender, action string, damage int) {
	if damage > 0 {
		l.Info("БОЙ [раунд %d]: %s %s %s (урон: %d)", round, attacker, action, defender, damage)
	} else {
		l.Info("БОЙ [раунд %d]: %s %s %s", round, attacker, action, defender)
	}
}
func (l *Logger) InventoryEvent(action, itemName string, success bool) {
	status := "успешно"
	if !success {
		status = "неудачно"
	}
	l.Info("ИНВЕНТАРЬ: %s предмет '%s' - %s", action, itemName, status)
}
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	l.closed = true
}
func (l *Logger) Flush() error {
	return nil
}

func (l *Logger) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

func Debug(format string, args ...interface{})   { GetDefaultLogger().Debug(format, args...) }
func Info(format string, args ...interface{})    { GetDefaultLogger().Info(format, args...) }
func Warning(format string, args ...interface{}) { GetDefaultLogger().Warning(format, args...) }
func Error(format string, args ...interface{})   { GetDefaultLogger().Error(format, args...) }
func Fatal(format string, args ...interface{})   { GetDefaultLogger().Fatal(format, args...) }
func GameEvent(eventType, details string)        { GetDefaultLogger().GameEvent(eventType, details) }
func BattleEvent(round int, attacker, defender, action string, damage int) {
	GetDefaultLogger().BattleEvent(round, attacker, defender, action, damage)
}
func InventoryEvent(action, itemName string, success bool) {
	GetDefaultLogger().InventoryEvent(action, itemName, success)
}
func SetLogLevel(level LogLevel) { GetDefaultLogger().SetLevel(level) }
func CloseLogger()               { GetDefaultLogger().Close() }
func FlushLogs() error           { return GetDefaultLogger().Flush() }
