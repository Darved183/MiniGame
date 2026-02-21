package game

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"MyGame/Struct/Character"
	"MyGame/sound"

	"MyGame/Struct/Item"
	"MyGame/game/ui"
	"MyGame/utils"
)

func serverHosts() []string {

	if host := os.Getenv("SERVER_HOST"); host != "" {
		return []string{host}
	}

	return []string{"localhost"}
}

func serverPortFromEnv() string {
	if port := os.Getenv("PVP_PORT"); port != "" {
		return port
	}
	return "7000"
}

type PvPConnectedMsg struct {
	Session *Session
	Err     error
}

type PvPConnectModel struct {
	Width      int
	Height     int
	ConnectErr string
}

func NewPvPConnectModel() *PvPConnectModel {
	return &PvPConnectModel{
		Width:  ui.MinWidth,
		Height: ui.MinHeight,
	}
}

func (m *PvPConnectModel) Init() tea.Cmd {

	return ConnectPvPWithFallbackCmd()
}

func ConnectPvPWithFallbackCmd() tea.Cmd {
	return func() tea.Msg {
		hosts := serverHosts()
		port := serverPortFromEnv()
		var lastErr error
		for _, host := range hosts {
			addr := strings.TrimSpace(host)
			if addr == "" {
				continue
			}
			if !strings.Contains(addr, ":") {
				addr = addr + ":" + port
			}
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err == nil {
				return PvPConnectedMsg{Session: NewSession(conn)}
			}
			lastErr = err
		}
		return PvPConnectedMsg{Err: lastErr}
	}
}

func (m *PvPConnectModel) Update(msg tea.Msg) (*PvPConnectModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
	}
	return m, nil
}

func (m *PvPConnectModel) View() string {
	var b strings.Builder
	w := m.Width
	if w < ui.MinWidth {
		w = ui.MinWidth
	}
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorTitle)).Bold(true).Render("🌐 Сетевой бой (PvP)")
	padding := (w - len([]rune(title))) / 2
	if padding < 0 {
		padding = 0
	}
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(title)
	b.WriteString("\n\n")
	if m.ConnectErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorDanger)).Render("❌ " + m.ConnectErr + "\n\nESC — назад"))
		return b.String()
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render("Подключение... ESC — отмена"))
	return b.String()
}

type PvPIncomingMsg struct {
	Line string
	Err  error
}

type pvpHideMessageMsg struct{}

var pvpSwordIDs = []int{1, 24, 25, 26}

const pvpHP, pvpStr, pvpAgl, pvpInt = 100, 14, 7, 4

func fixRunesForWindows(runes []rune) []rune {
	if len(runes) == 0 {
		return runes
	}
	for _, r := range runes {
		if r > 0xFF {
			return runes
		}
	}
	bytes := make([]byte, len(runes))
	for i, r := range runes {
		bytes[i] = byte(r)
	}
	if utf8.Valid(bytes) {
		return []rune(string(bytes))
	}
	decoder := charmap.Windows1251.NewDecoder()
	if utf8Bytes, _, err := transform.Bytes(decoder, bytes); err == nil && utf8.Valid(utf8Bytes) {
		return []rune(string(utf8Bytes))
	}
	return runes
}

type PvPFightModel struct {
	session         *Session
	MySide          int
	player          *Character.Character
	enemy           *Character.Character
	p1              *Character.Character
	p2              *Character.Character
	round           int
	turn            int
	state           FightViewState
	selected        int
	message         string
	showMessage     bool
	chatLines       []string
	chatInput       []rune
	chatFocused     bool
	Width           int
	Height          int
	turnHandler     *TurnHandler
	itemManager     *ItemEffectManager
	gameOver        bool
	winnerSide      int
	itemSelected    int
	waitingForMatch bool
	connectionErr   string
	weaponEquipped  bool
	waitingForState bool
}

func NewPvPFightModel(session *Session) *PvPFightModel {
	th := NewTurnHandler()
	iem := NewItemEffectManager()
	p1, _ := Character.New("Игрок1", pvpHP, pvpStr, pvpAgl, pvpInt)
	p2, _ := Character.New("Игрок2", pvpHP, pvpStr, pvpAgl, pvpInt)
	p1.CalculateStats()
	p2.CalculateStats()
	return &PvPFightModel{
		session:         session,
		MySide:          0,
		p1:              p1,
		p2:              p2,
		player:          p1,
		enemy:           p2,
		round:           1,
		turn:            1,
		state:           FightViewActionMenu,
		turnHandler:     th,
		itemManager:     iem,
		chatLines:       []string{},
		chatInput:       make([]rune, 0, 128),
		Width:           ui.MinWidth,
		Height:          ui.MinHeight,
		waitingForMatch: true,
	}
}

