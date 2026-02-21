package utils

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

type InputManager struct {
	mu      sync.Mutex
	scanner *bufio.Scanner
}

func NewInputManager() *InputManager {
	return &InputManager{scanner: bufio.NewScanner(os.Stdin)}
}

func (im *InputManager) WaitForEnter(message string) {
	if message == "" {
		message = "Нажмите Enter для продолжения..."
	}
	fmt.Print(message)
	im.mu.Lock()
	if im.scanner == nil {
		im.scanner = bufio.NewScanner(os.Stdin)
	}
	sc := im.scanner
	im.mu.Unlock()
	_ = sc.Scan()
}

func (im *InputManager) ClearInputBuffer() {
	im.mu.Lock()
	im.scanner = bufio.NewScanner(os.Stdin)
	im.mu.Unlock()
}

var defaultInputManager = NewInputManager()

func WaitForEnter(message string) { defaultInputManager.WaitForEnter(message) }
func ClearInputBuffer()           { defaultInputManager.ClearInputBuffer() }
func CloseInputManager()          {}
