package tui

import (
	"fmt"
	"strconv"
	"tagTonic/mp3"
)

type TagEditor struct {
	currentTags    *mp3.MP3Tags
	originalTags   *mp3.MP3Tags
	editingField   int
	editBuffer     string
	isEditing      bool
	isDirty        bool
	undoStack      []*mp3.MP3Tags
	redoStack      []*mp3.MP3Tags
	validationErrs map[string]string
	tagEditor      mp3.TagEditor
}

type TagField struct {
	Name      string
	Value     string
	Editable  bool
	Validator func(string) error
}

const (
	FieldTitle = iota
	FieldArtist
	FieldAlbum
	FieldYear
	FieldGenre
	FieldLyrics
	FieldCount
)

func NewTagEditor() *TagEditor {
	return &TagEditor{
		validationErrs: make(map[string]string),
		tagEditor:      mp3.NewTagEditor(),
		undoStack:      make([]*mp3.MP3Tags, 0),
		redoStack:      make([]*mp3.MP3Tags, 0),
	}
}

func (te *TagEditor) LoadTags(tags *mp3.MP3Tags) {
	te.currentTags = tags
	te.originalTags = &mp3.MP3Tags{
		Title:   tags.Title,
		Artist:  tags.Artist,
		Album:   tags.Album,
		Genre:   tags.Genre,
		Year:    tags.Year,
		Lyrics:  tags.Lyrics,
		Artwork: tags.Artwork,
	}
	te.isDirty = false
	te.validationErrs = make(map[string]string)
}

func (te *TagEditor) GetFields() []TagField {
	if te.currentTags == nil {
		return []TagField{}
	}

	return []TagField{
		{
			Name:      "Title",
			Value:     te.currentTags.Title,
			Editable:  true,
			Validator: te.validateTitle,
		},
		{
			Name:      "Artist",
			Value:     te.currentTags.Artist,
			Editable:  true,
			Validator: te.validateArtist,
		},
		{
			Name:      "Album",
			Value:     te.currentTags.Album,
			Editable:  true,
			Validator: te.validateAlbum,
		},
		{
			Name:      "Year",
			Value:     te.formatYear(te.currentTags.Year),
			Editable:  true,
			Validator: te.validateYear,
		},
		{
			Name:      "Genre",
			Value:     te.currentTags.Genre,
			Editable:  true,
			Validator: te.validateGenre,
		},
	}
}

func (te *TagEditor) StartEditing(fieldIndex int) {
	if fieldIndex < 0 || fieldIndex >= FieldCount {
		return
	}

	te.editingField = fieldIndex
	te.isEditing = true

	fields := te.GetFields()
	if fieldIndex < len(fields) {
		te.editBuffer = fields[fieldIndex].Value
	}
}

func (te *TagEditor) StopEditing() {
	if !te.isEditing {
		return
	}

	fields := te.GetFields()
	if te.editingField < len(fields) {
		field := fields[te.editingField]
		if field.Validator != nil {
			if err := field.Validator(te.editBuffer); err != nil {
				te.validationErrs[field.Name] = err.Error()
				return
			}
		}

		delete(te.validationErrs, field.Name)

		te.saveFieldValue(te.editingField, te.editBuffer)
	}

	te.isEditing = false
	te.editBuffer = ""
}

func (te *TagEditor) CancelEditing() {
	te.isEditing = false
	te.editBuffer = ""
}

func (te *TagEditor) UpdateEditBuffer(value string) {
	te.editBuffer = value
}

func (te *TagEditor) saveFieldValue(fieldIndex int, value string) {
	if te.currentTags == nil {
		return
	}

	switch fieldIndex {
	case FieldTitle:
		te.currentTags.Title = value
	case FieldArtist:
		te.currentTags.Artist = value
	case FieldAlbum:
		te.currentTags.Album = value
	case FieldYear:
		if year, err := strconv.Atoi(value); err == nil {
			te.currentTags.Year = year
		} else {
			te.currentTags.Year = 0
		}
	case FieldGenre:
		te.currentTags.Genre = value
	case FieldLyrics:
		te.currentTags.Lyrics = value
	}

	te.isDirty = true
}

func (te *TagEditor) MoveToPreviousField() {
	te.editingField--
	if te.editingField < 0 {
		te.editingField = FieldCount - 1
	}
}

func (te *TagEditor) MoveToNextField() {
	te.editingField++
	if te.editingField >= FieldCount {
		te.editingField = 0
	}
}

func (te *TagEditor) GetEditingField() int {
	return te.editingField
}

func (te *TagEditor) IsEditing() bool {
	return te.isEditing
}

func (te *TagEditor) GetEditBuffer() string {
	return te.editBuffer
}

func (te *TagEditor) IsDirty() bool {
	return te.isDirty
}

func (te *TagEditor) GetValidationError(fieldName string) string {
	return te.validationErrs[fieldName]
}

func (te *TagEditor) HasValidationErrors() bool {
	return len(te.validationErrs) > 0
}

func (te *TagEditor) SaveTags(filePath string) error {
	if te.currentTags == nil {
		return fmt.Errorf("no tags loaded")
	}

	if te.HasValidationErrors() {
		return fmt.Errorf("cannot save with validation errors")
	}

	updates := mp3.TagUpdates{
		Title:   te.currentTags.Title,
		Artist:  te.currentTags.Artist,
		Album:   te.currentTags.Album,
		Genre:   te.currentTags.Genre,
		Year:    te.currentTags.Year,
		Lyrics:  te.currentTags.Lyrics,
		Artwork: te.currentTags.Artwork,
	}

	err := te.tagEditor.EditTags(filePath, updates)
	if err != nil {
		return err
	}

	te.isDirty = false
	return nil
}

func (te *TagEditor) UpdateLyrics(lyrics string) {
	if te.currentTags == nil {
		return
	}

	te.currentTags.Lyrics = lyrics
	te.isDirty = true
}

func (te *TagEditor) UpdateArtwork(artwork []byte) {
	if te.currentTags == nil {
		return
	}

	te.currentTags.Artwork = artwork
	te.isDirty = true
}

func (te *TagEditor) validateTitle(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("title too long (max 200 characters)")
	}
	return nil
}

func (te *TagEditor) validateArtist(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("artist too long (max 200 characters)")
	}
	return nil
}

func (te *TagEditor) validateAlbum(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("album too long (max 200 characters)")
	}
	return nil
}

func (te *TagEditor) validateYear(value string) error {
	if value == "" {
		return nil // Empty year is allowed
	}

	year, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("year must be a number")
	}

	if year < 1000 || year > 9999 {
		return fmt.Errorf("year must be between 1000 and 9999")
	}

	return nil
}

func (te *TagEditor) validateGenre(value string) error {
	if len(value) > 100 {
		return fmt.Errorf("genre too long (max 100 characters)")
	}
	return nil
}

func (te *TagEditor) formatYear(year int) string {
	if year == 0 {
		return ""
	}
	return strconv.Itoa(year)
}
