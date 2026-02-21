package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"

	"MyGame/core"
	"MyGame/game"
	"MyGame/utils"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("💥 Критическая ошибка: %v\n", r)
			fmt.Println("Игра будет закрыта. Нажмите Enter для выхода...")
			utils.WaitForEnter("Нажмите Enter для выхода...")
		}
	}()

	if err := runGame(ctx); err != nil {
		fmt.Printf("❌ Ошибка запуска игры: %v\n", err)
		os.Exit(1)
	}

	_ = utils.CloseConsoleWindow()
	os.Exit(0)
}

func runGame(ctx context.Context) error {
	utils.Info("Запуск игры It's Hard")

	fmt.Println("Настройте размер консоли (Alt+Enter — полноэкранный режим).")
	fmt.Println("Нажмите Enter для запуска игры.")
	fmt.Println()
	fmt.Scanln()
	utils.ClearInputBuffer()

	gameCore := core.NewCore()
	if err := gameCore.Initialize(); err != nil {
		return fmt.Errorf("инициализация игры: %w", err)
	}
	defer gameCore.Shutdown()

	p := tea.NewProgram(
		game.NewAppModel(gameCore),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInputTTY(),
		tea.WithFPS(60),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("запуск bubbletea: %w", err)
	}

	return nil
}
