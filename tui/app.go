package tui

import (
	"os"
	"path/filepath"
	"tagTonic/config"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
)

type App struct {

	currentMode    Mode
	previousMode   Mode
	layout         *Layout

	statusMessage string
	statusTimeout int
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
	return ""
}

func (a *App) renderMiddlePanel(width, height, tagHeight int) string {
	return ""
}

func (a *App) renderStatusBar() string {
	return ""
}

func (a *App) renderArtworkPanel(width, height int) string {
	return ""
}

func (a *App) renderHelp() string {
	return `Help screen here`
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
