package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tagTonic/config"
	"tagTonic/mp3"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
)

type App struct {
	fileBrowser  *FileBrowser
	tagEditor    *TagEditor
	mediaManager *MediaManager
	layout       *Layout
	cache        *Cache

	currentMode    Mode
	previousMode   Mode
	currentFile    *FileEntry
	lyricsHasFocus bool

	statusMessage string
	statusTimeout int
	theme         *Theme
	isLoading     bool

	isBatchProcessing bool
	batchTotal        int
	batchProcessed    int
	batchSucceeded    int
	batchFailed       int
	batchFilePaths    []string
	batchMode         string // "lyrics", "artwork", "both"

	config *config.Config
}

func NewApp(startDir string) *App {
	cache := NewCache(50)

	return &App{
		fileBrowser:  NewFileBrowser(startDir),
		tagEditor:    NewTagEditor(),
		mediaManager: NewMediaManager(cache),
		layout:       NewLayout(),
		cache:        cache,
		currentMode:  FileBrowserMode,
		theme:        DefaultTheme(),
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
		layout := a.layout.Calculate()

		clearKittyImages()

		xPos := layout.LeftPanelWidth + layout.MiddlePanelWidth + 3
		yPos := 4

		if layout.ShowArtwork {
			if cmd := a.mediaManager.HandleWindowResizeWithPosition(layout.ArtworkMaxWidth, layout.ArtworkMaxHeight, xPos, yPos); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		cmd := a.handleKeyPress(msg.String())
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ArtworkRenderMsg:
		a.mediaManager.UpdateArtworkResult(msg.Result)
		if msg.Result.Error != nil {
			a.setError("Artwork rendering failed", msg.Result.Error.Error())
		}

	case FileLoadedMsg:
		a.tagEditor.LoadTags(msg.Tags)
		a.isLoading = false

		layout := a.layout.Calculate()
		xPos := layout.LeftPanelWidth + layout.MiddlePanelWidth + 3
		yPos := 4

		if cmd := a.mediaManager.LoadFileAsyncWithPosition(
			msg.FilePath,
			msg.Tags.Lyrics,
			msg.Tags.Artwork,
			layout.ArtworkMaxWidth,
			layout.ArtworkMaxHeight,
			xPos,
			yPos,
		); cmd != nil {
			cmds = append(cmds, cmd)
		}

		if cmd := a.setStatus("Loaded: "+filepath.Base(msg.FilePath), 1); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case FileLoadErrorMsg:
		a.setError("Failed to read tags", msg.Error.Error())
		a.isLoading = false

	case ArtworkFetchedMsg:
		if msg.Error != nil {
			a.setError("Failed to fetch artwork", msg.Error.Error())
		} else if len(msg.Artwork) > 0 {
			a.tagEditor.UpdateArtwork(msg.Artwork)

			layout := a.layout.Calculate()
			xPos := layout.LeftPanelWidth + layout.MiddlePanelWidth + 3
			yPos := 4

			if cmd := a.mediaManager.LoadFileAsyncWithPosition(
				a.currentFile.Path,
				a.mediaManager.GetLyricsPanel().GetLyrics(),
				msg.Artwork,
				layout.ArtworkMaxWidth,
				layout.ArtworkMaxHeight,
				xPos,
				yPos,
			); cmd != nil {
				cmds = append(cmds, cmd)
			}

			if cmd := a.setStatus("Artwork fetched successfully", 1); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case LyricsFetchedMsg:
		if msg.Error != nil {
			a.setError("Failed to fetch lyrics", msg.Error.Error())
		} else if msg.Lyrics != "" {
			a.tagEditor.UpdateLyrics(msg.Lyrics)

			a.mediaManager.GetLyricsPanel().SetLyrics(msg.Lyrics)

			if cmd := a.setStatus("Lyrics fetched successfully", 1); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case BatchProcessMsg:
		a.batchProcessed++
		if msg.Success {
			a.batchSucceeded++
		} else {
			a.batchFailed++
			logrus.Errorf("Batch process failed for %s: %v", msg.FilePath, msg.Error)
		}

		if cmd := a.setStatus(fmt.Sprintf("Processing: %d/%d (✓%d ✗%d)",
			a.batchProcessed, a.batchTotal, a.batchSucceeded, a.batchFailed), 0); cmd != nil {
			cmds = append(cmds, cmd)
		}

		if a.batchProcessed < a.batchTotal {
			switch a.batchMode {
			case "lyrics":
				cmds = append(cmds, a.processBatchLyrics(a.batchFilePaths, a.batchProcessed))
			case "artwork":
				cmds = append(cmds, a.processBatchArtwork(a.batchFilePaths, a.batchProcessed))
			case "both":
				cmds = append(cmds, a.processBatchBoth(a.batchFilePaths, a.batchProcessed))
			}
		} else {
			cmds = append(cmds, func() tea.Msg {
				return BatchCompleteMsg{
					Total:     a.batchTotal,
					Succeeded: a.batchSucceeded,
					Failed:    a.batchFailed,
				}
			})
		}

	case BatchCompleteMsg:
		a.isBatchProcessing = false
		a.batchTotal = 0
		a.batchProcessed = 0
		a.batchSucceeded = 0
		a.batchFailed = 0

		var status string
		if msg.Failed == 0 {
			status = fmt.Sprintf("✓ Batch completed successfully: %d files processed", msg.Succeeded)
		} else if msg.Succeeded == 0 {
			status = fmt.Sprintf("✗ Batch failed: all %d files had errors", msg.Failed)
		} else {
			status = fmt.Sprintf("Batch completed with errors: %d succeeded, %d failed", msg.Succeeded, msg.Failed)
		}

		if cmd := a.setStatus(status, 3); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case StatusTickMsg:
		a.statusMessage = ""
		a.statusTimeout = 0
	}

	return a, tea.Batch(cmds...)
}

func (a *App) loadFile(file *FileEntry) tea.Cmd {
	a.currentFile = file
	a.isLoading = true

	clearKittyImages()

	statusCmd := a.setStatus("Loading: "+file.Name, 3)

	loadCmd := func() tea.Msg {
		result := make(chan tea.Msg, 1)

		go func() {
			tagEditor := mp3.NewTagEditor()
			tags, err := tagEditor.ReadTags(file.Path)
			if err != nil {
				result <- FileLoadErrorMsg{
					FilePath: file.Path,
					Error:    err,
				}
				return
			}

			result <- FileLoadedMsg{
				FilePath: file.Path,
				Tags:     tags,
			}
		}()

		select {
		case msg := <-result:
			return msg
		case <-time.After(5 * time.Second):
			return FileLoadErrorMsg{
				FilePath: file.Path,
				Error:    fmt.Errorf("tag reading timed out"),
			}
		}
	}

	return tea.Batch(statusCmd, loadCmd)
}

func (a *App) saveTags() tea.Cmd {
	if a.currentFile == nil {
		return nil
	}

	err := a.tagEditor.SaveTags(a.currentFile.Path)
	if err != nil {
		a.setError("Failed to save tags", err.Error())
		return nil
	}

	return a.setStatus("Tags saved", 2)
}

func (a *App) setStatus(message string, timeoutSeconds int) tea.Cmd {
	a.statusMessage = message
	a.statusTimeout = timeoutSeconds * 60

	if timeoutSeconds > 0 {
		duration := time.Duration(timeoutSeconds) * time.Second
		return tea.Tick(duration, func(t time.Time) tea.Msg {
			return StatusTickMsg{}
		})
	}
	return nil
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
	if a.fileBrowser.IsSearchMode() {
		header = IconSearch + " Search: " + theme.NormalTextStyle.Render(a.fileBrowser.GetSearchQuery())
	}
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

func (a *App) renderLyricsPanel(width, height int) string {
	theme := a.theme

	lyricsHeight := height

	lyricsPanel := a.mediaManager.GetLyricsPanel()
	var lyricsContent string
	var lyricsHeader string

	if lyricsPanel.HasLyrics() {
		scrollPercent := 0
		totalLines := lyricsPanel.GetTotalLines()
		if totalLines > 0 {
			scrollPercent = (lyricsPanel.GetScrollOffset() * 100) / totalLines
		}

		scrollIndicator := ""
		availableContentHeight := lyricsHeight - 4
		if totalLines > availableContentHeight {
			scrollIndicator = theme.MutedTextStyle.Render(fmt.Sprintf(" (%d%%)", scrollPercent))
		}
		lyricsHeader = IconLyrics + " Lyrics" + scrollIndicator

		visibleLines := lyricsPanel.GetVisibleLines(availableContentHeight)
		var styledLines []string

		maxLineWidth := width - 10
		if maxLineWidth < 10 {
			maxLineWidth = 10
		}

		for _, line := range visibleLines {
			displayLine := sanitizeLyricsLine(line)

			runes := []rune(displayLine)
			if len(runes) > maxLineWidth {
				displayLine = string(runes[:maxLineWidth-3]) + "..."
			}

			if strings.HasPrefix(strings.TrimSpace(displayLine), "[") && strings.Contains(displayLine, "]") {
				displayLine = theme.HighlightStyle.Render(displayLine)
			} else {
				displayLine = theme.NormalTextStyle.Render(displayLine)
			}

			styledLines = append(styledLines, "  "+displayLine)
		}

		lyricsContent = strings.Join(styledLines, "\n")
	} else {
		lyricsHeader = IconLyrics + " Lyrics"
		lyricsContent = theme.MutedTextStyle.Render("\n  No lyrics available\n  Press 'f' to fetch")
	}

	if a.lyricsHasFocus {
		lyricsHeader += " " + StatusBadge("FOCUSED", "info", theme)
	}

	lyricsBox := TitledBox(lyricsHeader, lyricsContent, width, lyricsHeight, theme)

	return lyricsBox
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
		if a.fileBrowser.IsBatchMode() {
			hints = []string{
				KeyHelp("↑↓", "navigate", theme),
				KeyHelp("Space", "select", theme),
				KeyHelp("f", "fetch both", theme),
				KeyHelp("Ctrl+L", "lyrics", theme),
				KeyHelp("Ctrl+A", "artwork", theme),
				KeyHelp("Esc", "exit batch", theme),
				KeyHelp("?", "help", theme),
				KeyHelp("q", "quit", theme),
			}
		} else {
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
	theme := a.theme

	if a.currentFile == nil {
		var lines []string
		lines = append(lines, "No artwork")

		contentHeight := height - 2
		for len(lines) < contentHeight {
			lines = append(lines, "")
		}

		emptyStyle := lipgloss.NewStyle().
			Border(theme.PanelBorder).
			BorderForeground(ColorBorder).
			Width(width-2).
			Padding(0, 1)
		return emptyStyle.Render(strings.Join(lines, "\n"))
	}

	artworkResult := a.mediaManager.GetArtworkResult()

	var contentLines []string
	if artworkResult.IsKitty && artworkResult.Error == nil {
	} else if artworkResult.Error != nil {
		content := theme.ErrorStyle.Render(IconCross + " " + artworkResult.Content)
		contentLines = append(contentLines, content)
	} else if len(artworkResult.ImageData) > 0 {
		contentLines = strings.Split(artworkResult.Content, "\n")
	} else {
		content := theme.MutedTextStyle.Render("No artwork available")
		contentLines = append(contentLines, content)
	}

	availableHeight := height - 2
	contentLinesCount := len(contentLines)

	topPadding := (availableHeight - contentLinesCount) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	var lines []string
	for i := 0; i < topPadding; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, contentLines...)
	for len(lines) < availableHeight {
		lines = append(lines, "")
	}

	borderStyle := lipgloss.NewStyle().
		Border(theme.PanelBorder).
		BorderForeground(ColorBorder).
		Width(width-2).
		Padding(0, 1).
		Align(lipgloss.Center)

	return borderStyle.Render(strings.Join(lines, "\n"))
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
║ Batch Mode (after selecting files with Space):               ║
║   f, Ctrl+F   Fetch lyrics and artwork for all               ║
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

Press esc key to return...`
}

func sanitizeLyricsLine(line string) string {
	// Remove null bytes
	line = strings.ReplaceAll(line, "\x00", "\n")
	line = strings.ReplaceAll(line, "\r", "\n")

	var result strings.Builder
	result.Grow(len(line))

	for _, r := range line {
		// Filter control chars (0x00-0x1F except tab) and DEL (0x7F)
		if r == '\t' || (r >= 0x20 && r != 0x7F) {
			result.WriteRune(r)
		} else if r < 0x20 || r == 0x7F {
			continue
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
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
