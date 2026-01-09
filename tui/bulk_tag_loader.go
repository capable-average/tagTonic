package tui

import (
	"tagTonic/mp3"
)

// LoadCommonTags reads all selected files and populates the bulk editor
// with common values, or "<multiple values>" if values differ
func (a *App) loadCommonTagsForBulk() {
	selectedFiles := a.fileBrowser.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		return
	}

	te := mp3.NewTagEditor()

	// Read tags from all files
	var allTags []*mp3.MP3Tags
	for _, filePath := range selectedFiles {
		tags, err := te.ReadTags(filePath)
		if err != nil {
			continue
		}
		allTags = append(allTags, tags)
	}

	if len(allTags) == 0 {
		return
	}

	// Find common values
	title := getCommonValue(allTags, func(t *mp3.MP3Tags) string { return t.Title })
	artist := getCommonValue(allTags, func(t *mp3.MP3Tags) string { return t.Artist })
	album := getCommonValue(allTags, func(t *mp3.MP3Tags) string { return t.Album })
	genre := getCommonValue(allTags, func(t *mp3.MP3Tags) string { return t.Genre })

	// Year is special - convert to string
	var yearStrs []string
	for _, tags := range allTags {
		if tags.Year == 0 {
			yearStrs = append(yearStrs, "")
		} else {
			yearStrs = append(yearStrs, a.bulkTagEditor.formatYear(tags.Year))
		}
	}
	year := getCommonStringValue(yearStrs)

	// Set values in bulk editor
	a.bulkTagEditor.SetInitialValues(title, artist, album, year, genre)
}

// getCommonValue returns the common value if all tags have the same value,
// otherwise returns "<multiple values>"
func getCommonValue(tags []*mp3.MP3Tags, getter func(*mp3.MP3Tags) string) string {
	if len(tags) == 0 {
		return ""
	}

	firstValue := getter(tags[0])
	for i := 1; i < len(tags); i++ {
		if getter(tags[i]) != firstValue {
			return "<multiple values>"
		}
	}
	return firstValue
}

// getCommonStringValue returns the common value if all strings are the same,
// otherwise returns "<multiple values>"
func getCommonStringValue(values []string) string {
	if len(values) == 0 {
		return ""
	}

	firstValue := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] != firstValue {
			return "<multiple values>"
		}
	}
	return firstValue
}
