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
		case BatchFieldEditMode:
			return a.handleBatchFieldEditKeys(key)
		case LyricsEditMode:
			return a.handleLyricsEditKeys(key)
		case BatchTagEditMode:
			return a.handleBatchTagEditKeys(key)
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
	case BatchTagEditMode:
		return a.handleBatchTagEditKeys(key)
	case BatchFieldEditMode:
		return a.handleBatchFieldEditKeys(key)
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
		// In batch mode, pressing enter should open batch tag editor (or navigate directories)
		if a.fileBrowser.IsBatchMode() {
			if err := a.fileBrowser.Navigate(); err != nil {
				a.setError("Navigation failed", err.Error())
			} else {
				// If it's a file, treat it as a batch selection and open bulk editor
				if file := a.fileBrowser.GetSelectedFile(); file != nil {
					// Auto-select the file if not already selected
					if !a.fileBrowser.IsSelected(file.Path) {
						a.fileBrowser.ToggleSelection()
					}
					// Open bulk tag editor
					a.currentMode = BatchTagEditMode
					a.bulkTagEditor.Reset()
					a.loadCommonTagsForBulk()
					return a.setStatus("Batch tag editor - edit fields with Enter, save with 's'", 3)
				}
			}
		} else {
			// Normal mode - navigate or load file
			if err := a.fileBrowser.Navigate(); err != nil {
				a.setError("Navigation failed", err.Error())
			} else {
				if file := a.fileBrowser.GetSelectedFile(); file != nil {
					return a.loadFile(file)
				}
			}
		}
	case "tab":
		if a.currentFile != nil && !a.fileBrowser.IsBatchMode() {
			a.currentMode = TagEditMode
		} else if a.fileBrowser.IsBatchMode() && len(a.fileBrowser.GetSelectedFiles()) > 0 {
			// Enter batch tag edit mode
			a.currentMode = BatchTagEditMode
			a.bulkTagEditor.Reset()
			a.loadCommonTagsForBulk()
			return a.setStatus("Batch tag editor - edit fields with Enter, save with 's'", 3)
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
		// Clear current file when entering batch mode so single-file tags panel doesn't show
		if a.fileBrowser.IsBatchMode() {
			a.currentFile = nil
		}
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
	case "u":
		if a.tagEditor.CanUndo() {
			a.tagEditor.Undo()
			return a.setStatus("Undone", 1)
		}
	case "r":
		if a.tagEditor.CanRedo() {
			a.tagEditor.Redo()
			return a.setStatus("Redone", 1)
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

func (a *App) handleBatchTagEditKeys(key string) tea.Cmd {
	switch key {
	case "up", "k":
		a.bulkTagEditor.MoveToPreviousField()
	case "down", "j":
		a.bulkTagEditor.MoveToNextField()
	case "tab":
		a.currentMode = FileBrowserMode
	case "enter", "e":
		// Start editing the current field
		fieldIndex := a.bulkTagEditor.GetEditingField()
		a.bulkTagEditor.StartEditing(fieldIndex)
		// Auto-enable the field when starting to edit
		if !a.bulkTagEditor.IsFieldEnabled(fieldIndex) {
			a.bulkTagEditor.ToggleFieldEnabled(fieldIndex)
		}
		a.currentMode = BatchFieldEditMode
	case "s", "ctrl+s":
		// Save/Apply bulk tags
		return a.applyBulkTags()
	case "f", "ctrl+f":
		// Fetch for all selected files
		return a.batchFetchBoth()
	case "ctrl+l":
		// Fetch lyrics only
		return a.batchFetchLyrics()
	case "ctrl+a":
		// Fetch artwork only
		return a.batchFetchArtwork()
	case "esc":
		a.currentMode = FileBrowserMode
		a.bulkTagEditor.Reset()
		return a.setStatus("Exited batch tag editor", 1)
	}

	return nil
}

func (a *App) handleBatchFieldEditKeys(key string) tea.Cmd {
	switch key {
	case "enter":
		a.bulkTagEditor.StopEditing()
		a.currentMode = BatchTagEditMode
	case "esc":
		a.bulkTagEditor.CancelEditing()
		a.currentMode = BatchTagEditMode
	default:
		buffer := a.bulkTagEditor.GetEditBuffer()
		switch key {
		case "backspace":
			if len(buffer) > 0 {
				a.bulkTagEditor.UpdateEditBuffer(buffer[:len(buffer)-1])
			}
		case "ctrl+u":
			a.bulkTagEditor.UpdateEditBuffer("")
		default:
			if len(key) == 1 {
				a.bulkTagEditor.UpdateEditBuffer(buffer + key)
			}
		}
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
