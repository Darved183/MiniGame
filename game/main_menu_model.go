package game

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MyGame/EULA"
	"MyGame/core"
	"MyGame/game/ui"
)

var menuTitleLines = []string{
	"╔════════════════════════════════════════════════════════════════╗",
	"║                                                                ║",
	"║                    ██╗████████╗ █ '███████╗                    ║",
	"║                    ██║╚══██╔══╝ ═╝ ██╔════╝                    ║",
	"║                    ██║   ██║      ███████╗                     ║",
	"║                    ██║   ██║      ╚════██║                     ║",
	"║                    ██║   ██║      ███████║                     ║",
	"║                    ╚═╝   ╚═╝      ╚══════╝                     ║",
	"║                                                                ║",
	"║              ██╗ ██╗ █████╗ ██████╗ ██████╗                    ║",
	"║              ██║ ██║██╔══██╗██╔══██╗██╔══██╗                   ║",
	"║              ███████║███████║██████╔╝██║  ██║                  ║",
	"║              ██╔══██║██╔══██║██╔══██╗██║  ██║                  ║",
	"║              ██║  ██║██║  ██║██║  ██║██████╔╝                  ║",
	"║              ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝                   ║",
	"║                                                                ║",
	"╚════════════════════════════════════════════════════════════════╝",
}

type MainMenuModel struct {
	gameManager   *core.ExtendedGameManager
	selected      int
	Width         int
	Height        int
	displayedText string
	typingIndex   int
	isTyping      bool
	typingSpeed   time.Duration
}

func NewMainMenuModel(gameCore *core.Core) *MainMenuModel {
	return &MainMenuModel{
		gameManager:   gameCore.ExtendedGameManager,
		selected:      0,
		Width:         80,
		Height:        24,
		displayedText: "",
		typingIndex:   0,
		isTyping:      true,
		typingSpeed:   time.Millisecond,
	}
}

func (m *MainMenuModel) Init() tea.Cmd {
	return m.startTyping()
}

func (m *MainMenuModel) startTyping() tea.Cmd {
	return tea.Tick(m.typingSpeed, func(time.Time) tea.Msg { return TypingTickMsg{} })
}

