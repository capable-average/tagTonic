package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type LyricsFetcher interface {
	Fetch(title, artist string) (string, error)
}

type lyricsFetcher struct {
	client *http.Client
}

func NewLyricsFetcher() LyricsFetcher {
	return &lyricsFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (lf *lyricsFetcher) Fetch(title, artist string) (string, error) {
	if title == "" || artist == "" {
		return "", fmt.Errorf("title and artist are required")
	}

	sources := []func(string, string) (string, error){
		lf.fetchFromLyricsOvh,
		lf.fetchFromChartLyrics,
	}

	for _, source := range sources {
		if lyrics, err := source(title, artist); err == nil && lyrics != "" {
			return lyrics, nil
		}
	}

	return "", fmt.Errorf("failed to fetch lyrics from all sources")
}

func (lf *lyricsFetcher) fetchFromLyricsOvh(title, artist string) (string, error) {
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		re := regexp.MustCompile(`(?i)\s*[\[(].*?[\])]`)
		s = re.ReplaceAllString(s, "")
		s = strings.Join(strings.Fields(s), " ") // collapse spaces
		return s
	}
	cleanTitle := clean(title)
	cleanArtist := clean(artist)

	searchURL := fmt.Sprintf("https://api.lyrics.ovh/v1/%s/%s",
		url.QueryEscape(cleanArtist),
		url.QueryEscape(cleanTitle))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := lf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics.ovh API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		Lyrics string `json:"lyrics"`
		Error  string `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("lyrics.ovh error: %s", response.Error)
	}

	if response.Lyrics == "" {
		return "", fmt.Errorf("no lyrics found")
	}

	return response.Lyrics, nil
}

func (lf *lyricsFetcher) fetchFromChartLyrics(title, artist string) (string, error) {
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		re := regexp.MustCompile(`(?i)\s*[\[(].*?[\])]`)
		s = re.ReplaceAllString(s, "")
		return strings.Join(strings.Fields(s), " ")
	}
	t := clean(title)
	a := clean(artist)
	if t == "" || a == "" {
		return "", fmt.Errorf("missing terms")
	}

	searchURL := fmt.Sprintf("http://api.chartlyrics.com/apiv1.asmx/SearchLyricDirect?artist=%s&song=%s", url.QueryEscape(a), url.QueryEscape(t))
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "tagTonic/1.0")
	resp, err := lf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chartlyrics status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	xml := string(data)
	start := strings.Index(strings.ToLower(xml), "<lyric>")
	end := strings.Index(strings.ToLower(xml), "</lyric>")
	if start == -1 || end == -1 || end <= start+7 {
		return "", fmt.Errorf("no lyrics tag")
	}
	lyric := xml[start+7 : end]
	lyric = strings.TrimSpace(lyric)
	if lyric == "" {
		return "", fmt.Errorf("empty lyric")
	}
	replacer := strings.NewReplacer("&quot;", "\"", "&amp;", "&")
	lyric = replacer.Replace(lyric)
	return lyric, nil
}
