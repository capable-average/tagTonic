package tui

import tea "github.com/charmbracelet/bubbletea"

func (a *App) fetchBoth() tea.Cmd {
	if a.currentFile == nil {
		return nil
	}

	fields := a.tagEditor.GetFields()
	var title, artist, album string

	for _, field := range fields {
		switch field.Name {
		case "Title":
			title = field.Value
		case "Artist":
			artist = field.Value
		case "Album":
			album = field.Value
		}
	}

	cmds := []tea.Cmd{
		a.mediaManager.FetchLyrics(title, artist),
		a.mediaManager.FetchArtwork(title, artist, album),
		a.setStatus("Fetching lyrics and artwork...", 2),
	}

	return tea.Batch(cmds...)
}

func (a *App) fetchLyricsOnly() tea.Cmd {
	if a.currentFile == nil {
		return nil
	}

	fields := a.tagEditor.GetFields()
	var title, artist string

	for _, field := range fields {
		switch field.Name {
		case "Title":
			title = field.Value
		case "Artist":
			artist = field.Value
		}
	}

	return tea.Batch(
		a.mediaManager.FetchLyrics(title, artist),
		a.setStatus("Fetching lyrics...", 2),
	)
}

func (a *App) fetchArtworkOnly() tea.Cmd {
	if a.currentFile == nil {
		return nil
	}

	fields := a.tagEditor.GetFields()
	var title, artist, album string

	for _, field := range fields {
		switch field.Name {
		case "Title":
			title = field.Value
		case "Artist":
			artist = field.Value
		case "Album":
			album = field.Value
		}
	}

	return tea.Batch(
		a.mediaManager.FetchArtwork(title, artist, album),
		a.setStatus("Fetching artwork...", 2),
	)
}

func (a *App) batchFetchLyrics() tea.Cmd {
	selectedFiles := a.fileBrowser.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		return a.setStatus("No files selected", 2)
	}

	a.isBatchProcessing = true
	a.batchTotal = len(selectedFiles)
	a.batchProcessed = 0
	a.batchSucceeded = 0
	a.batchFailed = 0
	a.batchFilePaths = selectedFiles
	a.batchMode = "lyrics"

	return a.processBatchLyrics(selectedFiles, 0)
}

func (a *App) batchFetchArtwork() tea.Cmd {
	selectedFiles := a.fileBrowser.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		return a.setStatus("No files selected", 2)
	}

	a.isBatchProcessing = true
	a.batchTotal = len(selectedFiles)
	a.batchProcessed = 0
	a.batchSucceeded = 0
	a.batchFailed = 0
	a.batchFilePaths = selectedFiles
	a.batchMode = "artwork"

	return a.processBatchArtwork(selectedFiles, 0)
}

func (a *App) batchFetchBoth() tea.Cmd {
	selectedFiles := a.fileBrowser.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		return a.setStatus("No files selected", 2)
	}

	a.isBatchProcessing = true
	a.batchTotal = len(selectedFiles)
	a.batchProcessed = 0
	a.batchSucceeded = 0
	a.batchFailed = 0
	a.batchFilePaths = selectedFiles
	a.batchMode = "both"

	return a.processBatchBoth(selectedFiles, 0)
}
