package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleKeyPress(key string) tea.Cmd {
	// Global
	switch key {
	case "ctrl+c", "q":
		return tea.Quit
	case "?":
		return a.toggleHelp()
	}

	if key == "esc" {
		switch a.currentMode {
		case SearchMode:
			return a.handleSearchKeys(key)
		case FieldEditMode:
			return a.handleFieldEditKeys(key)
		case LyricsEditMode:
			return a.handleLyricsEditKeys(key)
		default:
			return a.handleEscape()
		}
	}

	switch a.currentMode {
	case FileBrowserMode:
		return a.handleFileBrowserKeys(key)
	case TagEditMode:
		return a.handleTagEditKeys(key)
	case FieldEditMode:
		return a.handleFieldEditKeys(key)
	case SearchMode:
		return a.handleSearchKeys(key)
	case LyricsEditMode:
		return a.handleLyricsEditKeys(key)
	case HelpMode:
		return a.handleHelpKeys(key)
	}

	return nil
}

func (a *App) handleFileBrowserKeys(key string) tea.Cmd {
	switch key {
	case "up", "k":
		a.fileBrowser.MoveUp()
	case "down", "j":
		a.fileBrowser.MoveDown()
	case "pgup":
		a.fileBrowser.PageUp(10)
	case "pgdn":
		a.fileBrowser.PageDown(10)
	case "enter":
		if err := a.fileBrowser.Navigate(); err != nil {
			a.setError("Navigation failed", err.Error())
		} else {
			if file := a.fileBrowser.GetSelectedFile(); file != nil {
				return a.loadFile(file)
			}
		}
	case "tab":
		if a.currentFile != nil {
			a.currentMode = TagEditMode
		}
	case "/":
		a.currentMode = SearchMode
	case " ", "space":
		if a.fileBrowser.IsBatchMode() {
			a.fileBrowser.ToggleSelection()
			selectedCount := len(a.fileBrowser.GetSelectedFiles())
			return a.setStatus(fmt.Sprintf("Selected: %d file(s)", selectedCount), 1)
		}
	case "b":
		a.fileBrowser.ToggleBatchMode()
		return a.setStatus("Batch mode: "+map[bool]string{true: "ON", false: "OFF"}[a.fileBrowser.IsBatchMode()], 2)
	case "h":
		a.fileBrowser.ToggleHidden()
		return a.setStatus("Hidden files: "+map[bool]string{true: "ON", false: "OFF"}[a.fileBrowser.showHidden], 2)
	case "f", "ctrl+f":
		if a.fileBrowser.IsBatchMode() {
			return a.batchFetchBoth()
		} else if a.currentFile != nil {
			return a.fetchBoth()
		}
	case "ctrl+l":
		if a.fileBrowser.IsBatchMode() {
			return a.batchFetchLyrics()
		} else if a.currentFile != nil {
			return a.fetchLyricsOnly()
		}
	case "ctrl+a":
		if a.fileBrowser.IsBatchMode() {
			return a.batchFetchArtwork()
		} else if a.currentFile != nil {
			return a.fetchArtworkOnly()
		}
	}

	return nil
}

func (a *App) handleTagEditKeys(key string) tea.Cmd {
	if a.tagEditor.IsEditing() {
		return a.handleFieldEditKeys(key)
	}

	switch key {
	case "tab":
		if a.lyricsHasFocus {
			a.lyricsHasFocus = false
		} else {
			a.currentMode = FileBrowserMode
		}
	case "l":
		a.lyricsHasFocus = !a.lyricsHasFocus
		if a.lyricsHasFocus {
			return a.setStatus("Lyrics panel focused - use j/k to scroll, l to unfocus", 2)
		}
	case "up", "k":
		if a.lyricsHasFocus {
			a.mediaManager.GetLyricsPanel().ScrollUp()
		} else {
			a.tagEditor.MoveToPreviousField()
		}
	case "down", "j":
		if a.lyricsHasFocus {
			a.mediaManager.GetLyricsPanel().ScrollDown()
		} else {
			a.tagEditor.MoveToNextField()
		}
	case "pgup":
		if a.lyricsHasFocus {
			a.mediaManager.GetLyricsPanel().PageUp(10)
		}
	case "pgdn":
		if a.lyricsHasFocus {
			a.mediaManager.GetLyricsPanel().PageDown(10)
		}
	case "enter", "e":
		if !a.lyricsHasFocus {
			a.tagEditor.StartEditing(a.tagEditor.GetEditingField())
			a.currentMode = FieldEditMode
		}
	case "s":
		if a.currentFile != nil {
			return a.saveTags()
		}
	case "ctrl+s":
		if a.currentFile != nil {
			return a.saveTags()
		}
	case "ctrl+f":
		if a.currentFile != nil {
			return a.fetchBoth()
		}
	case "ctrl+l":
		if a.currentFile != nil {
			return a.fetchLyricsOnly()
		}
	case "ctrl+a":
		if a.currentFile != nil {
			return a.fetchArtworkOnly()
		}
	case "f":
		if a.currentFile != nil {
			return a.fetchBoth()
		}
	}

	return nil
}

