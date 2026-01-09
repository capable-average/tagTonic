package tui

import (
	"fmt"
	"strconv"
	"tagTonic/mp3"
)

type BulkTagEditor struct {
	editingField   int
	editBuffer     string
	isEditing      bool
	validationErrs map[string]string

	// Track which fields are enabled for bulk editing
	enabledFields map[int]bool

	// Values to apply
	title  string
	artist string
	album  string
	year   int
	genre  string
}

func NewBulkTagEditor() *BulkTagEditor {
	return &BulkTagEditor{
		validationErrs: make(map[string]string),
		enabledFields:  make(map[int]bool),
	}
}

func (bte *BulkTagEditor) Reset() {
	bte.editingField = 0
	bte.editBuffer = ""
	bte.isEditing = false
	bte.validationErrs = make(map[string]string)
	bte.enabledFields = make(map[int]bool)
	bte.title = ""
	bte.artist = ""
	bte.album = ""
	bte.year = 0
	bte.genre = ""
}

func (bte *BulkTagEditor) SetInitialValues(title, artist, album, year, genre string) {
	bte.title = title
	bte.artist = artist
	bte.album = album
	bte.genre = genre

	// Parse year - handle "<multiple values>" and empty cases
	if year != "" && year != "<multiple values>" {
		if y, err := strconv.Atoi(year); err == nil {
			bte.year = y
		}
	}
}

func (bte *BulkTagEditor) GetFields() []TagField {
	return []TagField{
		{
			Name:      "Title",
			Value:     bte.title,
			Editable:  true,
			Validator: bte.validateTitle,
		},
		{
			Name:      "Artist",
			Value:     bte.artist,
			Editable:  true,
			Validator: bte.validateArtist,
		},
		{
			Name:      "Album",
			Value:     bte.album,
			Editable:  true,
			Validator: bte.validateAlbum,
		},
		{
			Name:      "Year",
			Value:     bte.formatYear(bte.year),
			Editable:  true,
			Validator: bte.validateYear,
		},
		{
			Name:      "Genre",
			Value:     bte.genre,
			Editable:  true,
			Validator: bte.validateGenre,
		},
	}
}

func (bte *BulkTagEditor) ToggleFieldEnabled(fieldIndex int) {
	if fieldIndex < 0 || fieldIndex >= FieldCount {
		return
	}

	bte.enabledFields[fieldIndex] = !bte.enabledFields[fieldIndex]
}

func (bte *BulkTagEditor) IsFieldEnabled(fieldIndex int) bool {
	return bte.enabledFields[fieldIndex]
}

func (bte *BulkTagEditor) StartEditing(fieldIndex int) {
	if fieldIndex < 0 || fieldIndex >= FieldCount {
		return
	}

	bte.editingField = fieldIndex
	bte.isEditing = true

	fields := bte.GetFields()
	if fieldIndex < len(fields) {
		bte.editBuffer = fields[fieldIndex].Value
	}
}

func (bte *BulkTagEditor) StopEditing() {
	if !bte.isEditing {
		return
	}

	fields := bte.GetFields()
	if bte.editingField < len(fields) {
		field := fields[bte.editingField]
		if field.Validator != nil {
			if err := field.Validator(bte.editBuffer); err != nil {
				bte.validationErrs[field.Name] = err.Error()
				return
			}
		}

		delete(bte.validationErrs, field.Name)
		bte.saveFieldValue(bte.editingField, bte.editBuffer)
	}

	bte.isEditing = false
	bte.editBuffer = ""
}

func (bte *BulkTagEditor) CancelEditing() {
	bte.isEditing = false
	bte.editBuffer = ""
}

func (bte *BulkTagEditor) UpdateEditBuffer(value string) {
	bte.editBuffer = value
}

func (bte *BulkTagEditor) saveFieldValue(fieldIndex int, value string) {
	switch fieldIndex {
	case FieldTitle:
		bte.title = value
	case FieldArtist:
		bte.artist = value
	case FieldAlbum:
		bte.album = value
	case FieldYear:
		if year, err := strconv.Atoi(value); err == nil {
			bte.year = year
		} else {
			bte.year = 0
		}
	case FieldGenre:
		bte.genre = value
	case FieldLyrics:
		// Lyrics not supported in bulk edit
	}
}

func (bte *BulkTagEditor) MoveToPreviousField() {
	bte.editingField--
	if bte.editingField < 0 {
		bte.editingField = FieldCount - 1
	}
}

func (bte *BulkTagEditor) MoveToNextField() {
	bte.editingField++
	if bte.editingField >= FieldCount {
		bte.editingField = 0
	}
}

func (bte *BulkTagEditor) GetEditingField() int {
	return bte.editingField
}

func (bte *BulkTagEditor) IsEditing() bool {
	return bte.isEditing
}

func (bte *BulkTagEditor) GetEditBuffer() string {
	return bte.editBuffer
}

func (bte *BulkTagEditor) GetValidationError(fieldName string) string {
	return bte.validationErrs[fieldName]
}

func (bte *BulkTagEditor) HasValidationErrors() bool {
	return len(bte.validationErrs) > 0
}

func (bte *BulkTagEditor) HasEnabledFields() bool {
	for _, enabled := range bte.enabledFields {
		if enabled {
			return true
		}
	}
	return false
}

func (bte *BulkTagEditor) GetUpdates() mp3.TagUpdates {
	updates := mp3.TagUpdates{}

	// Only include fields that are enabled AND have been changed (not "<multiple values>")
	if bte.enabledFields[FieldTitle] && bte.title != "" && bte.title != "<multiple values>" {
		updates.Title = bte.title
	}
	if bte.enabledFields[FieldArtist] && bte.artist != "" && bte.artist != "<multiple values>" {
		updates.Artist = bte.artist
	}
	if bte.enabledFields[FieldAlbum] && bte.album != "" && bte.album != "<multiple values>" {
		updates.Album = bte.album
	}
	if bte.enabledFields[FieldYear] && bte.year != 0 {
		updates.Year = bte.year
	}
	if bte.enabledFields[FieldGenre] && bte.genre != "" && bte.genre != "<multiple values>" {
		updates.Genre = bte.genre
	}

	return updates
}

func (bte *BulkTagEditor) validateTitle(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("title too long (max 200 characters)")
	}
	return nil
}

func (bte *BulkTagEditor) validateArtist(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("artist too long (max 200 characters)")
	}
	return nil
}

func (bte *BulkTagEditor) validateAlbum(value string) error {
	if len(value) > 200 {
		return fmt.Errorf("album too long (max 200 characters)")
	}
	return nil
}

func (bte *BulkTagEditor) validateYear(value string) error {
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

func (bte *BulkTagEditor) validateGenre(value string) error {
	if len(value) > 100 {
		return fmt.Errorf("genre too long (max 100 characters)")
	}
	return nil
}

func (bte *BulkTagEditor) formatYear(year int) string {
	if year == 0 {
		return ""
	}
	return strconv.Itoa(year)
}