func readPvPLineCmd(session *Session) tea.Cmd {
	return func() tea.Msg {
		if session == nil {
			return PvPIncomingMsg{Err: io.ErrClosedPipe}
		}
		line, err := session.ReadLine()
		return PvPIncomingMsg{Line: line, Err: err}
	}
}

func (m *PvPFightModel) Init() tea.Cmd {
	utils.SetUTF8CodePage()
	sound.PlayMusic()
	if m.session == nil {
		return nil
	}
	return readPvPLineCmd(m.session)
}

func (m *PvPFightModel) myTurn() bool {
	return m.MySide != 0 && m.turn == m.MySide
}

func (m *PvPFightModel) equipPvPWeapon() {
	if m.weaponEquipped || m.player == nil {
		return
	}
	for _, id := range pvpSwordIDs {
		if sword := Item.CreatePvPSword(id); sword != nil {
			m.player.Inventory.AddItem(sword)
		}
	}
	if potion := Item.CreateHealthPotion(); potion != nil {
		m.player.Inventory.AddItem(potion)
	}
	_ = m.player.EquipItem(pvpSwordIDs[0])
	m.player.CalculateStats()
	m.weaponEquipped = true
}

func (m *PvPFightModel) applyState(s State) {
	m.round = s.Round
	if m.MySide == 1 {
		m.player.SetHP(s.P1HP)
		m.enemy.SetHP(s.P2HP)
	} else {
		m.player.SetHP(s.P2HP)
		m.enemy.SetHP(s.P1HP)
	}
	if s.Turn == 1 || s.Turn == 2 {
		m.turn = s.Turn
		m.waitingForState = false
		if m.turn == m.MySide {
			m.state = FightViewActionMenu
		}
	}
}

func (m *PvPFightModel) applyAction(a Action) tea.Cmd {
	if a.Kind == "surrender" {
		m.gameOver = true
		m.winnerSide = m.MySide
		m.state = FightViewEnd
		m.message = "Противник сдался! Вы победили!"
		m.showMessage = true
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ViewChangeMsg{View: ViewMainMenu}
		})
	}
	if a.Kind == "attack" && a.BodyPart != "" {
		attacker, defender := m.enemy, m.player
		if a.Damage > 0 {
			defender.TakeDamage(a.Damage)
			m.message = fmt.Sprintf("⚔️ %s наносит %d урона! Ваш HP: %d/%d",
				attacker.GetName(), a.Damage, m.player.GetHP(), m.player.GetMaxHP())
		} else {
			m.message = fmt.Sprintf("🛡️ %s атаковал, но вы заблокировали!", attacker.GetName())
		}
		m.showMessage = true
		return nil
	}
	if a.Kind == "item" {
		m.message = "Противник использовал предмет"
		m.showMessage = true
		return nil
	}
	return nil
}

func (m *PvPFightModel) sendState() {
	if m.session == nil {
		return
	}
	s := State{Round: m.round}
	if m.MySide == 1 {
		s.P1HP = m.player.GetHP()
		s.P2HP = m.enemy.GetHP()
	} else {
		s.P1HP = m.enemy.GetHP()
		s.P2HP = m.player.GetHP()
	}

	if m.player.GetHP() <= 0 {
		m.gameOver = true
		m.winnerSide = 3 - m.MySide
		m.state = FightViewEnd
		_ = m.pvpSend(SerializeEnd(End{Winner: m.winnerSide}))
		time.Sleep(100 * time.Millisecond)
		_ = m.session.Close()
		m.session = nil
		sound.StopMusic()
		return
	}
	if m.enemy.GetHP() <= 0 {
		m.gameOver = true
		m.winnerSide = m.MySide
		m.state = FightViewEnd
		_ = m.pvpSend(SerializeEnd(End{Winner: m.winnerSide}))
		time.Sleep(100 * time.Millisecond)
		_ = m.session.Close()
		m.session = nil
		sound.StopMusic()
		return
	}

	m.turn = 3 - m.MySide
	s.Turn = m.turn
	m.waitingForState = true
	_ = m.pvpSend(SerializeState(s))
}

