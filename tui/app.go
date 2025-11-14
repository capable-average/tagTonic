package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tagTonic/config"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
)

type App struct {
	fileBrowser  *FileBrowser
	tagEditor    *TagEditor
	layout         *Layout
	
	currentMode    Mode
	previousMode   Mode
	currentFile    *FileEntry
	lyricsHasFocus bool

	statusMessage string
	statusTimeout int
	theme         *Theme
	isLoading     bool

	config *config.Config
}

func NewApp(startDir string) *App {

	return &App{
		currentMode:  FileBrowserMode,
		isLoading:    false,
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			a.layout.Update(msg.Width, msg.Height)
			return a, tea.Batch(cmds...)

		case tea.KeyMsg:
			cmd := a.handleKeyPress(msg.String())
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

		case FileLoadErrorMsg:
			a.setError("Failed to read tags", msg.Error.Error())
			a.isLoading = false

	}
	return a, tea.Batch(cmds...)
}

func (a *App) setError(message, details string) {
	errorMsg := message
	if details != "" {
		errorMsg += ": " + details
	}
	a.statusMessage = " ERROR: " + errorMsg
	a.statusTimeout = 0
}

func (a *App) View() string {
	if !a.layout.IsMinimumSize() {
		return "Terminal too small. Minimum size: 60x20"
	}

	layout := a.layout.Calculate()

	switch a.currentMode {
	case HelpMode:
		return a.renderHelp()
	default:
		return a.renderMainView(layout)
	}
}

func (a *App) renderMainView(layout AdaptiveLayout) string {
	leftPanel := a.renderFileBrowser(layout.LeftPanelWidth, layout.ContentHeight)

	middlePanel := a.renderMiddlePanel(layout.MiddlePanelWidth, layout.ContentHeight, layout.TagsPanelHeight)

	statusBar := a.renderStatusBar()

	var mainContent string

	if layout.ShowArtwork && layout.RightPanelWidth > 0 {
		rightPanel := a.renderArtworkPanel(layout.RightPanelWidth, layout.ContentHeight)

		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPanel,
			middlePanel,
			rightPanel,
		)
	} else {
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPanel,
			middlePanel,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
	)
}

