package tui

type Layout struct {
	WindowWidth  int
	WindowHeight int
	Breakpoints  LayoutBreakpoints
}

type LayoutBreakpoints struct {
	MinWidth       int
	MinHeight      int
	StandardWidth  int
	StandardHeight int
}

type AdaptiveLayout struct {
	LeftPanelWidth   int
	MiddlePanelWidth int
	RightPanelWidth  int
	ContentHeight    int
	ArtworkMaxWidth  int
	ArtworkMaxHeight int
	LyricsWidth      int
	LyricsHeight     int
	TagsPanelHeight  int
	ShowArtwork      bool
}

func NewLayout() *Layout {
	return &Layout{
		Breakpoints: LayoutBreakpoints{
			MinWidth:       60,
			MinHeight:      20,
			StandardWidth:  120,
			StandardHeight: 35,
		},
	}
}

func (l *Layout) Update(width, height int) {
	l.WindowWidth = width
	l.WindowHeight = height
}

func (l *Layout) Calculate() AdaptiveLayout {
	leftPanelWidth := min(max(l.WindowWidth/3, 25), 45)

	artworkPanelWidth := 0
	artworkMaxWidth := 0
	artworkMaxHeight := 0

	if l.WindowWidth >= 100 {
		artworkPanelWidth = min(50, l.WindowWidth/3)
		artworkMaxWidth = artworkPanelWidth - 6
		artworkMaxHeight = l.WindowHeight - 6

		if artworkMaxWidth < 20 || artworkMaxHeight < 10 {
			artworkPanelWidth = 0
			artworkMaxWidth = 0
			artworkMaxHeight = 0
		}
	}

	middlePanelWidth := l.WindowWidth - leftPanelWidth - artworkPanelWidth - 2
	if middlePanelWidth < 30 {
		artworkPanelWidth = 0
		artworkMaxWidth = 0
		artworkMaxHeight = 0
		middlePanelWidth = l.WindowWidth - leftPanelWidth - 2
	}

	contentHeight := l.WindowHeight - 3

	tagsPanelHeight := min(12, contentHeight/2)
	lyricsHeight := contentHeight - tagsPanelHeight

	lyricsWidth := middlePanelWidth - 6

	return AdaptiveLayout{
		LeftPanelWidth:   leftPanelWidth,
		MiddlePanelWidth: middlePanelWidth,
		RightPanelWidth:  artworkPanelWidth,
		ContentHeight:    contentHeight,
		ArtworkMaxWidth:  artworkMaxWidth,
		ArtworkMaxHeight: artworkMaxHeight,
		LyricsWidth:      lyricsWidth,
		LyricsHeight:     lyricsHeight,
		TagsPanelHeight:  tagsPanelHeight,
		ShowArtwork:      artworkMaxWidth > 0 && artworkMaxHeight > 0,
	}
}

func (l *Layout) IsMinimumSize() bool {
	return l.WindowWidth >= l.Breakpoints.MinWidth &&
		l.WindowHeight >= l.Breakpoints.MinHeight
}