func pvpScheduleHideMessage() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return pvpHideMessageMsg{} })
}

func (m *PvPFightModel) pvpSend(line string) error {
	if m.session == nil {
		return io.ErrClosedPipe
	}
	return m.session.WriteLine(line)
}

func (m *PvPFightModel) Update(msg tea.Msg) (*PvPFightModel, tea.Cmd) {
	if m.session == nil && !m.gameOver && m.connectionErr == "" && !m.waitingForMatch {
		m.connectionErr = "⚠️ Соединение разорвано. Нажмите Enter для выхода в меню."
	}

	switch msg := msg.(type) {
	case pvpHideMessageMsg:
		m.showMessage = false
		return m, nil

	case PvPIncomingMsg:
		if msg.Err != nil {
			m.connectionErr = "⚠️ Соединение разорвано. Нажмите Enter для выхода в меню."
			if m.session != nil {
				_ = m.session.Close()
				m.session = nil
			}
			return m, nil
		}

		line := strings.TrimSpace(msg.Line)
		if line == "" {
			return m, readPvPLineCmd(m.session)
		}

		if m.waitingForMatch {
			if strings.HasPrefix(line, "YOU_ARE") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if parts[1] == "1" {
						m.MySide = 1
						m.player = m.p1
						m.enemy = m.p2
					} else if parts[1] == "2" {
						m.MySide = 2
						m.player = m.p2
						m.enemy = m.p1
					}
				}
			} else if strings.HasPrefix(line, MsgInit) {
				init, _ := ParseInit(line)
				if m.MySide == 1 {
					m.player.SetHP(init.P1HP)
					m.enemy.SetHP(init.P2HP)
				} else {
					m.player.SetHP(init.P2HP)
					m.enemy.SetHP(init.P1HP)
				}
				m.round = init.Round
				if init.Turn == 1 || init.Turn == 2 {
					m.turn = init.Turn
				} else {
					m.turn = 1
				}
				m.waitingForMatch = false
				m.equipPvPWeapon()
			}

			return m, readPvPLineCmd(m.session)
		}

		switch {
		case strings.HasPrefix(line, MsgAction):

			a, _ := ParseAction(line)
			cmd := m.applyAction(a)
			if cmd != nil {

				return m, tea.Batch(cmd, readPvPLineCmd(m.session))
			}

			return m, tea.Batch(readPvPLineCmd(m.session), pvpScheduleHideMessage())

		case strings.HasPrefix(line, MsgState):
			s, _ := ParseState(line)
			m.applyState(s)

			return m, readPvPLineCmd(m.session)

		case strings.HasPrefix(line, MsgChat):
			text := strings.TrimSpace(line[len(MsgChat):])
			if text != "" {
				m.chatLines = append(m.chatLines, "Соперник: "+text)
			}
			return m, readPvPLineCmd(m.session)

		case strings.HasPrefix(line, MsgEnd):
			e, _ := ParseEnd(line)
			if !m.gameOver {
				m.gameOver = true
				m.winnerSide = e.Winner
				m.state = FightViewEnd
				if m.session != nil {
					_ = m.session.Close()
					m.session = nil
				}
				sound.StopMusic()
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					sound.StopMusic()
					return ViewChangeMsg{View: ViewMainMenu}
				})
			}
			return m, readPvPLineCmd(m.session)

		default:

			return m, readPvPLineCmd(m.session)
		}
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.gameOver && m.state == FightViewEnd {
		sound.StopMusic()
		return m, func() tea.Msg {
			sound.StopMusic()
			return ViewChangeMsg{View: ViewMainMenu}
		}
	}

	if m.chatFocused {
		switch keyMsg.String() {
		case "enter":
			text := strings.TrimSpace(string(m.chatInput))
			m.chatInput = m.chatInput[:0]
			m.chatFocused = false
			if text != "" {
				text = string(fixRunesForWindows([]rune(text)))
				m.chatLines = append(m.chatLines, "Вы: "+text)
				_ = m.pvpSend("CHAT " + text)
			}
		case "esc":
			m.chatFocused = false
		case "backspace":
			if len(m.chatInput) > 0 {
				m.chatInput = m.chatInput[:len(m.chatInput)-1]
			}
		default:
			fixedRunes := fixRunesForWindows(keyMsg.Runes)
			m.chatInput = append(m.chatInput, fixedRunes...)
		}
		return m, nil
	}

	if keyMsg.String() == "t" || keyMsg.String() == "T" {
		m.chatFocused = true
		return m, nil
	}

	if keyMsg.String() == "ctrl+c" || keyMsg.String() == "esc" {
		if m.session != nil {
			_ = m.session.Close()
			m.session = nil
		}
		sound.StopMusic()
		return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
	}

	if m.connectionErr != "" {
		if keyMsg.String() == "enter" || keyMsg.String() == " " {
			return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
		}
		return m, nil
	}

	if m.waitingForMatch {
		return m, nil
	}

	if m.waitingForState {
		return m, nil
	}

	if !m.myTurn() {
		return m, nil
	}

	switch m.state {
	case FightViewActionMenu:
		return m.updatePvPActionMenu(keyMsg)
	case FightViewItemMenu:
		return m.updatePvPItemMenu(keyMsg)
	case FightViewSurrenderConfirm:
		return m.updatePvPSurrender(keyMsg)
	case FightViewExitConfirm:
		return m.updatePvPExitConfirm(keyMsg)
	}

	return m, nil
}

