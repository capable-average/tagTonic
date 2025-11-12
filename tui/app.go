package tui

import (
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
	currentMode    Mode
	previousMode   Mode
	layout         *Layout

	statusMessage string
	statusTimeout int
	isLoading     bool
	lyricsHasFocus bool
	theme         *Theme

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
	return ""
}

func (a *App) renderMiddlePanel(width, height, tagHeight int) string {
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