func (m *MainMenuModel) Update(msg tea.Msg) (*MainMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TypingTickMsg:
		if m.isTyping {
			fullText := m.buildFullMenuText()
			runes := []rune(fullText)
			if m.typingIndex < len(runes) {
				m.displayedText = string(runes[:m.typingIndex+1])
				m.typingIndex++
				return m, m.startTyping()
			}
			m.isTyping = false
			m.displayedText = fullText
		}
	case tea.KeyMsg:
		if m.isTyping {
			fullText := m.buildFullMenuText()
			m.displayedText = fullText
			m.typingIndex = len([]rune(fullText))
			m.isTyping = false
		}
		if !m.isTyping {
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < 4 {
					m.selected++
				}
			case "enter", " ":
				return m, m.handleSelection()
			case "q":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *MainMenuModel) buildFullMenuText() string {
	var b strings.Builder
	for _, line := range menuTitleLines {
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m *MainMenuModel) handleSelection() tea.Cmd {
	switch m.selected {
	case 0:
		return func() tea.Msg { return ViewChangeMsg{ViewFight} }
	case 1:
		return func() tea.Msg { return ViewChangeMsg{ViewPvPConnect} }
	case 2:
		return func() tea.Msg { return ViewChangeMsg{ViewChat} }
	case 3:
		return func() tea.Msg { return ViewChangeMsg{ViewEULA} }
	case 4:
		return func() tea.Msg { return ViewChangeMsg{ViewExitConfirm} }
	}
	return nil
}

func (m *MainMenuModel) View() string {
	var b strings.Builder
	width := max(m.Width, ui.MinWidth)

	titleTop := (m.Height * 12) / 100
	for i := 0; i < titleTop; i++ {
		b.WriteString("\n")
	}

	if m.isTyping && m.displayedText != "" {
		lines := strings.Split(m.displayedText, "\n")
		for i, line := range lines {
			if line != "" {
				rendered := line
				if i == len(lines)-1 {
					rendered += "█"
				}
				ui.CenteredLineByVisibleWidth(&b, ui.TitleStyle.Render(rendered), len([]rune(rendered)), width)
			}
		}
	} else {
		for _, line := range menuTitleLines {
			if line != "" {
				ui.CenteredLineByVisibleWidth(&b, ui.TitleStyle.Render(line), len([]rune(line)), width)
			} else {
				b.WriteString("\n")
			}
		}

		menuTop := (m.Height * 58) / 100
		currentLines := strings.Count(b.String(), "\n")
		for i := 0; i < max(0, menuTop-currentLines); i++ {
			b.WriteString("\n")
		}

		for i, item := range []string{"1. Быстрый бой", "2. Сетевой бой (PvP)", "3. Чат", "4. Лицензия", "5. Выход"} {
			ui.CenteredLineBuilder(&b, ui.RenderMenuItem(i == m.selected, item), width)
		}

		helpTop := (m.Height * 82) / 100
		currentLines = strings.Count(b.String(), "\n")
		for i := 0; i < max(0, helpTop-currentLines); i++ {
			b.WriteString("\n")
		}
		ui.CenteredLineBuilder(&b, ui.HelpStyle.Render("↑↓ Навигация  │  Enter Выбор  │  ESC Выход"), width)
	}
	return b.String()
}

type ViewChangeMsg struct{ View ViewType }
type ViewType int

const (
	ViewMainMenu ViewType = iota
	ViewFight
	ViewInventory
	ViewEULA
	ViewExitConfirm
	ViewChat
	ViewFullEULA
	ViewPvPConnect
	ViewPvPFight
)
const SkipEULA = false

type EULAModel struct {
	gameManager  *core.ExtendedGameManager
	selected     int
	Width        int
	Height       int
	eulaAccepted bool
}

func NewEULAModel(gameManager *core.ExtendedGameManager) *EULAModel {
	return &EULAModel{gameManager: gameManager, selected: 0, Width: ui.MinWidth, Height: ui.MinHeight}
}

func (m *EULAModel) SetEULAAccepted(accepted bool) { m.eulaAccepted = accepted }

func (m *EULAModel) Init() tea.Cmd { return nil }

func (m *EULAModel) Update(msg tea.Msg) (*EULAModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < 2 {
				m.selected++
			}
		case "enter", " ":
			return m, m.handleSelection()
		case "esc":
			return m, func() tea.Msg { return ViewChangeMsg{ViewMainMenu} }
		}
	}
	return m, nil
}

func (m *EULAModel) handleSelection() tea.Cmd {
	switch m.selected {
	case 0:
		if m.gameManager != nil && m.gameManager.Deps != nil && m.gameManager.Deps.GetLogger() != nil {
			m.gameManager.Deps.GetLogger().GameEvent("eula", "лицензия принята")
		}
		fallthrough
	case 2:
		return func() tea.Msg { return ViewChangeMsg{ViewMainMenu} }
	case 1:
		return tea.Quit
	}
	return nil
}

func (m *EULAModel) View() string {
	var b strings.Builder
	width := max(m.Width, ui.MinWidth)

	verticalPadding := min(3, max(0, (m.Height-25)/2))
	for i := 0; i < verticalPadding; i++ {
		b.WriteString("\n")
	}

	titleStyle := ui.TitleStyle.Copy().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ui.ColorBorder)).Padding(0, 1)
	title := titleStyle.Render("📄 ЛИЦЕНЗИОННОЕ СОГЛАШЕНИЕ")
	ui.CenteredLineBuilder(&b, title, width)
	b.WriteString("\n")

	eulaContent := EULA.ShowDevelopersText()
	if m.eulaAccepted {
		eulaContent = "✅ ЛИЦЕНЗИОННОЕ СОГЛАШЕНИЕ ПРИНЯТО\n\n" + eulaContent
	}

	for _, line := range strings.Split(eulaContent, "\n") {
		centeredWrappedLine(&b, line, width-4)
	}
	b.WriteString("\n")

	for i, item := range []string{"1. Принять", "2. Отклонить", "3. Назад"} {
		ui.CenteredLineBuilder(&b, ui.RenderMenuItem(i == m.selected, item), width)
	}
	return b.String()
}

type FullEULAModel struct {
	gameManager   *core.ExtendedGameManager
	selected      int
	Width         int
	Height        int
	currentText   string
	displayedText string
	typingIndex   int
	isTyping      bool
	typingSpeed   time.Duration
	EulaAccepted  bool
}

