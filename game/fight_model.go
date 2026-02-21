package game

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MyGame/Struct/Character"
	"MyGame/core"
	"MyGame/game/ui"
	"MyGame/sound"
)

type FightModel struct {
	gameManager     *core.ExtendedGameManager
	turnHandler     *TurnHandler
	itemManager     *ItemEffectManager
	player          *Character.Character
	enemy           *Character.Character
	selected        int
	state           FightViewState
	Width           int
	Height          int
	message         string
	showMessage     bool
	round           int
	waitingForEnemy bool
	itemSelected    int
	gameOver        bool
}

type FightViewState int

const (
	FightViewActionMenu FightViewState = iota
	FightViewItemMenu
	FightViewSurrenderConfirm
	FightViewExitConfirm
	FightViewEnd
)

func NewFightModel(gameManager *core.ExtendedGameManager) *FightModel {
	if gameManager == nil {
		return nil
	}

	player := gameManager.GetPlayer()
	if player == nil {
		fallback, err := Character.New("Герой", 100, 1, 1, 1)
		if err != nil {
			return nil
		}
		gameManager.UpdatePlayer(fallback)
		player = gameManager.GetPlayer()
		if player == nil {
			return nil
		}
	}

	playerCopy, err := Character.New(
		player.GetName(),
		player.GetBaseHP(),
		player.GetBaseStrength(),
		player.GetBaseAgility(),
		player.GetBaseIntelligence(),
	)
	if err != nil {
		return nil
	}

	playerCopy.AddStarterItems()
	playerCopy.CalculateStats()

	enemy, err := Character.New("Дракон", 120, 15, 1, 1)
	if err != nil {
		return nil
	}
	enemy.AddStarterItems()
	enemy.CalculateStats()

	return &FightModel{
		gameManager:     gameManager,
		turnHandler:     NewTurnHandler(),
		itemManager:     NewItemEffectManager(),
		player:          playerCopy,
		enemy:           enemy,
		selected:        0,
		state:           FightViewActionMenu,
		Width:           ui.MinWidth,
		Height:          ui.MinHeight,
		message:         "",
		showMessage:     false,
		round:           1,
		waitingForEnemy: false,
		itemSelected:    0,
		gameOver:        false,
	}
}

func (m *FightModel) Init() tea.Cmd {
	if m.player != nil && m.enemy != nil {
		m.player.CalculateStats()
		m.enemy.CalculateStats()
		m.round = 1
		sound.PlayMusic()
	}
	return nil
}

func (m *FightModel) stopMusic() {
	sound.StopMusic()
}

func (m *FightModel) Update(msg tea.Msg) (*FightModel, tea.Cmd) {
	if m.checkBattleEnd() && !m.gameOver {
		m.gameOver = true
		m.state = FightViewEnd
		if m.player.GetHP() <= 0 {
			m.message = fmt.Sprintf("💀 Вы проиграли! %s победил!", m.enemy.GetName())
		} else if m.enemy.GetHP() <= 0 {
			m.message = fmt.Sprintf("🎉 Победа! Вы победили %s!", m.enemy.GetName())
		}
		m.showMessage = true
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.gameOver && m.state == FightViewEnd {
			return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
		}

		switch m.state {
		case FightViewActionMenu:
			return m.updateActionMenu(msg)
		case FightViewItemMenu:
			return m.updateItemMenu(msg)
		case FightViewSurrenderConfirm:
			return m.updateSurrenderConfirm(msg)
		case FightViewExitConfirm:
			return m.updateExitConfirm(msg)
		}
	}
	return m, nil
}

func (m *FightModel) updateActionMenu(msg tea.KeyMsg) (*FightModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < 3 {
			m.selected++
		}
	case "enter", " ":
		switch m.selected {
		case 0:
			m.doPlayerAttackAndEnemyTurn()
			return m, nil
		case 1:
			m.state = FightViewItemMenu
			m.itemSelected = 0
		case 2:
			m.showMessage = true
			m.message = m.getBattleStats()
			return m, nil
		case 3:
			m.state = FightViewSurrenderConfirm
			m.selected = 0
		}
	case "esc":
		m.state = FightViewExitConfirm
		m.selected = 0
	}
	return m, nil
}

func (m *FightModel) updateExitConfirm(msg tea.KeyMsg) (*FightModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "д", "Д":
		return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
	case "n", "N", "н", "Н", "esc":
		m.state = FightViewActionMenu
		m.selected = 0
	}
	return m, nil
}

