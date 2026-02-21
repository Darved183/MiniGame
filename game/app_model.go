package game

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MyGame/core"
	"MyGame/game/ui"
	"MyGame/sound"
	"MyGame/utils"
)

type AppModel struct {
	gameCore        *core.Core
	currentView     ViewType
	mainMenu        *MainMenuModel
	eulaModel       *EULAModel
	fullEulaModel   *FullEULAModel
	fightModel      *FightModel
	chatModel       *ChatModel
	pvpConnectModel *PvPConnectModel
	pvpFightModel   *PvPFightModel
	quitting        bool
	width           int
	height          int
	eulaAccepted    bool
}

func NewAppModel(gameCore *core.Core) AppModel {
	initialView := ViewMainMenu
	var fullEulaModel *FullEULAModel

	if !SkipEULA {
		initialView = ViewFullEULA
		fullEulaModel = NewFullEULAModel(gameCore.ExtendedGameManager)
		if fullEulaModel != nil {
			fullEulaModel.Width, fullEulaModel.Height = ui.MinWidth, ui.MinHeight
		}
	}

	return AppModel{
		gameCore:      gameCore,
		currentView:   initialView,
		mainMenu:      NewMainMenuModel(gameCore),
		fullEulaModel: fullEulaModel,
		width:         ui.MinWidth,
		height:        ui.MinHeight,
		eulaAccepted:  SkipEULA,
	}
}

func (m AppModel) Init() tea.Cmd {
	hideCursorCmd := tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		utils.HideCursors()
		return nil
	})

	if m.fullEulaModel != nil {
		return tea.Sequence(hideCursorCmd, m.fullEulaModel.Init())
	}
	return hideCursorCmd
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKeyInput(msg)
	case ViewChangeMsg:
		return m.handleViewChange(msg)
	case PvPConnectedMsg:
		return m.handlePvPConnected(msg)
	case QuitMsg:
		m.quitting = true
		if m.gameCore != nil {
			m.gameCore.Shutdown()
		}
		return m, tea.Quit
	}
	return m.delegateToCurrentView(msg)
}

func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.currentView {
	case ViewMainMenu:
		if m.mainMenu != nil {
			content = m.mainMenu.View()
		}
	case ViewFight:
		if m.fightModel != nil {
			content = m.fightModel.View()
		}
	case ViewChat:
		if m.chatModel != nil {
			content = m.chatModel.View()
		} else {
			content = "Подключение к чату..."
		}
	case ViewEULA:
		if m.eulaModel != nil {
			content = m.eulaModel.View()
		}
	case ViewFullEULA:
		if m.fullEulaModel != nil {
			content = m.fullEulaModel.View()
		} else {
			content = "Загрузка EULA..."
		}
	case ViewExitConfirm:
		content = m.renderExitConfirm()
	case ViewPvPConnect:
		if m.pvpConnectModel != nil {
			content = m.pvpConnectModel.View()
		} else {
			content = "Подключение..."
		}
	case ViewPvPFight:
		if m.pvpFightModel != nil {
			content = m.pvpFightModel.View()
		} else {
			content = "Бой..."
		}
	default:
		content = "Загрузка..."
	}
	return "\033[?25l\033[?1c" + content
}

func (m *AppModel) applySizeToView(width, height int) {
	if m.mainMenu != nil {
		m.mainMenu.Width, m.mainMenu.Height = width, height
	}
	if m.eulaModel != nil {
		m.eulaModel.Width, m.eulaModel.Height = width, height
	}
	if m.fullEulaModel != nil {
		m.fullEulaModel.Width, m.fullEulaModel.Height = width, height
	}
	if m.fightModel != nil {
		m.fightModel.Width, m.fightModel.Height = width, height
	}
	if m.chatModel != nil {
		m.chatModel.Width, m.chatModel.Height = width, height
	}
	if m.pvpConnectModel != nil {
		m.pvpConnectModel.Width, m.pvpConnectModel.Height = width, height
	}
	if m.pvpFightModel != nil {
		m.pvpFightModel.Width, m.pvpFightModel.Height = width, height
	}
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (AppModel, tea.Cmd) {
	m.width, m.height = msg.Width, msg.Height
	m.applySizeToView(msg.Width, msg.Height)

	utils.HideCursors()
	utils.HideScrollBars()

	return *m, tea.Sequence(
		tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
			utils.HideScrollBars()
			utils.HideCursors()
			return nil
		}),
		tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			utils.HideCursors()
			utils.HideScrollBars()
			return nil
		}),
		tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
			utils.HideCursors()
			return nil
		}),
	)
}

