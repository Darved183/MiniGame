package game

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MyGame/game/ui"
	"MyGame/utils"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type ChatModel struct {
	Width          int
	Height         int
	addr           string
	conn           net.Conn
	reader         *bufio.Reader
	status         string
	username       string
	nameInput      []rune
	nameError      string
	input          []rune
	messages       []string
	showNamePicker bool
	connectError   string
}

func chatServers() []string {

	if s := os.Getenv("CHAT_SERVERS"); s != "" {
		return strings.Split(s, ",")
	}

	if s := os.Getenv("CHAT_HOSTS"); s != "" {
		return strings.Split(s, ",")
	}

	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("CHAT_PORT")
	if port == "" {
		port = "8081"
	}

	return []string{host + ":" + port}
}

func FixRunesForWindows(runes []rune) []rune {
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

type chatConnectedMsg struct {
	conn net.Conn
	err  error
}
type chatIncomingMsg struct {
	line string
	err  error
}

func NewChatModel() *ChatModel {
	username := strings.TrimSpace(os.Getenv("CHAT_NAME"))

	return &ChatModel{
		Width:          ui.MinWidth,
		Height:         ui.MinHeight,
		addr:           "",
		status:         "",
		username:       username,
		nameInput:      []rune("Гость"),
		input:          make([]rune, 0, 128),
		messages:       []string{},
		showNamePicker: username == "",
		connectError:   "",
	}
}

func (m *ChatModel) Init() tea.Cmd {
	utils.SetUTF8CodePage()
	utils.CloseInputManager()
	utils.ClearInputBuffer()

	if strings.TrimSpace(m.username) != "" {
		m.showNamePicker = false
		m.status = "Подключение..."
		m.messages = append(m.messages, "Подключение к серверу...")
		return connectChatWithFallbackCmd()
	}

	m.showNamePicker = true
	m.status = "Введите имя"
	m.messages = append(m.messages, "Введите имя пользователя и нажмите Enter.")
	return nil
}

func (m *ChatModel) Disconnect() {
	if m.conn != nil {
		m.conn.Close()
	}
	m.conn = nil
	m.reader = nil
	m.status = "Отключено"
}

func connectChatWithFallbackCmd() tea.Cmd {
	return func() tea.Msg {
		servers := chatServers()
		var lastErr error
		for _, addr := range servers {
			conn, err := net.DialTimeout("tcp", strings.TrimSpace(addr), 5*time.Second)
			if err == nil {
				return chatConnectedMsg{conn: conn, err: nil}
			}
			lastErr = err
		}
		return chatConnectedMsg{conn: nil, err: fmt.Errorf("не удалось подключиться ни к одному серверу (%v): %w", servers, lastErr)}
	}
}

var serverAddrPrefix = regexp.MustCompile(`^\[.*?:\d+\]\s*`)

func stripServerAddressPrefix(line string) string {
	return strings.TrimSpace(serverAddrPrefix.ReplaceAllString(line, ""))
}

func readChatLineCmd(reader *bufio.Reader) tea.Cmd {
	return func() tea.Msg {
		line, err := reader.ReadString('\n')
		if err != nil {
			return chatIncomingMsg{err: err}
		}
		return chatIncomingMsg{line: strings.TrimRight(line, "\r\n")}
	}
}

func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chatConnectedMsg:
		if msg.err != nil {
			m.connectError = msg.err.Error()
			m.status = "Ошибка подключения"
			m.messages = append(m.messages, fmt.Sprintf("❌ %s", m.connectError))
			return m, nil
		}
		m.conn = msg.conn
		m.reader = bufio.NewReaderSize(m.conn, 4096)
		m.status = "Подключено"
		m.messages = append(m.messages, fmt.Sprintf("✅ Подключено как %s", m.username))
		m.connectError = ""
		return m, readChatLineCmd(m.reader)

	case chatIncomingMsg:
		if msg.err != nil {
			if m.conn != nil {
				m.conn.Close()
			}
			m.conn = nil
			m.reader = nil
			m.status = "Отключено"
			m.messages = append(m.messages, fmt.Sprintf("⚠️ Соединение закрыто: %v", msg.err))
			return m, nil
		}
		if line := stripServerAddressPrefix(msg.line); line != "" {
			m.messages = append(m.messages, line)
		}
		if m.reader != nil {
			return m, readChatLineCmd(m.reader)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
		if m.showNamePicker {
			return m.updateNamePicker(msg)
		}
		if m.connectError != "" && (msg.String() == "r" || msg.String() == "R") {
			m.connectError = ""
			m.status = "Подключение..."
			m.messages = append(m.messages, "Повторная попытка подключения...")
			return m, connectChatWithFallbackCmd()
		}
		return m.updateChatInput(msg)
	}
	return m, nil
}

func (m *ChatModel) updateNamePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyBackspace:
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	case tea.KeySpace:
		m.nameInput = append(m.nameInput, ' ')
		if utf8.RuneCountInString(string(m.nameInput)) > 20 {
			m.nameInput = m.nameInput[:20]
		}
	case tea.KeyEnter:
		if name := strings.TrimSpace(string(m.nameInput)); name != "" {
			if utf8.RuneCountInString(name) > 20 {
				name = string([]rune(name)[:20])
			}
			m.username, m.showNamePicker, m.nameError = name, false, ""
			m.status = "Подключение..."
			m.messages = append(m.messages, fmt.Sprintf("Вы вошли как %s", m.username), "Подключение к серверу...")
			return m, connectChatWithFallbackCmd()
		}
		m.nameError = "Имя не может быть пустым"
	case tea.KeyRunes:
		m.nameInput = append(m.nameInput, FixRunesForWindows(msg.Runes)...)
		if utf8.RuneCountInString(string(m.nameInput)) > 20 {
			m.nameInput = m.nameInput[:20]
		}
	}
	return m, nil
}

