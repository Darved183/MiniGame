package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"MyGame/Struct/Character"
)

const (
	ColorTitle      = "213"
	ColorNormal     = "245"
	ColorSelected   = "229"
	ColorSelectedBg = "62"
	ColorHelp       = "238"
	ColorDanger     = "196"
	ColorWarning    = "220"
	ColorSuccess    = "46"
	ColorStats      = "159"
	ColorBorder     = "62"
)

const (
	MinWidth  = 120
	MinHeight = 70
)

func CenteredLine(line string, width int) string {
	lineWidth := len([]rune(line))
	padding := (width - lineWidth) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + line
}

func CenteredLineBuilder(b io.StringWriter, line string, width int) {
	b.WriteString(CenteredLine(line, width))
	b.WriteString("\n")
}

func CenteredLineByVisibleWidth(b io.StringWriter, line string, visibleWidth, totalWidth int) {
	padding := (totalWidth - visibleWidth) / 2
	if padding < 0 {
		padding = 0
	}
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(line)
	b.WriteString("\n")
}

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTitle)).
			Bold(true)

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorNormal))

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSelected)).
			Background(lipgloss.Color(ColorSelectedBg)).
			Bold(true)

	MenuItemSelected = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSelected)).Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorHelp)).
			Italic(true)

	DangerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDanger)).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Italic(true)
)

func RenderMenuItem(selected bool, text string) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	return MenuItemSelected.Render(prefix + text)
}

func centerText(text string, width int) string {
	r := []rune(text)
	if len(r) >= width {
		return string(r[:width])
	}
	pad := (width - len(r)) / 2
	return strings.Repeat(" ", pad) + text + strings.Repeat(" ", width-len(r)-pad)
}

func RenderCharacterBox(char *Character.Character, label string, width int) string {
	var b strings.Builder
	if width < MinWidth {
		width = MinWidth
	}
	hpPercent := float64(char.GetHP()) / float64(char.GetMaxHP())
	hpBarWidth := 35
	filled := int(float64(hpBarWidth) * hpPercent)
	empty := hpBarWidth - filled
	var hpBarColor string
	if hpPercent > 0.6 {
		hpBarColor = ColorSuccess
	} else if hpPercent > 0.3 {
		hpBarColor = ColorWarning
	} else {
		hpBarColor = ColorDanger
	}
	hpBar := lipgloss.NewStyle().Foreground(lipgloss.Color(hpBarColor)).Render(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTitle)).Bold(true)
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorStats))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorder))
	boxWidth := width - 4
	nameText := centerText(nameStyle.Render(fmt.Sprintf("%s: %s", label, char.GetName())), boxWidth-2)
	hpText := fmt.Sprintf("HP: [%s] %d%%", hpBar, int(hpPercent*100))
	statsText := fmt.Sprintf("❤️  %d/%d  │  ⚔️  %.1f  │  🛡️  %.1f", char.GetHP(), char.GetMaxHP(), char.GetAttack(), char.GetDefense())
	info := fmt.Sprintf("╔%s╗\n║%s║\n╠%s╣\n║ %s ║\n║ %s ║\n╚%s╝",
		borderStyle.Render(strings.Repeat("═", boxWidth)), nameText, borderStyle.Render(strings.Repeat("═", boxWidth)),
		centerText(hpText, boxWidth-2), centerText(statsStyle.Render(statsText), boxWidth-2), borderStyle.Render(strings.Repeat("═", boxWidth)))
	for _, line := range strings.Split(info, "\n") {
		pad := (width - len([]rune(line))) / 2
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad) + line + "\n")
	}
	return b.String()
}

func RenderBattleHpLine(name string, hp, maxHp int, width int) string {
	if maxHp <= 0 {
		maxHp = 1
	}
	hpPercent := float64(hp) / float64(maxHp)
	barLen := 24
	filled := int(float64(barLen) * hpPercent)
	if filled > barLen {
		filled = barLen
	}
	empty := barLen - filled
	var barColor string
	if hpPercent > 0.6 {
		barColor = ColorSuccess
	} else if hpPercent > 0.3 {
		barColor = ColorWarning
	} else {
		barColor = ColorDanger
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
	line := fmt.Sprintf("%s  [%s]  %d/%d", name, bar, hp, maxHp)
	return CenteredLine(line, width)
}