func (m *FightModel) doPlayerAttackAndEnemyTurn() {
	if m.turnHandler == nil {
		m.message = "Ошибка боевой системы"
		m.showMessage = true
		return
	}
	var lines []string
	damage1, blocked1 := m.turnHandler.SimpleStrike(m.player, m.enemy)
	if blocked1 {
		lines = append(lines, fmt.Sprintf("🛡️ %s заблокировал удар!", m.enemy.GetName()))
	} else {
		lines = append(lines, fmt.Sprintf("💥 Вы нанесли %d урона! %s: %d/%d HP", damage1, m.enemy.GetName(), m.enemy.GetHP(), m.enemy.GetMaxHP()))
	}
	if m.checkBattleEnd() {
		m.message = lines[0]
		m.showMessage = true
		return
	}
	damage2, blocked2 := m.turnHandler.SimpleStrike(m.enemy, m.player)
	m.round++
	if blocked2 {
		lines = append(lines, fmt.Sprintf("🛡️ Вы заблокировали атаку %s!", m.enemy.GetName()))
	} else {
		lines = append(lines, fmt.Sprintf("⚔️ %s наносит %d урона! Ваш HP: %d/%d", m.enemy.GetName(), damage2, m.player.GetHP(), m.player.GetMaxHP()))
	}
	m.message = strings.Join(lines, "  │  ")
	m.showMessage = true
}

func (m *FightModel) updateItemMenu(msg tea.KeyMsg) (*FightModel, tea.Cmd) {
	if m.itemManager == nil {
		m.state = FightViewActionMenu
		m.message = "Ошибка: менеджер предметов не инициализирован"
		m.showMessage = true
		return m, nil
	}

	items := m.player.GetInventory().GetItems()
	usableItems := m.itemManager.GetUsableItems(items)

	if len(usableItems) == 0 {
		m.state = FightViewActionMenu
		m.message = "❌ Нет предметов для использования!"
		m.showMessage = true
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.itemSelected > 0 {
			m.itemSelected--
		}
	case "down", "j":
		if m.itemSelected < len(usableItems)-1 {
			m.itemSelected++
		}
	case "enter", " ":

		selectedItem := usableItems[m.itemSelected]
		success := m.itemManager.UseItem(m.player, m.enemy, selectedItem)

		if success {
			if _, err := m.player.GetInventory().RemoveItem(selectedItem.Template.ID); err != nil {
				m.message = fmt.Sprintf("✅ %s использован (предмет не удалён из инвентаря: %v)", selectedItem.Template.Name, err)
			} else {
				m.message = fmt.Sprintf("✅ %s использован!", selectedItem.Template.Name)
			}
			m.showMessage = true
			m.state = FightViewActionMenu
			m.itemSelected = 0

			if m.checkBattleEnd() {
				m.gameOver = true
				m.state = FightViewEnd
				if m.player.GetHP() <= 0 {
					m.message += fmt.Sprintf("  │  💀 Вы проиграли! %s победил!", m.enemy.GetName())
				} else if m.enemy.GetHP() <= 0 {
					m.message += fmt.Sprintf("  │  🎉 Победа! Вы победили %s!", m.enemy.GetName())
				}
				return m, nil
			}

			damage, blocked := m.turnHandler.SimpleStrike(m.enemy, m.player)
			m.round++
			if blocked {
				m.message += fmt.Sprintf("  │  🛡️ Вы заблокировали атаку %s!", m.enemy.GetName())
			} else {
				m.message += fmt.Sprintf("  │  ⚔️ %s наносит %d урона! Ваш HP: %d/%d", m.enemy.GetName(), damage, m.player.GetHP(), m.player.GetMaxHP())
			}
			return m, nil
		}
		m.message = fmt.Sprintf("❌ Не удалось использовать %s", selectedItem.Template.Name)
		m.showMessage = true
		return m, nil
	case "esc":
		m.state = FightViewActionMenu
		m.itemSelected = 0
	}
	return m, nil
}

func (m *FightModel) updateSurrenderConfirm(msg tea.KeyMsg) (*FightModel, tea.Cmd) {
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
		if m.selected == 0 {
			m.gameOver = true
			m.state = FightViewEnd
			m.message = "Вы сдались!"
			m.showMessage = true
		} else {
			m.state = FightViewActionMenu
			m.selected = 0
		}
	case "esc":
		m.state = FightViewActionMenu
		m.selected = 0
	}
	return m, nil
}

func (m *FightModel) checkBattleEnd() bool {
	if m.player == nil || m.enemy == nil {
		return false
	}
	return m.player.GetHP() <= 0 || m.enemy.GetHP() <= 0
}

func (m *FightModel) getBattleStats() string {
	return fmt.Sprintf(`📊 СТАТИСТИКА БОЯ

%s: HP=%d/%d, Атака=%.1f, Защита=%.1f
%s: HP=%d/%d, Атака=%.1f, Защита=%.1f`,
		m.player.GetName(), m.player.GetHP(), m.player.GetMaxHP(),
		m.player.GetAttack(), m.player.GetDefense(),
		m.enemy.GetName(), m.enemy.GetHP(), m.enemy.GetMaxHP(),
		m.enemy.GetAttack(), m.enemy.GetDefense())
}

func (m *FightModel) View() string {
	if m.state == FightViewEnd {
		return m.renderEndScreen()
	}
	return m.renderBattleScreen()
}