func (m *PvPFightModel) updatePvPActionMenu(msg tea.KeyMsg) (*PvPFightModel, tea.Cmd) {
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
			attackPart := m.player.Hit()
			blockPart := m.enemy.Block()
			dmg := 0

			if attackPart == blockPart {
				m.message = fmt.Sprintf("🛡️ %s заблокировал удар!", m.enemy.GetName())
			} else {
				dmg = m.turnHandler.CalculateDamage(m.player, m.enemy, attackPart)
				m.enemy.TakeDamage(dmg)
				m.message = fmt.Sprintf("💥 Вы нанесли %d урона! %s: %d/%d HP",
					dmg, m.enemy.GetName(), m.enemy.GetHP(), m.enemy.GetMaxHP())
			}

			_ = m.pvpSend(SerializeAction(Action{
				Kind:      "attack",
				BodyPart:  attackPart,
				BlockPart: blockPart,
				Damage:    dmg,
			}))

			m.showMessage = true
			m.round++
			m.sendState()

			if m.gameOver {
				m.state = FightViewEnd
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return ViewChangeMsg{View: ViewMainMenu}
				})
			}

			return m, pvpScheduleHideMessage()

		case 1:
			m.state = FightViewItemMenu
			m.itemSelected = 0
		case 2:
			m.showMessage = true
			m.message = m.getPvPStats()
			return m, pvpScheduleHideMessage()
		case 3:
			m.state = FightViewSurrenderConfirm
			m.selected = 0
		}
	case "esc":
		m.state = FightViewExitConfirm
	}
	return m, nil
}

func (m *PvPFightModel) updatePvPItemMenu(msg tea.KeyMsg) (*PvPFightModel, tea.Cmd) {
	usable, equippable := m.getPvPItemLists()
	total := len(usable) + len(equippable)

	if total == 0 {
		m.state = FightViewActionMenu
		return m, nil
	}

	if m.itemSelected >= total {
		m.itemSelected = total - 1
	}

	switch msg.String() {
	case "up", "k":
		if m.itemSelected > 0 {
			m.itemSelected--
		}
	case "down", "j":
		if m.itemSelected < total-1 {
			m.itemSelected++
		}
	case "enter", " ":
		if m.itemSelected < len(usable) {
			item := usable[m.itemSelected]
			success := m.itemManager.UseItem(m.player, m.enemy, item)
			if success {
				if _, err := m.player.GetInventory().RemoveItem(item.Template.ID); err != nil {
					m.message = fmt.Sprintf("✅ %s использован (предмет не удалён из инвентаря: %v)", item.Template.Name, err)
				} else {
					m.message = fmt.Sprintf("✅ %s использован!", item.Template.Name)
				}
				m.showMessage = true
				m.state = FightViewActionMenu

				_ = m.pvpSend(SerializeAction(Action{Kind: "item", ItemIdx: m.itemSelected}))
				m.round++
				m.sendState()

				if m.gameOver {
					m.state = FightViewEnd
					return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return ViewChangeMsg{View: ViewMainMenu}
					})
				}

				return m, pvpScheduleHideMessage()
			}
			m.message = fmt.Sprintf("❌ Не удалось использовать %s", item.Template.Name)
			m.showMessage = true
			return m, pvpScheduleHideMessage()
		} else {
			idx := m.itemSelected - len(usable)
			toEquip := equippable[idx]
			if toEquip != nil && toEquip.Template != nil {
				_ = m.player.EquipItem(toEquip.Template.ID)
				m.player.CalculateStats()
				m.message = "Экипирован: " + toEquip.Template.Name
				m.showMessage = true
				m.state = FightViewActionMenu
				return m, pvpScheduleHideMessage()
			}
		}
	case "esc":
		m.state = FightViewActionMenu
	}
	return m, nil
}

