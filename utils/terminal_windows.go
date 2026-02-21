package utils

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

type TerminalManager struct {
	width         int
	height        int
	lastUpdate    time.Time
	cacheDuration time.Duration
}

func NewTerminalManager() *TerminalManager {
	tm := &TerminalManager{
		cacheDuration: 500 * time.Millisecond,
	}
	tm.updateSize()
	return tm
}

func (tm *TerminalManager) GetSize() (int, int) {
	if time.Since(tm.lastUpdate) > tm.cacheDuration {
		tm.updateSize()
	}
	return tm.width, tm.height
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func (tm *TerminalManager) updateSize() {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 120, 30
		if cols := os.Getenv("COLUMNS"); cols != "" {
			if w, err := parseInt(cols); err == nil {
				width = w
			}
		}
		if rows := os.Getenv("LINES"); rows != "" {
			if h, err := parseInt(rows); err == nil {
				height = h
			}
		}
	}
	tm.width, tm.height = width, height
	tm.lastUpdate = time.Now()
}

func (tm *TerminalManager) IsFullscreen() bool {
	tm.GetSize()
	return tm.width >= 100 && tm.height >= 30
}

func GetTerminalSize() (int, int) {
	tm := NewTerminalManager()
	return tm.GetSize()
}

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleMode           = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode           = kernel32.NewProc("SetConsoleMode")
	procGetConsoleCursorInfo     = kernel32.NewProc("GetConsoleCursorInfo")
	procSetConsoleCursorInfo     = kernel32.NewProc("SetConsoleCursorInfo")
	procGetCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procShowWindow               = user32.NewProc("ShowWindow")
	procGetWindowLong            = user32.NewProc("GetWindowLongW")
	procSetWindowLong            = user32.NewProc("SetWindowLongW")
	procSendMessage              = user32.NewProc("SendMessageW")
	procShowScrollBar            = user32.NewProc("ShowScrollBar")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetActiveWindow          = user32.NewProc("SetActiveWindow")
	procSetFocus                 = user32.NewProc("SetFocus")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetCurrentConsoleFontEx  = kernel32.NewProc("GetCurrentConsoleFontEx")
	procSetCurrentConsoleFontEx  = kernel32.NewProc("SetCurrentConsoleFontEx")
	procSetConsoleOutputCP       = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP             = kernel32.NewProc("SetConsoleCP")
)

const (
	cpUTF8            = 65001
	SW_MAXIMIZE       = 3
	SW_SHOWMAXIMIZED  = 3
	SW_RESTORE        = 9
	SM_CXSCREEN       = 0
	SM_CYSCREEN       = 1
	WS_CAPTION        = uintptr(0x00C00000)
	WS_THICKFRAME     = uintptr(0x00040000)
	WS_SYSMENU        = uintptr(0x00080000)
	WS_MINIMIZEBOX    = uintptr(0x00020000)
	WS_MAXIMIZEBOX    = uintptr(0x00010000)
	WS_HSCROLL        = uintptr(0x00100000)
	WS_VSCROLL        = uintptr(0x00200000)
	SWP_FRAMECHANGED  = 0x0020
	SWP_SHOWWINDOW    = 0x0040
	SWP_NOMOVE        = 0x0002
	SWP_NOSIZE        = 0x0001
	SWP_NOZORDER      = 0x0004
	HWND_TOP          = ^uintptr(0)
	WM_SYSCOMMAND     = 0x0112
	WM_CLOSE          = 0x0010
	SC_MAXIMIZE       = 0xF030
	ENABLE_QUICK_EDIT = 0x0040
	SB_BOTH           = 3
	SB_HORZ           = 0
	SB_VERT           = 1
)

var isFullscreen = false

func GetConsoleWindow() uintptr {
	ret, _, _ := procGetConsoleWindow.Call()
	return ret
}

func CloseConsoleWindow() error {
	hwnd := GetConsoleWindow()
	if hwnd == 0 || procSendMessage == nil {
		return nil
	}
	procSendMessage.Call(hwnd, WM_CLOSE, 0, 0)
	return nil
}

