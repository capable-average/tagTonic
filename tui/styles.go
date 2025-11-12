package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#3B82F6") // Blue
	ColorAccent    = lipgloss.Color("#10B981") // Green

	// Status colors
	ColorSuccess = lipgloss.Color("#10B981") // Green
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue

	// UI colors
	ColorBorder      = lipgloss.Color("#6B7280") // Gray
	ColorBorderLight = lipgloss.Color("#9CA3AF") // Light gray
	ColorBackground  = lipgloss.Color("#1F2937") // Dark gray
	ColorText        = lipgloss.Color("#F9FAFB") // Almost white
	ColorTextMuted   = lipgloss.Color("#9CA3AF") // Gray
	ColorHighlight   = lipgloss.Color("#8B5CF6") // Light purple

	// Special
	ColorSelected = lipgloss.Color("#7C3AED") // Purple
	ColorModified = lipgloss.Color("#F59E0B") // Amber
)

type Theme struct {
	PanelBorder      lipgloss.Border
	PanelBorderColor lipgloss.Color
	PanelPadding     []int

	TitleStyle      lipgloss.Style
	HeaderStyle     lipgloss.Style
	NormalTextStyle lipgloss.Style
	MutedTextStyle  lipgloss.Style
	HighlightStyle  lipgloss.Style

	SelectedItemStyle lipgloss.Style
	ActivePanelStyle  lipgloss.Style
	StatusBarStyle    lipgloss.Style
	ErrorStyle        lipgloss.Style
	SuccessStyle      lipgloss.Style
	WarningStyle      lipgloss.Style

	FieldLabelStyle lipgloss.Style
	FieldValueStyle lipgloss.Style
	EditingStyle    lipgloss.Style
	ModifiedStyle   lipgloss.Style
	HelpStyle       lipgloss.Style
}

func DefaultTheme() *Theme {
	return &Theme{
		PanelBorder:      lipgloss.RoundedBorder(),
		PanelBorderColor: ColorBorder,
		PanelPadding:     []int{0, 1},

		TitleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1),

		HeaderStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary).
			Padding(0, 1),

		NormalTextStyle: lipgloss.NewStyle().
			Foreground(ColorText),

		MutedTextStyle: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		HighlightStyle: lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true),

		SelectedItemStyle: lipgloss.NewStyle().
			Foreground(ColorSelected).
			Bold(true).
			Background(lipgloss.Color("#312E81")). // Dark purple
			Padding(0, 1),

		ActivePanelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1),

		StatusBarStyle: lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorText).
			Padding(0, 1),

		ErrorStyle: lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true),

		SuccessStyle: lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true),

		WarningStyle: lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true),

		FieldLabelStyle: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(12).
			Align(lipgloss.Right),

		FieldValueStyle: lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(false),

		EditingStyle: lipgloss.NewStyle().
			Foreground(ColorAccent).
			Background(lipgloss.Color("#064E3B")). // Dark green
			Bold(true),

		ModifiedStyle: lipgloss.NewStyle().
			Foreground(ColorModified).
			Bold(true),

		HelpStyle: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true),
	}
}

const (
	IconFile       = "📄"
	IconFolder     = "📁"
	IconMusic      = "🎵"
	IconCheck      = "✓"
	IconCross      = "✗"
	IconArrowRight = "▶"
	IconSearch     = "🔍"
)

func BorderedBox(content string, width, height int, style lipgloss.Style) string {
	return style.
		Width(width).
		Height(height).
		Render(content)
}

func SuccessText(text string, theme *Theme) string {
	return theme.SuccessStyle.Render(IconCheck + " " + text)
}

func ErrorText(text string, theme *Theme) string {
	return theme.ErrorStyle.Render(IconCross + " " + text)
}

func WarningText(text string, theme *Theme) string {
	return theme.WarningStyle.Render("⚠ " + text)
}

func InfoText(text string, theme *Theme) string {
	return theme.NormalTextStyle.Render("ℹ " + text)
}

func HighlightText(text string, theme *Theme) string {
	return theme.HighlightStyle.Render(text)
}

func RenderProgressBar(current, total int, width int, theme *Theme) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(float64(width-2) * percentage)
	empty := width - 2 - filled

	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	bar += "]"

	return theme.HighlightStyle.Render(bar)
}

func StatusBadge(text string, statusType string, theme *Theme) string {
	var style lipgloss.Style

	switch statusType {
	case "success":
		style = theme.SuccessStyle.Copy().Background(lipgloss.Color("#065F46"))
	case "error":
		style = theme.ErrorStyle.Copy().Background(lipgloss.Color("#7F1D1D"))
	case "warning":
		style = theme.WarningStyle.Copy().Background(lipgloss.Color("#78350F"))
	case "info":
		style = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Background(lipgloss.Color("#1E3A8A")).
			Bold(true)
	default:
		style = theme.NormalTextStyle
	}

	return style.Padding(0, 1).Render(text)
}

func KeyHelp(key, description string, theme *Theme) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("#1F2937"))

	descStyle := theme.MutedTextStyle

	return keyStyle.Render(key) + " " + descStyle.Render(description)
}

func Separator(width int, char string, color lipgloss.Color) string {
	style := lipgloss.NewStyle().Foreground(color)
	line := ""
	for i := 0; i < width; i++ {
		line += char
	}
	return style.Render(line)
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