func (m *AppModel) handleKeyInput(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	if msg.Alt && (msg.String() == "f4" || msg.String() == "F4") {
		m.quitting = true
		if m.gameCore != nil {
			m.gameCore.Shutdown()
		}
		return *m, tea.Quit
	}
	if msg.Alt && msg.Type == tea.KeyEnter {
		_ = utils.ToggleFullscreen()
		return *m, createFullscreenToggleCommands()
	}

	isEscOrQuit := msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC || msg.String() == "esc" || msg.String() == "ctrl+c"
	if isEscOrQuit {
		if m.currentView == ViewFullEULA {
			return *m, nil
		}
		if m.currentView == ViewChat {
			if m.chatModel != nil {
				m.chatModel.Disconnect()
			}
			m.currentView = ViewMainMenu
			return *m, nil
		}
		if m.currentView == ViewPvPConnect {
			m.currentView = ViewMainMenu
			return *m, nil
		}
		if m.currentView == ViewPvPFight {
			if m.pvpFightModel != nil {
				m.pvpFightModel.Disconnect()
			}
			m.currentView = ViewMainMenu
			return *m, nil
		}
		if m.currentView == ViewFight {
			return m.delegateToCurrentView(msg)
		}
		if m.currentView == ViewMainMenu && (msg.Type == tea.KeyEsc || msg.String() == "esc") {
			return *m, nil
		}
		if m.currentView == ViewMainMenu {
			m.currentView = ViewExitConfirm
		} else {
			m.currentView = ViewMainMenu
		}
		return *m, nil
	}

	if m.currentView == ViewExitConfirm {
		switch msg.String() {
		case "y", "Y", "д", "Д":
			m.quitting = true
			if m.gameCore != nil {
				m.gameCore.Shutdown()
			}
			return *m, tea.Quit
		case "n", "N", "н", "Н", "esc":
			m.currentView = ViewMainMenu
			return *m, nil
		}
	}
	return m.delegateToCurrentView(msg)
}

func (m *AppModel) handleViewChange(msg ViewChangeMsg) (AppModel, tea.Cmd) {
	m.currentView = msg.View
	var cmd tea.Cmd

	if msg.View == ViewMainMenu {
		sound.StopMusic()
	}

	switch msg.View {
	case ViewFight:

		if m.fightModel == nil || m.fightModel.gameOver || m.fightModel.state == FightViewEnd {
			m.fightModel = NewFightModel(m.gameCore.ExtendedGameManager)
			if m.fightModel != nil {
				m.fightModel.Width, m.fightModel.Height = m.width, m.height
				cmd = m.fightModel.Init()
			}
		} else {
			if m.fightModel.Width != m.width || m.fightModel.Height != m.height {
				m.fightModel.Width, m.fightModel.Height = m.width, m.height

			}
			cmd = nil
		}
	case ViewChat:
		if m.chatModel != nil {
			m.chatModel.Disconnect()
		}
		m.chatModel = NewChatModel()
		if m.chatModel != nil {
			m.chatModel.Width, m.chatModel.Height = m.width, m.height
			cmd = m.chatModel.Init()
		}
	case ViewPvPConnect:
		m.pvpConnectModel = NewPvPConnectModel()
		if m.pvpConnectModel != nil {
			m.pvpConnectModel.Width, m.pvpConnectModel.Height = m.width, m.height
			cmd = ConnectPvPWithFallbackCmd()
		}
	case ViewEULA:
		if m.eulaModel == nil {
			m.eulaModel = NewEULAModel(m.gameCore.ExtendedGameManager)
			if m.eulaModel != nil {
				m.eulaModel.Width, m.eulaModel.Height = m.width, m.height
				m.eulaModel.SetEULAAccepted(m.eulaAccepted)
			}
		} else {
			m.eulaModel.SetEULAAccepted(m.eulaAccepted)
		}
	case ViewFullEULA:
		if m.fullEulaModel == nil {
			m.fullEulaModel = NewFullEULAModel(m.gameCore.ExtendedGameManager)
			if m.fullEulaModel != nil {
				m.fullEulaModel.Width, m.fullEulaModel.Height = m.width, m.height
			}
		}
	case ViewMainMenu:
		sound.StopMusic()
		if m.fullEulaModel != nil {
			m.eulaAccepted = m.fullEulaModel.EulaAccepted
		}

	}

	return *m, cmd
}

func (m *AppModel) handlePvPConnected(msg PvPConnectedMsg) (AppModel, tea.Cmd) {
	if msg.Err != nil {
		if m.pvpConnectModel != nil {
			m.pvpConnectModel.ConnectErr = msg.Err.Error()
		}
		return *m, nil
	}
	if msg.Session == nil {
		m.currentView = ViewMainMenu
		return *m, nil
	}
	m.pvpFightModel = NewPvPFightModel(msg.Session)
	if m.pvpFightModel != nil {
		m.pvpFightModel.Width, m.pvpFightModel.Height = m.width, m.height
		m.currentView = ViewPvPFight
		return *m, m.pvpFightModel.Init()
	}
	m.currentView = ViewMainMenu
	return *m, nil
}