func hideScrollbar(hwnd uintptr, bar int) {
	if hwnd == 0 || procShowScrollBar == nil {
		return
	}
	procShowScrollBar.Call(hwnd, uintptr(bar), 0)
}

func HideScrollBars() {
	hwnd := GetConsoleWindow()
	hideScrollbar(hwnd, SB_BOTH)
	hideScrollbar(hwnd, SB_HORZ)
	hideScrollbar(hwnd, SB_VERT)
}

func HideCursors() {
	for i := 0; i < 3; i++ {
		hideConsoleCursor()
	}
}

func hideConsoleCursor() {
	if procGetConsoleCursorInfo == nil || procSetConsoleCursorInfo == nil {
		return
	}
	stdoutHandle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil || stdoutHandle == windows.InvalidHandle {
		return
	}
	type CONSOLE_CURSOR_INFO struct {
		Size    uint32
		Visible int32
	}
	var cursorInfo CONSOLE_CURSOR_INFO
	procGetConsoleCursorInfo.Call(uintptr(stdoutHandle), uintptr(unsafe.Pointer(&cursorInfo)))
	cursorInfo.Visible = 0
	cursorInfo.Size = 1
	procSetConsoleCursorInfo.Call(uintptr(stdoutHandle), uintptr(unsafe.Pointer(&cursorInfo)))
	os.Stdout.WriteString("\033[?25l\x1b[?25l")
	_ = os.Stdout.Sync()
}

func SetWindowFocus(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	if procAttachThreadInput != nil && procGetWindowThreadProcessId != nil && procGetCurrentThreadId != nil {
		var processId uint32
		foregroundThreadId, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processId)))
		currentThreadId, _, _ := procGetCurrentThreadId.Call()
		if foregroundThreadId != 0 && currentThreadId != 0 && foregroundThreadId != currentThreadId {
			procAttachThreadInput.Call(currentThreadId, foregroundThreadId, 1)
			if procSetForegroundWindow != nil {
				procSetForegroundWindow.Call(hwnd)
			}
			if procSetActiveWindow != nil {
				procSetActiveWindow.Call(hwnd)
			}
			procAttachThreadInput.Call(currentThreadId, foregroundThreadId, 0)
		}
	}
	if procBringWindowToTop != nil {
		procBringWindowToTop.Call(hwnd)
	}
	if procSetActiveWindow != nil {
		procSetActiveWindow.Call(hwnd)
	}
	if procSetFocus != nil {
		procSetFocus.Call(hwnd)
	}
	if procSetForegroundWindow != nil {
		procSetForegroundWindow.Call(hwnd)
	}
}

func getScreenSize() (cx, cy uintptr) {
	if procGetSystemMetrics == nil {
		return 0, 0
	}
	cx, _, _ = procGetSystemMetrics.Call(SM_CXSCREEN)
	cy, _, _ = procGetSystemMetrics.Call(SM_CYSCREEN)
	return
}

func SetFullscreen() error {
	if user32 == nil || kernel32 == nil {
		return nil
	}
	hwnd := GetConsoleWindow()
	if hwnd == 0 {
		return nil
	}
	cxScreen, cyScreen := getScreenSize()
	if cxScreen == 0 || cyScreen == 0 {
		return nil
	}
	HideScrollBars()
	if procGetWindowLong != nil && procSetWindowLong != nil {
		gwlStyle := int32(-16)
		style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlStyle))
		removeStyle := WS_CAPTION | WS_THICKFRAME | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_HSCROLL | WS_VSCROLL
		procSetWindowLong.Call(hwnd, uintptr(gwlStyle), style&^removeStyle)
		if procSetWindowPos != nil {
			procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED)
		}
	}
	if procSetWindowPos != nil {
		procSetWindowPos.Call(hwnd, HWND_TOP, 0, 0, cxScreen, cyScreen, SWP_SHOWWINDOW|SWP_FRAMECHANGED)
	}
	if procShowWindow != nil {
		procShowWindow.Call(hwnd, SW_SHOWMAXIMIZED)
	}
	if procSendMessage != nil {
		procSendMessage.Call(hwnd, WM_SYSCOMMAND, SC_MAXIMIZE, 0)
	}
	HideScrollBars()
	DisableQuickEditMode()
	HideCursors()
	isFullscreen = true
	go func() {
		for _, d := range []time.Duration{50, 100, 150} {
			time.Sleep(d * time.Millisecond)
			HideScrollBars()
		}
	}()
	return nil
}