func (a *App) handleHelpKeys(key string) tea.Cmd {
	if key == "esc" {
		a.currentMode = a.previousMode
	}
	return nil
}

func (a *App) handleFieldEditKeys(key string) tea.Cmd {
	switch key {
	case "enter":
		a.tagEditor.StopEditing()
		a.currentMode = TagEditMode
	case "esc":
		a.tagEditor.CancelEditing()
		a.currentMode = TagEditMode
	default:
		buffer := a.tagEditor.GetEditBuffer()
		switch key {
		case "backspace":
			if len(buffer) > 0 {
				a.tagEditor.UpdateEditBuffer(buffer[:len(buffer)-1])
			}
		case "ctrl+u":
			a.tagEditor.UpdateEditBuffer("")
		default:
			if len(key) == 1 {
				a.tagEditor.UpdateEditBuffer(buffer + key)
			}
		}
	}

	return nil
}

func (a *App) handleSearchKeys(key string) tea.Cmd {
	switch key {
	case "enter":
		a.currentMode = FileBrowserMode
	case "esc":
		a.fileBrowser.ClearSearch()
		a.currentMode = FileBrowserMode
	case "backspace":
		query := a.fileBrowser.GetSearchQuery()
		if len(query) > 0 {
			a.fileBrowser.SetSearch(query[:len(query)-1])
		}
	case "ctrl+u":
		a.fileBrowser.SetSearch("")
	default:
		if len(key) == 1 {
			query := a.fileBrowser.GetSearchQuery()
			a.fileBrowser.SetSearch(query + key)
		}
	}

	return nil
}

func (a *App) handleLyricsEditKeys(key string) tea.Cmd {
	switch key {
	case "esc":
		a.mediaManager.GetLyricsPanel().CancelEditing()
		a.currentMode = TagEditMode
	case "ctrl+s":
		a.mediaManager.GetLyricsPanel().StopEditing()
		a.currentMode = TagEditMode
		return a.setStatus("Lyrics updated", 2)
	default:
		// text input for lyrics???
	}

	return nil
}

func (a *App) toggleHelp() tea.Cmd {
	if a.currentMode == HelpMode {
		a.currentMode = a.previousMode
	} else {
		a.previousMode = a.currentMode
		a.currentMode = HelpMode
		clearKittyImages()
	}
	return nil
}

func (a *App) handleEscape() tea.Cmd {
	if a.statusMessage != "" && strings.HasPrefix(a.statusMessage, IconCross) {
		a.statusMessage = ""
		return nil
	}

	if a.currentMode == HelpMode {
		a.currentMode = a.previousMode
		clearKittyImages()
		if a.currentFile != nil && a.mediaManager.HasArtwork() {
			layout := a.layout.Calculate()
			xPos := layout.LeftPanelWidth
			yPos := 4
			return a.mediaManager.RenderArtworkWithSizeAndPosition(
				layout.ArtworkMaxWidth,
				layout.ArtworkMaxHeight,
				xPos,
				yPos,
			)
		}
		return nil
	}

	if a.currentMode == FileBrowserMode && a.fileBrowser.IsBatchMode() {
		a.fileBrowser.ToggleBatchMode()
		return a.setStatus("Batch mode: OFF", 2)
	}

	return nil
}