func (m *FightModel) renderBattleScreen() string {
	var b strings.Builder

	width := m.Width
	if width < ui.MinWidth {
		width = ui.MinWidth
	}

	roundStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorTitle)).
		Bold(true).
		Padding(0, 1)
	title := roundStyle.Render(fmt.Sprintf("⚔️ РАУНД %d", m.round))
	b.WriteString(ui.CenteredLine(title, width))
	b.WriteString("\n\n")

	enemyName := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorDanger)).Bold(true).Render("◆ " + m.enemy.GetName())
	b.WriteString(ui.RenderBattleHpLine(enemyName, m.enemy.GetHP(), m.enemy.GetMaxHP(), width))
	b.WriteString("\n")

	vsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorBorder)).Bold(true)
	b.WriteString(ui.CenteredLine(vsStyle.Render("────── VS ──────"), width))
	b.WriteString("\n")

	playerName := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Bold(true).Render("◆ " + m.player.GetName())
	b.WriteString(ui.RenderBattleHpLine(playerName, m.player.GetHP(), m.player.GetMaxHP(), width))
	b.WriteString("\n\n")

	if m.showMessage && m.message != "" {
		logLine := strings.ReplaceAll(m.message, "\n", " ")
		logStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ui.ColorStats)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ui.ColorBorder)).
			Padding(0, 2).
			Width(width - 4)
		b.WriteString(ui.CenteredLine(logStyle.Render(logLine), width))
		b.WriteString("\n\n")
	}

	switch m.state {
	case FightViewActionMenu:
		b.WriteString(m.renderActionMenu())
	case FightViewItemMenu:
		b.WriteString(m.renderItemMenu())
	case FightViewSurrenderConfirm:
		b.WriteString(m.renderSurrenderConfirm())
	case FightViewExitConfirm:
		b.WriteString(m.renderExitConfirm())
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp))
	helpPlain := "↑↓ Выбор  │  Enter Подтвердить  │  ESC Меню"
	if m.state == FightViewExitConfirm {
		helpPlain = "Y — выйти  │  N / ESC — остаться"
	}
	b.WriteString("\n" + ui.CenteredLine(helpStyle.Render(helpPlain), width))

	return b.String()
}

func (m *FightModel) renderActionMenu() string {
	items := []string{
		"⚔ Атака",
		"🧪 Предмет",
		"📊 Статистика",
		"🚪 Сдаться",
	}
	var b strings.Builder
	for i, item := range items {
		b.WriteString(ui.RenderMenuItem(i == m.selected, item) + "\n")
	}
	return b.String()
}

func (m *FightModel) renderExitConfirm() string {
	return ui.WarningStyle.Render("🚪 Выйти из боя? Y — да  N/ESC — нет") + "\n"
}

func (m *FightModel) renderItemMenu() string {
	if m.itemManager == nil {
		return "Ошибка: менеджер предметов не инициализирован\n"
	}

	items := m.player.GetInventory().GetItems()
	usableItems := m.itemManager.GetUsableItems(items)

	if len(usableItems) == 0 {
		return ui.DangerStyle.Render("❌ НЕТ ПРЕДМЕТОВ") + "\n"
	}

	var b strings.Builder
	b.WriteString(ui.TitleStyle.Render("📦 ВЫБЕРИТЕ ПРЕДМЕТ") + "\n\n")

	for i, item := range usableItems {
		effectDesc := m.itemManager.GetItemEffectDescription(item)
		itemText := fmt.Sprintf("%d. %s — %s", i+1, item.Template.Name, effectDesc)
		b.WriteString(ui.RenderMenuItem(i == m.itemSelected, itemText) + "\n")
	}
	b.WriteString("\n" + ui.HelpStyle.Render("ESC — Назад"))
	return b.String()
}

func (m *FightModel) renderSurrenderConfirm() string {
	var b strings.Builder
	b.WriteString(ui.DangerStyle.Render("🚩 СДАЧА") + "\n\n")
	b.WriteString(ui.WarningStyle.Render("Вы уверены, что хотите сдаться?") + "\n\n")
	items := []string{"Да, сдаться", "Нет, продолжить бой"}
	for i, item := range items {
		b.WriteString(ui.RenderMenuItem(i == m.selected, item) + "\n")
	}
	return b.String()
}

func (m *FightModel) renderEndScreen() string {
	var b strings.Builder
	width := m.Width
	if width < ui.MinWidth {
		width = ui.MinWidth
	}

	verticalPadding := (m.Height - 10) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}
	for i := 0; i < verticalPadding; i++ {
		b.WriteString("\n")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorSuccess)).
		Bold(true).
		Padding(0, 1)
	title := titleStyle.Render("🎉 БОЙ ЗАВЕРШЕН 🎉")
	b.WriteString(ui.CenteredLine(title, width) + "\n\n")

	if m.message != "" {
		b.WriteString(ui.CenteredLine(m.message, width) + "\n\n")
	}

	help := ui.HelpStyle.Render("▶ Любая клавиша — в главное меню")
	b.WriteString(ui.CenteredLine(help, width))

	return b.String()
}