func (m *ChatModel) updateChatInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	case tea.KeySpace:
		m.input = append(m.input, ' ')
	case tea.KeyEnter:
		text := strings.TrimSpace(string(m.input))
		if text == "" {
			break
		}
		if m.conn == nil {
			m.messages = append(m.messages, "⚠️ Нет подключения. Нажмите R для повторной попытки или ESC для выхода.")
			break
		}
		m.input = m.input[:0]
		payload := fmt.Sprintf("%s: %s\n", m.username, text)
		if _, err := m.conn.Write([]byte(payload)); err != nil {
			m.messages = append(m.messages, fmt.Sprintf("❌ Ошибка отправки: %v", err))
		} else {

		}
	case tea.KeyRunes:
		m.input = append(m.input, FixRunesForWindows(msg.Runes)...)
	}
	return m, nil
}

func chatHeaderStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorTitle)).Bold(true).Width(width).Align(lipgloss.Center).Padding(0, 1)
}

func (m *ChatModel) View() string {
	width, height := max(m.Width, ui.MinWidth), max(m.Height, ui.MinHeight)
	if m.showNamePicker {
		return m.viewNamePicker(width, height)
	}
	return m.viewChat(width, height)
}

func (m *ChatModel) viewNamePicker(w, h int) string {
	var b strings.Builder
	b.WriteString(chatHeaderStyle(w).Render("💬 Подключение к чату"))
	b.WriteString("\n\n")

	name := string(m.nameInput)
	if utf8.RuneCountInString(name) > w-10 {
		r := []rune(name)
		name = string(r[len(r)-(w-10):])
	}

	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorTitle)).Render("Ваше имя: ")
	inputField := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorStats)).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Render(name + "█")

	line := prompt + inputField
	padding := (w - len([]rune(line))) / 2
	if padding < 0 {
		padding = 0
	}
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(line)
	b.WriteString("\n")

	if m.nameError != "" {
		errLine := ui.WarningStyle.Render("  " + m.nameError)
		errPadding := (w - len([]rune(errLine))) / 2
		if errPadding < 0 {
			errPadding = 0
		}
		b.WriteString(strings.Repeat(" ", errPadding))
		b.WriteString(errLine)
		b.WriteString("\n")
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render("Enter — войти   ESC — назад")
	helpPadding := (w - len([]rune(help))) / 2
	if helpPadding < 0 {
		helpPadding = 0
	}
	b.WriteString("\n" + strings.Repeat(" ", helpPadding) + help)

	lines := strings.Count(b.String(), "\n")
	if lines < h-2 {
		topPad := (h - 2 - lines) / 2
		if topPad > 0 {
			return strings.Repeat("\n", topPad) + b.String()
		}
	}
	return b.String()
}

func (m *ChatModel) viewChat(w, h int) string {
	user := strings.TrimSpace(m.username)
	if user == "" {
		user = "Гость"
	}

	var b strings.Builder
	b.WriteString(chatHeaderStyle(w).Render(fmt.Sprintf("💬 Чат  ·  %s", user)))
	statusLine := fmt.Sprintf("  %s", m.status)
	if m.connectError != "" {
		statusLine += "  │  R — повторить"
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render(statusLine) + "\n\n")

	msgContent := m.renderLastLines(w-6, h-6)
	if msgContent == "" {
		msgContent = " "
	}
	msgBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ui.ColorBorder)).
		Padding(0, 1).Width(w - 4).Render(msgContent)
	b.WriteString(msgBox + "\n")

	input := string(m.input)
	if utf8.RuneCountInString(input) > w-12 {
		r := []rune(input)
		input = string(r[len(r)-(w-12):])
	}

	inputBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(ui.ColorSelectedBg)).
		Padding(0, 1).Width(w - 4).Render("> " + input + "▌")
	b.WriteString(inputBox)

	return b.String()
}

func (m *ChatModel) renderLastLines(maxW, maxL int) string {
	if maxW <= 0 || maxL <= 0 {
		return " "
	}

	var wrapped []string
	for _, msg := range m.messages {
		for _, line := range wrapRunes(msg, maxW) {
			switch {
			case strings.HasPrefix(msg, "Вы: "):
				wrapped = append(wrapped, lipgloss.NewStyle().
					Foreground(lipgloss.Color(ui.ColorStats)).Render(line))
			case strings.HasPrefix(msg, "✅") || strings.HasPrefix(msg, "❌") || strings.HasPrefix(msg, "⚠️"):
				wrapped = append(wrapped, lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorHelp)).Render(line))
			default:
				wrapped = append(wrapped, line)
			}
		}
	}

	if len(wrapped) > maxL {
		wrapped = wrapped[len(wrapped)-maxL:]
	}
	out := strings.Join(wrapped, "\n")
	if out != "" && out != " " {
		return out
	}
	return " "
}

func wrapRunes(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	r := []rune(s)
	if len(r) <= width {
		return []string{s}
	}

	var out []string
	for len(r) > 0 {
		n := width
		if n > len(r) {
			n = len(r)
		}
		out = append(out, string(r[:n]))
		r = r[n:]
	}
	return out
}