func (m *PvPFightModel) updatePvPSurrender(msg tea.KeyMsg) (*PvPFightModel, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		m.winnerSide = 3 - m.MySide
		_ = m.pvpSend(SerializeAction(Action{Kind: "surrender"}))
		_ = m.pvpSend(SerializeEnd(End{Winner: m.winnerSide}))
		time.Sleep(100 * time.Millisecond)
		if m.session != nil {
			_ = m.session.Close()
			m.session = nil
		}
		m.gameOver = true
		m.state = FightViewEnd
		m.message = "Вы сдались"
		m.showMessage = true
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ViewChangeMsg{View: ViewMainMenu}
		})
	case "esc":
		m.state = FightViewActionMenu
	}
	return m, nil
}

func (m *PvPFightModel) updatePvPExitConfirm(msg tea.KeyMsg) (*PvPFightModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "д", "Д":
		m.winnerSide = 3 - m.MySide
		_ = m.pvpSend(SerializeAction(Action{Kind: "surrender"}))
		_ = m.pvpSend(SerializeEnd(End{Winner: m.winnerSide}))
		time.Sleep(100 * time.Millisecond)
		if m.session != nil {
			_ = m.session.Close()
			m.session = nil
		}
		return m, func() tea.Msg { return ViewChangeMsg{View: ViewMainMenu} }
	case "n", "N", "н", "Н", "esc":
		m.state = FightViewActionMenu
	}
	return m, nil
}

func (m *PvPFightModel) getPvPItemLists() (usable []*Item.Item, equippable []*Item.Item) {
	usable = m.itemManager.GetUsableItems(m.player.GetInventory().GetItems())
	equippable = m.player.GetInventory().FindEquippableItems()
	return usable, equippable
}

func (m *PvPFightModel) getPvPStats() string {
	return fmt.Sprintf("📊 Раунд %d  │  Вы: %d/%d  │  Соперник: %d/%d",
		m.round, m.player.GetHP(), m.player.GetMaxHP(), m.enemy.GetHP(), m.enemy.GetMaxHP())
}

func (m *PvPFightModel) View() string {
	if m.state == FightViewEnd {
		return m.renderPvPEndScreen()
	}
	return m.renderPvPBattleScreen()
}

