package tui

import (
	"path/filepath"
	"tagTonic/fetcher"
	"tagTonic/mp3"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
)

func (a *App) processBatchLyrics(filePaths []string, index int) tea.Cmd {
	if index >= len(filePaths) {
		return nil
	}

	filePath := filePaths[index]

	return func() tea.Msg {
		te := mp3.NewTagEditor()
		lf := fetcher.NewLyricsFetcher()

		tags, err := te.ReadTags(filePath)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		lyrics, err := lf.Fetch(tags.Title, tags.Artist)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		updates := mp3.TagUpdates{
			Lyrics: lyrics,
		}
		err = te.EditTags(filePath, updates)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		return BatchProcessMsg{
			FilePath: filePath,
			Success:  true,
			Error:    nil,
		}
	}
}

func (a *App) processBatchArtwork(filePaths []string, index int) tea.Cmd {
	if index >= len(filePaths) {
		return nil
	}

	filePath := filePaths[index]

	return func() tea.Msg {
		te := mp3.NewTagEditor()
		af := fetcher.NewArtworkFetcher()

		tags, err := te.ReadTags(filePath)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		artwork, err := af.Fetch(tags.Title, tags.Artist, tags.Album)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		updates := mp3.TagUpdates{
			Artwork: artwork,
		}
		err = te.EditTags(filePath, updates)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		return BatchProcessMsg{
			FilePath: filePath,
			Success:  true,
			Error:    nil,
		}
	}
}

func (a *App) processBatchBoth(filePaths []string, index int) tea.Cmd {
	if index >= len(filePaths) {
		return nil
	}

	filePath := filePaths[index]

	return func() tea.Msg {
		te := mp3.NewTagEditor()
		lf := fetcher.NewLyricsFetcher()
		af := fetcher.NewArtworkFetcher()

		tags, err := te.ReadTags(filePath)
		if err != nil {
			return BatchProcessMsg{
				FilePath: filePath,
				Success:  false,
				Error:    err,
			}
		}

		var lyrics string
		var artwork []byte

		lyrics, err = lf.Fetch(tags.Title, tags.Artist)
		if err != nil {
			logrus.Debugf("Failed to fetch lyrics for %s: %v", filepath.Base(filePath), err)
		}

		artwork, err = af.Fetch(tags.Title, tags.Artist, tags.Album)
		if err != nil {
			logrus.Debugf("Failed to fetch artwork for %s: %v", filepath.Base(filePath), err)
		}

		if lyrics != "" || len(artwork) > 0 {
			updates := mp3.TagUpdates{
				Lyrics:  lyrics,
				Artwork: artwork,
			}
			err = te.EditTags(filePath, updates)
			if err != nil {
				return BatchProcessMsg{
					FilePath: filePath,
					Success:  false,
					Error:    err,
				}
			}
		}

		return BatchProcessMsg{
			FilePath: filePath,
			Success:  true,
			Error:    nil,
		}
	}
}
