package mp3

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strconv"

	"github.com/bogem/id3v2"
	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
)

type TagUpdates struct {
	Title        string
	Artist       string
	Album        string
	Genre        string
	Year         int
	Lyrics       string
	Artwork      []byte
	ClearLyrics  bool
	ClearArtwork bool
}

type MP3Tags struct {
	Title   string
	Artist  string
	Album   string
	Genre   string
	Year    int
	Lyrics  string
	Artwork []byte
}

type TagEditor interface {
	ReadTags(filePath string) (*MP3Tags, error)
	EditTags(filePath string, updates TagUpdates) error
	ValidateArtwork(data []byte) error
	ResizeArtwork(data []byte, maxWidth, maxHeight int) ([]byte, error)
}

type tagEditor struct{}

func NewTagEditor() TagEditor {
	return &tagEditor{}
}

func (te *tagEditor) ReadTags(filePath string) (*MP3Tags, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	year := 0
	if yearStr := tag.Year(); yearStr != "" {
		if yearInt, err := strconv.Atoi(yearStr); err == nil {
			year = yearInt
		}
	}

	tags := &MP3Tags{
		Title:  tag.Title(),
		Artist: tag.Artist(),
		Album:  tag.Album(),
		Genre:  tag.Genre(),
		Year:   year,
	}

	if lyricsFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription")); len(lyricsFrames) > 0 {
		if lyricsFrame, ok := lyricsFrames[0].(id3v2.UnsynchronisedLyricsFrame); ok {
			tags.Lyrics = lyricsFrame.Lyrics
		}
	}

	frames := tag.GetFrames("APIC")
	for _, f := range frames {
		if pf, ok := f.(id3v2.PictureFrame); ok {
			tags.Artwork = pf.Picture
			break
		}
	}

	return tags, nil
}

func (te *tagEditor) EditTags(filePath string, updates TagUpdates) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	// Update basic tags
	if updates.Title != "" {
		tag.SetTitle(updates.Title)
	}
	if updates.Artist != "" {
		tag.SetArtist(updates.Artist)
	}
	if updates.Album != "" {
		tag.SetAlbum(updates.Album)
	}
	if updates.Genre != "" {
		tag.SetGenre(updates.Genre)
	}
	if updates.Year != 0 {
		tag.SetYear(strconv.Itoa(updates.Year))
	}

	lyricsID := tag.CommonID("Unsynchronised lyrics/text transcription")
	if updates.ClearLyrics || updates.Lyrics != "" {
		if lyricsID != "" {
			tag.DeleteFrames(lyricsID)
		}
		if updates.Lyrics != "" {
			lyricsFrame := id3v2.UnsynchronisedLyricsFrame{
				Encoding:          id3v2.EncodingUTF8,
				Language:          "eng",
				ContentDescriptor: "Lyrics",
				Lyrics:            updates.Lyrics,
			}
			tag.AddUnsynchronisedLyricsFrame(lyricsFrame)
		}
	}

	if updates.ClearArtwork || len(updates.Artwork) > 0 {
		tag.DeleteFrames("APIC")
		if len(updates.Artwork) > 0 {
			if err := te.ValidateArtwork(updates.Artwork); err != nil {
				return fmt.Errorf("invalid artwork: %w", err)
			}
			resizedArtwork, err := te.ResizeArtwork(updates.Artwork, 500, 500)
			if err != nil {
				logrus.Warnf("Failed to resize artwork: %v", err)
				resizedArtwork = updates.Artwork
			}
			pictureFrame := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    getMIMEType(resizedArtwork),
				PictureType: id3v2.PTFrontCover,
				Description: "Cover Art",
				Picture:     resizedArtwork,
			}
			tag.AddAttachedPicture(pictureFrame)
		}
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}

	return nil
}

func (te *tagEditor) ValidateArtwork(data []byte) error {
	const maxArtworkSize = 5 * 1024 * 1024 // 5MB
	if len(data) > maxArtworkSize {
		return fmt.Errorf("artwork exceeds %d bytes", maxArtworkSize)
	}
	kind, err := filetype.Match(data)
	if err != nil {
		return fmt.Errorf("failed to determine file type: %w", err)
	}

	if !filetype.IsImage(data) {
		return fmt.Errorf("file is not an image: %s", kind.Extension)
	}

	return nil
}

func (te *tagEditor) ResizeArtwork(data []byte, maxWidth, maxHeight int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= maxWidth && height <= maxHeight {
		return data, nil
	}

	ratio := float64(width) / float64(height)
	var newWidth, newHeight int

	if ratio > 1 {
		newWidth = maxWidth
		newHeight = int(float64(maxWidth) / ratio)
	} else {
		newHeight = maxHeight
		newWidth = int(float64(maxHeight) * ratio)
	}

	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * float64(width) / float64(newWidth))
			srcY := int(float64(y) * float64(height) / float64(newHeight))
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(&buf, resized)
	default:
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 90})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}

func getMIMEType(data []byte) string {
	kind, _ := filetype.Match(data)
	switch kind.Extension {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}