func SetWindowed() error {
	if user32 == nil || kernel32 == nil {
		return nil
	}
	hwnd := GetConsoleWindow()
	if hwnd == 0 {
		return nil
	}
	if procShowWindow != nil {
		procShowWindow.Call(hwnd, SW_RESTORE)
	}
	if procGetWindowLong != nil && procSetWindowLong != nil {
		gwlStyle := int32(-16)
		style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlStyle))
		addStyle := WS_CAPTION | WS_THICKFRAME | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX
		procSetWindowLong.Call(hwnd, uintptr(gwlStyle), (style|addStyle)&^(WS_HSCROLL|WS_VSCROLL))
		if procSetWindowPos != nil {
			procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED)
		}
	}
	HideScrollBars()
	HideCursors()
	isFullscreen = false
	return nil
}

func ToggleFullscreen() error {
	if isFullscreen {
		return SetWindowed()
	}
	return SetFullscreen()
}

func IsFullscreen() bool {
	return isFullscreen
}

func MaximizeWindow() error {
	if user32 == nil || procShowWindow == nil {
		return nil
	}
	hwnd := GetConsoleWindow()
	if hwnd == 0 {
		return nil
	}
	if err := SetFullscreen(); err == nil {
		return nil
	}
	procShowWindow.Call(hwnd, SW_MAXIMIZE)
	if procGetSystemMetrics != nil && procSetWindowPos != nil {
		cxScreen, cyScreen := getScreenSize()
		if cxScreen != 0 && cyScreen != 0 {
			procSetWindowPos.Call(hwnd, HWND_TOP, 0, 0, cxScreen, cyScreen, SWP_SHOWWINDOW)
		}
	}
	return nil
}

func DisableQuickEditMode() {
	if procGetConsoleMode == nil || procSetConsoleMode == nil {
		return
	}
	stdinHandle, _ := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if stdinHandle == windows.InvalidHandle {
		return
	}
	var mode uint32
	procGetConsoleMode.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&mode)))
	mode &^= ENABLE_QUICK_EDIT
	procSetConsoleMode.Call(uintptr(stdinHandle), uintptr(mode))
}

func SetConsoleFontSize(size uint16) error {
	if procGetCurrentConsoleFontEx == nil || procSetCurrentConsoleFontEx == nil {
		return nil
	}
	stdoutHandle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil || stdoutHandle == windows.InvalidHandle {
		return err
	}
	type COORD struct{ X, Y int16 }
	type CONSOLE_FONT_INFOEX struct {
		cbSize     uint32
		nFont      uint32
		dwFontSize COORD
		FontFamily uint32
		FontWeight uint32
		FaceName   [32]uint16
	}
	var fontInfo CONSOLE_FONT_INFOEX
	fontInfo.cbSize = uint32(unsafe.Sizeof(fontInfo))
	procGetCurrentConsoleFontEx.Call(uintptr(stdoutHandle), 0, uintptr(unsafe.Pointer(&fontInfo)))
	fontInfo.dwFontSize.X, fontInfo.dwFontSize.Y = int16(size), int16(size)
	procSetCurrentConsoleFontEx.Call(uintptr(stdoutHandle), 0, uintptr(unsafe.Pointer(&fontInfo)))
	return nil
}

func SetUTF8CodePage() error {
	if procSetConsoleOutputCP == nil || procSetConsoleCP == nil {
		return nil
	}
	r1, _, e1 := procSetConsoleOutputCP.Call(uintptr(cpUTF8))
	if r1 == 0 && e1 != windows.ERROR_SUCCESS {
		return fmt.Errorf("SetConsoleOutputCP(%d): %w", cpUTF8, e1)
	}
	r2, _, e2 := procSetConsoleCP.Call(uintptr(cpUTF8))
	if r2 == 0 && e2 != windows.ERROR_SUCCESS {
		return fmt.Errorf("SetConsoleCP(%d): %w", cpUTF8, e2)
	}
	return nil
}
