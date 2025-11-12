package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleKeyPress(key string) tea.Cmd {
	// Global keys
	switch key {
	case "ctrl+c", "q":
		return tea.Quit
	case "?":
		return a.toggleHelp()
	}

	return nil
}

func (a *App) toggleHelp() tea.Cmd {
	if a.currentMode == HelpMode {
		a.currentMode = a.previousMode
	} else {
		a.previousMode = a.currentMode
		a.currentMode = HelpMode
	}
	return nil
}