func (a *App) renderFileBrowser(width, height int) string {
	theme := a.theme
	entries := a.fileBrowser.GetFilteredEntries()
	selectedIndex := a.fileBrowser.GetSelectedIndex()

	var lines []string

	header := IconFolder + " Files"
	if a.fileBrowser.IsBatchMode() {
		header += " " + StatusBadge("BATCH", "info", theme)
	}

	headerStyle := theme.HeaderStyle.Copy().Width(width - 4)
	headerLine := headerStyle.Render(header)
	lines = append(lines, headerLine)

	currentPath := a.fileBrowser.GetCurrentDir()
	maxPathLen := width - 8
	if len(currentPath) > maxPathLen {
		currentPath = "..." + currentPath[len(currentPath)-(maxPathLen-3):]
	}
	breadcrumb := theme.MutedTextStyle.Render("📍 " + currentPath)
	lines = append(lines, breadcrumb)

	sepWidth := width - 6
	if sepWidth < 1 {
		sepWidth = 1
	}
	lines = append(lines, Separator(sepWidth, "─", ColorBorderLight))

	contentHeight := height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}

	startIdx := 0
	if selectedIndex >= contentHeight {
		startIdx = selectedIndex - contentHeight + 1
	}
	endIdx := startIdx + contentHeight
	if endIdx > len(entries) {
		endIdx = len(entries)
	}

	for i := startIdx; i < endIdx; i++ {
		entry := entries[i]

		var lineContent string

		if i == selectedIndex {
			lineContent = IconArrowRight + " "
		} else {
			lineContent = "  "
		}

		if a.fileBrowser.IsBatchMode() {
			if a.fileBrowser.IsSelected(entry.Path) {
				lineContent += theme.SuccessStyle.Render("[✓] ")
			} else {
				lineContent += theme.MutedTextStyle.Render("[ ] ")
			}
		}

		icon := IconFile
		if entry.IsDir {
			icon = IconFolder
		} else if strings.HasSuffix(entry.Name, ".mp3") {
			icon = IconMusic
		}
		lineContent += icon + " "

		name := entry.Name
		maxNameLen := width - 25
		if maxNameLen < 10 {
			maxNameLen = 10
		}

		if entry.IsDir {
			displayName := name + "/"
			if len(displayName) > maxNameLen {
				displayName = displayName[:maxNameLen-3] + "..."
			}
			name = theme.HighlightStyle.Render(displayName)
		} else {
			displayName := name
			if len(displayName) > maxNameLen {
				displayName = displayName[:maxNameLen-3] + "..."
			}
			name = theme.NormalTextStyle.Render(displayName)
		}

		lineContent += name

		if !entry.IsDir && entry.Size > 0 {
			sizeStr := formatFileSize(entry.Size)
			lineContent += " " + theme.MutedTextStyle.Render(sizeStr)
		}

		if i == selectedIndex {
			lineContent = theme.SelectedItemStyle.Render(lineContent)
		}

		lines = append(lines, lineContent)
	}

	var footerLine string
	if len(entries) > 0 {
		footer := fmt.Sprintf("%d/%d files", selectedIndex+1, len(entries))
		if len(entries) > contentHeight {
			scrollPercent := float64(selectedIndex) / float64(len(entries)-1) * 100
			footer += fmt.Sprintf(" │ %d%%", int(scrollPercent))
		}
		footerLine = theme.MutedTextStyle.Render(footer)
	} else {
		footerLine = theme.MutedTextStyle.Render("No files")
	}

	emptyLinesNeeded := contentHeight - (endIdx - startIdx) - 1
	if emptyLinesNeeded < 0 {
		emptyLinesNeeded = 0
	}
	for i := 0; i < emptyLinesNeeded; i++ {
		lines = append(lines, "")
	}

	lines = append(lines, footerLine)

	content := strings.Join(lines, "\n")

	borderStyle := lipgloss.NewStyle().
		Border(theme.PanelBorder).
		Width(width-2).
		Height(height-2).
		Padding(0, 1)

	if a.currentMode == FileBrowserMode {
		borderStyle = borderStyle.BorderForeground(ColorPrimary)
	} else {
		borderStyle = borderStyle.BorderForeground(ColorBorder)
	}

	return borderStyle.Render(content)
}

func (a *App) renderMiddlePanel(width, height, tagHeight int) string {
	if a.currentFile == nil {
		var lines []string
		lines = append(lines, "No file selected")

		contentHeight := height - 2
		for len(lines) < contentHeight {
			lines = append(lines, "")
		}

		emptyStyle := lipgloss.NewStyle().
			Border(a.theme.PanelBorder).
			BorderForeground(ColorBorder).
			Width(width-2).
			Padding(0, 1)
		return emptyStyle.Render(strings.Join(lines, "\n"))
	}

	lyricsHeight := height - tagHeight

	tagsPanel := a.renderTagsPanel(width, tagHeight)

	lyricsPanel := a.renderLyricsPanel(width, lyricsHeight)

	return lipgloss.JoinVertical(lipgloss.Left, tagsPanel, lyricsPanel)
}