func NewFullEULAModel(gameManager *core.ExtendedGameManager) *FullEULAModel {
	return &FullEULAModel{
		gameManager: gameManager,
		selected:    0,
		Width:       ui.MinWidth,
		Height:      ui.MinHeight,
		currentText: EULA.GetFullEULAText(),
		typingSpeed: 20 * time.Millisecond,
		isTyping:    true,
	}
}

func (m *FullEULAModel) Init() tea.Cmd { return m.startTyping() }

func (m *FullEULAModel) startTyping() tea.Cmd {
	return tea.Tick(m.typingSpeed, func(time.Time) tea.Msg { return TypingTickMsg{} })
}

type TypingTickMsg struct{}

func (m *FullEULAModel) Update(msg tea.Msg) (*FullEULAModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TypingTickMsg:
		if m.isTyping && m.typingIndex < len([]rune(m.currentText)) {
			runes := []rune(m.currentText)
			m.displayedText = string(runes[:m.typingIndex+1])
			m.typingIndex++
			return m, m.startTyping()
		}
		m.isTyping = false
	case tea.KeyMsg:
		if !m.isTyping {
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < 1 {
					m.selected++
				}
			case "enter", " ":
				return m, m.handleSelection()
			}
		}
	}
	return m, nil
}

func (m *FullEULAModel) handleSelection() tea.Cmd {
	switch m.selected {
	case 0:
		m.EulaAccepted = true
		if m.gameManager != nil && m.gameManager.Deps != nil && m.gameManager.Deps.GetLogger() != nil {
			m.gameManager.Deps.GetLogger().GameEvent("eula", "лицензия принята при запуске")
		}
		return func() tea.Msg { return ViewChangeMsg{ViewMainMenu} }
	case 1:
		return tea.Quit
	}
	return nil
}

func (m *FullEULAModel) View() string {
	var b strings.Builder
	width := max(m.Width, ui.MinWidth)

	titleStyle := ui.TitleStyle.Copy().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ui.ColorBorder)).Padding(0, 1)
	titleLines := strings.Split(titleStyle.Render("📄 ЛИЦЕНЗИОННОЕ СОГЛАШЕНИЕ"), "\n")
	titleMaxWidth := lipglossWidth(titleLines)
	for _, line := range titleLines {
		ui.CenteredLineByVisibleWidth(&b, line, titleMaxWidth, width)
	}
	b.WriteString("\n")

	if m.displayedText != "" {
		lines := strings.Split(m.displayedText, "\n")
		for i, line := range lines {
			centeredWrappedLineWithCursor(&b, line, width-4, m.isTyping && i == len(lines)-1)
		}
	}

	if !m.isTyping {
		b.WriteString("\n\n")
		for i, item := range []string{"1. Принять условия соглашения", "2. Отклонить (игра будет закрыта)"} {
			ui.CenteredLineBuilder(&b, ui.RenderMenuItem(i == m.selected, item), width)
		}
	}
	return b.String()
}

func lipglossWidth(lines []string) int {
	maxWidth := 0
	for _, line := range lines {
		if line != "" {
			visible := 0
			inAnsi := false
			for _, r := range line {
				if r == '\033' {
					inAnsi = true
				} else if inAnsi && r == 'm' {
					inAnsi = false
				} else if !inAnsi {
					visible++
				}
			}
			if visible > maxWidth {
				maxWidth = visible
			}
		}
	}
	return maxWidth
}

func centeredWrappedLine(b *strings.Builder, line string, maxWidth int) {
	words := strings.Fields(line)
	current := ""
	for _, word := range words {
		if len([]rune(current+" "+word)) > maxWidth {
			if current != "" {
				ui.CenteredLineBuilder(b, current, maxWidth+4)
			}
			current = word
		} else {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		}
	}
	if current != "" {
		ui.CenteredLineBuilder(b, current, maxWidth+4)
	}
}

func centeredWrappedLineWithCursor(b *strings.Builder, line string, maxWidth int, showCursor bool) {
	words := strings.Fields(line)
	current := ""
	for _, word := range words {
		if len([]rune(current+" "+word)) > maxWidth {
			if current != "" {
				ui.CenteredLineBuilder(b, current, maxWidth+4)
			}
			current = word
		} else {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		}
	}
	if current != "" {
		if showCursor {
			current += "█"
		}
		ui.CenteredLineBuilder(b, current, maxWidth+4)
	}
}