func (m *PvPFightModel) renderPvPBattleScreen() string {
	var b strings.Builder
	w := m.Width
	if w < ui.MinWidth {
		w = ui.MinWidth
	}
	h := m.Height
	if h < ui.MinHeight {
		h = ui.MinHeight
	}
	if m.connectionErr != "" {
		b.WriteString(m.centerPvPText(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorDanger)).Render(m.connectionErr), w))
		return b.String()
	}
	if m.waitingForMatch {
		b.WriteString(m.centerPvPText("Поиск противника... ESC — отмена", w))
		return b.String()
	}
	title := "⚔️ PvP  РАУНД " + fmt.Sprintf("%d", m.round)
	var status string
	if m.waitingForState {
		status = "  │  Ожидание ответа противника"
	} else if m.turn != m.MySide {
		status = "  │  Ожидание хода противника"
	} else {
		status = "  │  Ваш ход"
	}
	title += status
	b.WriteString(m.centerPvPText(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorTitle)).Bold(true).Render(title), w))
	b.WriteString("\n\n")

	enemyName := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorDanger)).Bold(true).Render("◆ " + m.enemy.GetName())
	b.WriteString(ui.RenderBattleHpLine(enemyName, m.enemy.GetHP(), m.enemy.GetMaxHP(), w))
	b.WriteString("\n")

	vsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorBorder)).Bold(true)
	b.WriteString(m.centerPvPText(vsStyle.Render("────── VS ──────"), w))
	b.WriteString("\n")

	playerName := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Bold(true).Render("◆ " + m.player.GetName())
	b.WriteString(ui.RenderBattleHpLine(playerName, m.player.GetHP(), m.player.GetMaxHP(), w))
	b.WriteString("\n\n")

	canAct := m.myTurn() && !m.waitingForState
	if canAct {
		switch m.state {
		case FightViewActionMenu:
			b.WriteString(m.renderPvPActionMenu())
		case FightViewItemMenu:
			b.WriteString(m.renderPvPItemMenu())
		case FightViewSurrenderConfirm:
			b.WriteString(m.renderPvPSurrenderConfirm())
		case FightViewExitConfirm:
			b.WriteString(m.renderPvPExitConfirm())
		}
	} else if m.state == FightViewExitConfirm {
		b.WriteString(m.renderPvPExitConfirm())
	} else if m.waitingForState {
		b.WriteString(m.centerPvPText(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render("Ожидание ответа противника..."), w))
	} else {
		b.WriteString(m.centerPvPText(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render("Ожидание хода противника..."), w))
	}

	if m.showMessage && m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(m.centerPvPText(m.message, w))
	}

	b.WriteString("\n\n")
	help := "↑↓ Enter T Чат ESC Выход"
	if m.state == FightViewExitConfirm {
		help = "Y — выйти   N/ESC — остаться"
	}
	b.WriteString(m.centerPvPText(lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render(help), w))

	if h > 20 {
		b.WriteString("\n\n── Чат (T) ──")
		for i := len(m.chatLines) - 1; i >= 0 && (len(m.chatLines)-1-i) < 5; i-- {
			b.WriteString("\n" + m.chatLines[i])
		}
		if m.chatFocused {
			b.WriteString("\n> " + string(m.chatInput) + "▌")
		}
	}
	return b.String()
}

func (m *PvPFightModel) centerPvPText(text string, width int) string {
	r := []rune(text)
	if len(r) >= width {
		return string(r[:width])
	}
	pad := (width - len(r)) / 2
	return strings.Repeat(" ", pad) + text + strings.Repeat(" ", width-pad-len(r))
}

func (m *PvPFightModel) renderPvPActionMenu() string {
	items := []string{"1. Атаковать", "2. Предмет", "3. Статистика", "4. Сдаться"}
	var b strings.Builder
	for i, it := range items {
		b.WriteString(ui.RenderMenuItem(i == m.selected, it) + "\n")
	}
	return b.String()
}

func (m *PvPFightModel) renderPvPItemMenu() string {
	usable, equippable := m.getPvPItemLists()
	total := len(usable) + len(equippable)
	if total == 0 {
		return "Нет предметов\n"
	}
	var b strings.Builder
	n := 0
	for _, item := range usable {
		name := item.Template.Name + " (использовать)"
		b.WriteString(ui.RenderMenuItem(n == m.itemSelected, name) + "\n")
		n++
	}
	for _, item := range equippable {
		if item.Template != nil {
			b.WriteString(ui.RenderMenuItem(n == m.itemSelected, item.Template.Name+" (экипировать)") + "\n")
			n++
		}
	}
	return b.String()
}

func (m *PvPFightModel) renderPvPSurrenderConfirm() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render("Сдаться? Enter — да, ESC — нет") + "\n"
}

func (m *PvPFightModel) renderPvPExitConfirm() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render("🚪 Выйти? Y — да   N/ESC — нет") + "\n"
}

func (m *PvPFightModel) renderPvPEndScreen() string {
	var b strings.Builder
	w := m.Width
	if w < ui.MinWidth {
		w = ui.MinWidth
	}
	title := "💀 Поражение"
	if m.winnerSide == m.MySide {
		title = "🎉 Победа!"
	}
	b.WriteString(m.centerPvPText(title, w))
	b.WriteString("\n\n")
	b.WriteString(m.centerPvPText("Через 2 сек — в меню", w))
	return b.String()
}

func (m *PvPFightModel) Disconnect() {
	if m.session != nil {
		_ = m.session.Close()
		m.session = nil
	}
}