func (m AppModel) delegateToCurrentView(msg tea.Msg) (AppModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if (keyMsg.Type == tea.KeyEsc || keyMsg.Type == tea.KeyCtrlC) && m.currentView == ViewFullEULA {
			return m, nil
		}
	}

	switch m.currentView {
	case ViewMainMenu, ViewExitConfirm:
		if m.mainMenu != nil {
			var cmd tea.Cmd
			m.mainMenu, cmd = m.mainMenu.Update(msg)
			return m, cmd
		}
	case ViewFight:
		if m.fightModel != nil {
			var cmd tea.Cmd
			m.fightModel, cmd = m.fightModel.Update(msg)
			return m, cmd
		}
	case ViewChat:
		if m.chatModel != nil {
			updated, cmd := m.chatModel.Update(msg)
			if newChatModel, ok := updated.(*ChatModel); ok {
				m.chatModel = newChatModel
			}
			return m, cmd
		}
	case ViewPvPConnect:
		if m.pvpConnectModel != nil {
			var cmd tea.Cmd
			m.pvpConnectModel, cmd = m.pvpConnectModel.Update(msg)
			return m, cmd
		}
	case ViewPvPFight:
		if m.pvpFightModel != nil {
			var cmd tea.Cmd
			m.pvpFightModel, cmd = m.pvpFightModel.Update(msg)
			return m, cmd
		}
	case ViewEULA:
		if m.eulaModel != nil {
			var cmd tea.Cmd
			m.eulaModel, cmd = m.eulaModel.Update(msg)
			return m, cmd
		}
	case ViewFullEULA:
		if m.fullEulaModel != nil {
			var cmd tea.Cmd
			m.fullEulaModel, cmd = m.fullEulaModel.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m AppModel) renderExitConfirm() string {
	var b strings.Builder

	width := m.width
	if width < ui.MinWidth {
		width = ui.MinWidth
	}

	verticalPadding := (m.height - 15) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}
	for i := 0; i < verticalPadding; i++ {
		b.WriteString("\n")
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorDanger)).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorDanger)).
		Bold(true)

	questionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorWarning)).
		Italic(true)

	boxContent := fmt.Sprintf("%s\n\n%s\n\n%s\n%s",
		titleStyle.Render("🚪 ПОДТВЕРЖДЕНИЕ ВЫХОДА"),
		questionStyle.Render("Вы уверены, что хотите выйти из игры?"),
		"[Y] Да, выйти",
		"[N] Нет, остаться")

	box := boxStyle.Render(boxContent)
	boxLines := strings.Split(box, "\n")
	maxLineWidth := 0
	for _, line := range boxLines {
		if len([]rune(line)) > maxLineWidth {
			maxLineWidth = len([]rune(line))
		}
	}
	boxPadding := (width - maxLineWidth) / 2
	if boxPadding < 0 {
		boxPadding = 0
	}
	for _, line := range boxLines {
		b.WriteString(strings.Repeat(" ", boxPadding))
		b.WriteString(line)
		b.WriteString("\n")
	}

	helpText := ui.HelpStyle.Render("Y - Выход  │  N - Отмена")
	helpPadding := (width - len([]rune(helpText))) / 2
	if helpPadding < 0 {
		helpPadding = 0
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", helpPadding))
	b.WriteString(helpText)

	return b.String()
}

type QuitMsg struct{}

func createFullscreenToggleCommands() tea.Cmd {
	utils.HideCursors()
	utils.HideScrollBars()
	requestSizeAfter := func(delay time.Duration) tea.Cmd {
		return tea.Tick(delay, func(time.Time) tea.Msg {
			w, h := utils.GetTerminalSize()
			if w <= 0 {
				w = ui.MinWidth
			}
			if h <= 0 {
				h = ui.MinHeight
			}
			return tea.WindowSizeMsg{Width: w, Height: h}
		})
	}

	if utils.IsFullscreen() {
		return tea.Sequence(
			requestSizeAfter(25*time.Millisecond),
			tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
				utils.HideScrollBars()
				utils.HideCursors()
				return nil
			}),
			tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
				utils.SetWindowFocus(utils.GetConsoleWindow())
				utils.HideCursors()
				utils.HideScrollBars()
				return nil
			}),
			requestSizeAfter(120*time.Millisecond),
			tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
				utils.HideCursors()
				return nil
			}),
			requestSizeAfter(250*time.Millisecond),
		)
	}
	return tea.Sequence(
		requestSizeAfter(25*time.Millisecond),
		tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			utils.HideScrollBars()
			utils.HideCursors()
			return nil
		}),
		requestSizeAfter(120*time.Millisecond),
		tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
			utils.HideCursors()
			return nil
		}),
		requestSizeAfter(250*time.Millisecond),
	)
}
