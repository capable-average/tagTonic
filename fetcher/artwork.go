package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ArtworkFetcher interface {
	Fetch(title, artist, album string) ([]byte, error)
}

type artworkFetcher struct {
	client *http.Client
}

func NewArtworkFetcher() ArtworkFetcher {
	return &artworkFetcher{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (af *artworkFetcher) Fetch(title, artist, album string) ([]byte, error) {
	if title == "" && artist == "" && album == "" {
		return nil, fmt.Errorf("at least one of title, artist, or album is required")
	}

	sources := []func(string, string, string) ([]byte, error){
		af.fetchFromDeezer,
		af.fetchFromITunes,
		af.fetchFromMusicBrainz,
	}

	if artwork := af.fetchConcurrently(title, artist, album, sources); len(artwork) > 0 {
		return artwork, nil
	}

	return nil, fmt.Errorf("failed to fetch artwork from free sources")
}

func (af *artworkFetcher) fetchConcurrently(title, artist, album string, sources []func(string, string, string) ([]byte, error)) []byte {
	type result struct {
		artwork []byte
		err     error
	}

	results := make(chan result, len(sources))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, source := range sources {
		go func(src func(string, string, string) ([]byte, error)) {
			artwork, err := src(title, artist, album)
			select {
			case results <- result{artwork: artwork, err: err}:
			case <-ctx.Done():
			}
		}(source)
	}

	for i := 0; i < len(sources); i++ {
		select {
		case res := <-results:
			if res.err == nil && len(res.artwork) > 0 {
				return res.artwork
			}
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

func (af *artworkFetcher) fetchFromMusicBrainz(title, artist, album string) ([]byte, error) {
	searchRelease := strings.TrimSpace(album)
	if searchRelease == "" {
		searchRelease = strings.TrimSpace(title)
	}
	if searchRelease == "" {
		return nil, fmt.Errorf("no release term")
	}

	queryParts := []string{}
	if artist != "" {
		queryParts = append(queryParts, fmt.Sprintf("artist:\"%s\"", sanitizeMBQuery(artist)))
	}
	if searchRelease != "" {
		queryParts = append(queryParts, fmt.Sprintf("release:\"%s\"", sanitizeMBQuery(searchRelease)))
	}
	query := strings.Join(queryParts, " AND ")
	mbURL := fmt.Sprintf("https://musicbrainz.org/ws/2/release/?query=%s&fmt=json&limit=1", url.QueryEscape(query))

	req, err := http.NewRequest("GET", mbURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tagTonic/1.0 (https://github.com/sumit_pathak/tagTonic)")

	resp, err := af.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var mb struct {
		Releases []struct {
			ID string `json:"id"`
		} `json:"releases"`
	}
	if err := json.Unmarshal(body, &mb); err != nil {
		return nil, err
	}
	if len(mb.Releases) == 0 || mb.Releases[0].ID == "" {
		return nil, fmt.Errorf("no release match")
	}
	releaseID := mb.Releases[0].ID

	attempts := []string{
		fmt.Sprintf("https://coverartarchive.org/release/%s/front-500", releaseID),
		fmt.Sprintf("https://coverartarchive.org/release/%s/front", releaseID),
	}
	for _, u := range attempts {
		imgReq, _ := http.NewRequest("GET", u, nil)
		imgReq.Header.Set("User-Agent", "tagTonic/1.0 (https://github.com/sumit_pathak/tagTonic)")
		imgResp, err := af.client.Do(imgReq)
		if err != nil {
			continue
		}
		if imgResp.StatusCode == http.StatusOK {
			data, _ := io.ReadAll(imgResp.Body)
			imgResp.Body.Close()
			if len(data) > 0 {
				return data, nil
			}
		}
		imgResp.Body.Close()
	}
	return nil, fmt.Errorf("no cover art via musicbrainz")
}

func sanitizeMBQuery(s string) string {
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.TrimSpace(s)
	return s
}

func (af *artworkFetcher) fetchFromITunes(title, artist, album string) ([]byte, error) {
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		re := regexp.MustCompile(`(?i)\s*[\[(].*?[\])]`)
		s = re.ReplaceAllString(s, "")
		s = strings.Join(strings.Fields(s), " ")
		return s
	}
	searchTerm := fmt.Sprintf("%s %s", clean(artist), clean(album))
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		searchTerm = clean(title)
	}

	searchURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=album&limit=1",
		url.QueryEscape(searchTerm))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := af.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iTunes API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		ResultCount int `json:"resultCount"`
		Results     []struct {
			ArtworkURL100 string `json:"artworkUrl100"`
			ArtworkURL60  string `json:"artworkUrl60"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse iTunes response: %w", err)
	}

	if response.ResultCount == 0 || len(response.Results) == 0 {
		return nil, fmt.Errorf("no results found in iTunes")
	}

	result := response.Results[0]

	artworkURL := result.ArtworkURL100
	if artworkURL == "" {
		artworkURL = result.ArtworkURL60
	}

	if artworkURL == "" {
		return nil, fmt.Errorf("no artwork URL found in iTunes result")
	}

	largerURL := strings.Replace(artworkURL, "100x100", "500x500", 1)
	if largerURL == artworkURL {
		largerURL = strings.Replace(artworkURL, "60x60", "500x500", 1)
	}

	if largerArtwork, err := af.downloadImage(largerURL); err == nil && len(largerArtwork) > 0 {
		return largerArtwork, nil
	}

	return af.downloadImage(artworkURL)
}

func (af *artworkFetcher) downloadImage(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := af.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (af *artworkFetcher) fetchFromDeezer(title, artist, album string) ([]byte, error) {
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		re := regexp.MustCompile(`(?i)\s*[\[(].*?[\])]`)
		s = re.ReplaceAllString(s, "")
		s = strings.Join(strings.Fields(s), " ")
		return s
	}
	searchTerm := fmt.Sprintf("%s %s", clean(artist), clean(album))
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		searchTerm = clean(title)
	}

	searchURL := fmt.Sprintf("https://api.deezer.com/search?q=%s",
		url.QueryEscape(searchTerm))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := af.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deezer API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []struct {
			Album struct {
				Cover       string `json:"cover"`
				CoverMedium string `json:"cover_medium"`
				CoverBig    string `json:"cover_big"`
				CoverXL     string `json:"cover_xl"`
			} `json:"album"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse deezer response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no results found in Deezer")
	}

	result := response.Data[0]
	coverURL := result.Album.CoverXL
	if coverURL == "" {
		coverURL = result.Album.CoverBig
	}
	if coverURL == "" {
		coverURL = result.Album.CoverMedium
	}
	if coverURL == "" {
		coverURL = result.Album.Cover
	}

	if coverURL == "" {
		return nil, fmt.Errorf("no artwork URL found in Deezer result")
	}

	return af.downloadImage(coverURL)
}