func (a *App) renderTagsPanel(width, height int) string {
	theme := a.theme
	var lines []string

	header := IconMusic + " Tags"
	if a.tagEditor.IsDirty() {
		header += " " + StatusBadge("MODIFIED", "warning", theme)
	}

	headerLine := theme.HeaderStyle.Render(header)
	lines = append(lines, headerLine)

	sepWidth := max(width-6, 1)
	lines = append(lines, Separator(sepWidth, "─", ColorBorderLight))

	fields := a.tagEditor.GetFields()
	editingField := a.tagEditor.GetEditingField()

	for i, field := range fields {
		var lineContent string
		if i == editingField && a.currentMode == TagEditMode {
			lineContent = IconArrowRight + " "
		} else {
			lineContent = "  "
		}

		label := theme.MutedTextStyle.Render(field.Name + ":")
		lineContent += label + " "

		value := field.Value
		if a.tagEditor.IsEditing() && i == editingField {
			value = a.tagEditor.GetEditBuffer() + theme.EditingStyle.Render("_")
		} else if value == "" {
			value = theme.MutedTextStyle.Render("(empty)")
		} else {
			maxValueLen := width - 20
			if len(value) > maxValueLen {
				value = value[:maxValueLen-3] + theme.MutedTextStyle.Render("...")
			}
			value = theme.FieldValueStyle.Render(value)
		}

		lineContent += value

		if a.tagEditor.IsEditing() && i == editingField {
			lineContent = theme.EditingStyle.Render(lineContent)
		}

		lines = append(lines, lineContent)

		if err := a.tagEditor.GetValidationError(field.Name); err != "" {
			errorLine := "  " + theme.ErrorStyle.Render(IconCross+" "+err)
			lines = append(lines, errorLine)
		}
	}

	var hints []string
	if a.tagEditor.IsDirty() {
		hints = append(hints, theme.SuccessStyle.Render("s:save"))
	}
	hints = append(hints, theme.MutedTextStyle.Render("e:edit"))
	hints = append(hints, theme.MutedTextStyle.Render("f:fetch"))

	footer := strings.Join(hints, theme.MutedTextStyle.Render(" │ "))
	footerLine := theme.MutedTextStyle.Render(footer)

	contentHeight := height - 2
	for len(lines) < contentHeight-1 {
		lines = append(lines, "")
	}

	lines = append(lines, footerLine)

	content := strings.Join(lines, "\n")

	borderStyle := lipgloss.NewStyle().
		Border(theme.PanelBorder).
		Width(width-2).
		Padding(0, 1)

	if a.currentMode == TagEditMode && !a.lyricsHasFocus {
		borderStyle = borderStyle.BorderForeground(ColorPrimary)
	} else {
		borderStyle = borderStyle.BorderForeground(ColorBorder)
	}

	return borderStyle.Render(content)
}

func (a *App) renderLyricsPanel(width, tagHeight int) string {
	return ""
}

func (a *App) renderStatusBar() string {
	theme := a.theme
	separator := theme.MutedTextStyle.Render(" │ ")

	if a.statusMessage != "" {
		if strings.HasPrefix(a.statusMessage, IconCross) {
			dismissHelp := theme.MutedTextStyle.Render(" │ Press ESC to dismiss")
			return theme.ErrorStyle.Render(a.statusMessage) + dismissHelp
		}

		if strings.Contains(a.statusMessage, "✓") ||
			strings.Contains(a.statusMessage, "successfully") ||
			strings.Contains(a.statusMessage, "saved") {
			return theme.SuccessStyle.Render(a.statusMessage)
		} else if strings.Contains(a.statusMessage, "✗") ||
			strings.Contains(a.statusMessage, "error") ||
			strings.Contains(a.statusMessage, "with errors") {
			return theme.ErrorStyle.Render(a.statusMessage)
		}
		return theme.NormalTextStyle.Render(a.statusMessage)
	}

	var hints []string

	switch a.currentMode {
	case FileBrowserMode:
		hints = []string{
			KeyHelp("↑↓", "navigate", theme),
			KeyHelp("Enter", "select", theme),
			KeyHelp("Tab", "tags", theme),
			KeyHelp("/", "search", theme),
			KeyHelp("b", "batch", theme),
			KeyHelp("h", "hidden", theme),
			KeyHelp("?", "help", theme),
			KeyHelp("q", "quit", theme),
		}
	case TagEditMode:
		if a.lyricsHasFocus {
			hints = []string{
				KeyHelp("↑↓/j/k", "scroll", theme),
				KeyHelp("PgUp/PgDn", "page", theme),
				KeyHelp("l", "unfocus", theme),
				KeyHelp("Tab", "files", theme),
				KeyHelp("s", "save", theme),
				KeyHelp("f", "fetch", theme),
				KeyHelp("?", "help", theme),
				KeyHelp("q", "quit", theme),
			}
		} else {
			hints = []string{
				KeyHelp("↑↓", "navigate", theme),
				KeyHelp("Enter/e", "edit", theme),
				KeyHelp("l", "lyrics", theme),
				KeyHelp("s", "save", theme),
				KeyHelp("u", "undo", theme),
				KeyHelp("r", "redo", theme),
				KeyHelp("f", "fetch", theme),
				KeyHelp("Tab", "files", theme),
				KeyHelp("?", "help", theme),
				KeyHelp("q", "quit", theme),
			}
		}
	case FieldEditMode:
		hints = []string{
			KeyHelp("Type", "edit", theme),
			KeyHelp("Enter", "save", theme),
			KeyHelp("Esc", "cancel", theme),
			KeyHelp("Ctrl+U", "clear", theme),
		}
	case SearchMode:
		hints = []string{
			KeyHelp("Type", "search", theme),
			KeyHelp("Enter", "done", theme),
			KeyHelp("Esc", "cancel", theme),
			KeyHelp("Ctrl+U", "clear", theme),
		}
	case LyricsEditMode:
		hints = []string{
			KeyHelp("Edit", "lyrics", theme),
			KeyHelp("Ctrl+S", "save", theme),
			KeyHelp("Esc", "cancel", theme),
		}
	case HelpMode:
		return theme.NormalTextStyle.Render("Press esc key to return")
	default:
		return theme.SuccessStyle.Render(IconCheck + " Ready")
	}

	return strings.Join(hints, separator)
}

