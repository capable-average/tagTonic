package tui

import "tagTonic/mp3"

type Mode int

const (
	FileBrowserMode Mode = iota
	TagEditMode
	FieldEditMode
	LyricsEditMode
	SearchMode
	HelpMode
	BatchTagEditMode
	BatchFieldEditMode
)

type FileLoadedMsg struct {
	FilePath string
	Tags     *mp3.MP3Tags
}

type FileLoadErrorMsg struct {
	FilePath string
	Error    error
}

type ArtworkFetchedMsg struct {
	Artwork []byte
	Error   error
}

type LyricsFetchedMsg struct {
	Lyrics string
	Error  error
}

type BatchProcessMsg struct {
	FilePath string
	Success  bool
	Error    error
}

type BatchCompleteMsg struct {
	Total     int
	Succeeded int
	Failed    int
}

type BatchTagAppliedMsg struct {
	FilePath string
	Success  bool
	Error    error
}

type StatusTickMsg struct{}