func (a *App) renderArtworkPanel(width, height int) string {
	return ""
}

func (a *App) renderHelp() string {
	return `╔══════════════════════════════════════════════════════════════╗
║                        tagTonic TUI Help                     ║
╠══════════════════════════════════════════════════════════════╣
║ File Browser:                                                ║
║   ↑/↓, k/j    Navigate files                                 ║
║   PgUp/PgDn   Page up/down                                   ║
║   Enter       Open directory / Select MP3                    ║
║   /           Search files                                   ║
║   b           Toggle batch mode                              ║
║   Space       Select file (in batch mode)                    ║
║   h           Toggle hidden files                            ║
║   Tab         Switch to tag editor                           ║
║                                                              ║
║ Batch Mode (after selecting files with Space):              ║
║   f, Ctrl+F   Fetch lyrics and artwork for all              ║
║   Ctrl+L      Fetch lyrics only for all                      ║
║   Ctrl+A      Fetch artwork only for all                     ║
║   Esc         Exit batch mode                                ║
║                                                              ║
║ Tag Editor:                                                  ║
║   ↑/↓, k/j    Navigate fields                                ║
║   Enter, e    Edit field                                     ║
║   s, Ctrl+S   Save tags                                      ║
║   u           Undo last change                               ║
║   r           Redo last undone change                        ║
║   f, Ctrl+F   Fetch lyrics and artwork                       ║
║   Ctrl+L      Fetch lyrics only                              ║
║   Ctrl+A      Fetch artwork only                             ║
║   l           Focus lyrics panel                             ║
║   Tab         Switch to file browser                         ║
║                                                              ║
║ Field Editing:                                               ║
║   Type        Edit field content                             ║
║   Enter       Save changes                                   ║
║   Esc         Cancel editing                                 ║
║   Ctrl+U      Clear field                                    ║
║                                                              ║
║ Lyrics Panel:                                                ║
║   ↑/↓, k/j    Scroll lyrics                                  ║
║   PgUp/PgDn   Page up/down                                   ║
║   Home        Go to top                                      ║
║   e           Edit lyrics                                    ║
║                                                              ║
║ Global:                                                      ║
║   ?           Show/hide this help                            ║
║   Esc         Cancel current action                          ║
║   q, Ctrl+C   Quit application                               ║
╚══════════════════════════════════════════════════════════════╝

Press any key to return...`
}

func initLogging() error {
	logDir := filepath.Join(os.TempDir(), "tagTonic")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	logFile := filepath.Join(logDir, "tui.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	logrus.SetOutput(f)
	logrus.WithField("ts", time.Now().Format(time.RFC3339)).Info("tui session start")
	return nil
}

func Run() error {
	if err := initLogging(); err != nil {
		return err
	}

	cfg, _ := config.LoadConfig()
	startDir := "."
	if cfg != nil && cfg.DefaultDirectory != "" {
		startDir = cfg.DefaultDirectory
	} else if wd, err := os.Getwd(); err == nil {
		startDir = wd
	}

	app := NewApp(startDir)
	if cfg != nil {
		app.config = cfg
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()

	return err
}
