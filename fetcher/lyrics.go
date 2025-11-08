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

type LyricsFetcher interface {
	Fetch(title, artist string) (string, error)
}

type lyricsFetcher struct {
	client *http.Client
}

func NewLyricsFetcher() LyricsFetcher {
	return &lyricsFetcher{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (lf *lyricsFetcher) Fetch(title, artist string) (string, error) {
	if title == "" || artist == "" {
		return "", fmt.Errorf("title and artist are required")
	}

	searchVariants := lf.generateSearchVariants(title, artist)
	
	sources := []func(string, string) (string, error){
		lf.fetchFromGenius,
		lf.fetchFromAZLyrics,
		lf.fetchFromLyricsOvh,
		lf.fetchFromChartLyrics,
	}

	if len(searchVariants) > 0 {
		variant := searchVariants[0]
		if lyrics := lf.fetchConcurrently(variant.title, variant.artist, sources); lyrics != "" {
			return lyrics, nil
		}
	}

	for i := 1; i < len(searchVariants); i++ {
		variant := searchVariants[i]
		for _, source := range sources {
			if lyrics, err := source(variant.title, variant.artist); err == nil && lyrics != "" {
				return lyrics, nil
			}
		}
	}

	return "", fmt.Errorf("failed to fetch lyrics from all sources with all search variants")
}

func (lf *lyricsFetcher) fetchConcurrently(title, artist string, sources []func(string, string) (string, error)) string {
	type result struct {
		lyrics string
		err    error
	}

	results := make(chan result, len(sources))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, source := range sources {
		go func(src func(string, string) (string, error)) {
			lyrics, err := src(title, artist)
			select {
			case results <- result{lyrics: lyrics, err: err}:
			case <-ctx.Done():
			}
		}(source)
	}

	for i := 0; i < len(sources); i++ {
		select {
		case res := <-results:
			if res.err == nil && res.lyrics != "" {
				return res.lyrics
			}
		case <-ctx.Done():
			return ""
		}
	}

	return ""
}

type searchVariant struct {
	title  string
	artist string
}

func (lf *lyricsFetcher) generateSearchVariants(title, artist string) []searchVariant {
	variants := []searchVariant{}
	
	variants = append(variants, searchVariant{title, artist})
	
	normalizedTitle := lf.normalizeForSearch(title)
	normalizedArtist := lf.normalizeForSearch(artist)
	if normalizedTitle != title || normalizedArtist != artist {
		variants = append(variants, searchVariant{normalizedTitle, normalizedArtist})
	}
	
	cleanTitle := lf.removeFeaturing(title)
	cleanArtist := lf.getMainArtist(artist)
	if cleanTitle != title || cleanArtist != artist {
		variants = append(variants, searchVariant{cleanTitle, cleanArtist})
	}
	
	if cleanTitle != normalizedTitle || cleanArtist != normalizedArtist {
		variants = append(variants, searchVariant{
			lf.normalizeForSearch(cleanTitle),
			lf.normalizeForSearch(cleanArtist),
		})
	}
	
	variants = append(variants, searchVariant{artist, title})
	
	return variants
}

func (lf *lyricsFetcher) normalizeForSearch(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	
	s = regexp.MustCompile(`['"''""„"«»]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[.!?]$`).ReplaceAllString(s, "")
	
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	
	return strings.TrimSpace(s)
}

func (lf *lyricsFetcher) removeFeaturing(title string) string {
	patterns := []string{
		`\s*[\[(].*?(feat|ft|featuring).*?[\])]`,
		`\s*[-–—]\s*(feat|ft|featuring).*$`,
		`\s+(feat|ft|featuring)\.?\s+.*$`,
	}
	
	result := title
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		result = re.ReplaceAllString(result, "")
	}
	
	return strings.TrimSpace(result)
}

func (lf *lyricsFetcher) getMainArtist(artist string) string {
	patterns := []string{
		`\s*[\[(].*?(feat|ft|featuring).*?[\])]`,
		`\s*[,&]\s*(feat|ft|featuring).*$`,
		`\s+(feat|ft|featuring)\.?\s+.*$`,
	}
	
	result := artist
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		result = re.ReplaceAllString(result, "")
	}
	
	if strings.Contains(result, " & ") {
		parts := strings.Split(result, " & ")
		result = strings.TrimSpace(parts[0])
	}
	
	return strings.TrimSpace(result)
}

func (lf *lyricsFetcher) fetchFromGenius(title, artist string) (string, error) {
	searchURL := fmt.Sprintf("https://api.genius.com/search?q=%s", 
		url.QueryEscape(fmt.Sprintf("%s %s", artist, title)))
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	
	resp, err := lf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("genius search failed with status: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var searchResp struct {
		Response struct {
			Hits []struct {
				Result struct {
					ID                int    `json:"id"`
					Title             string `json:"title"`
					PrimaryArtist     struct {
						Name string `json:"name"`
					} `json:"primary_artist"`
					URL               string `json:"url"`
				} `json:"result"`
			} `json:"hits"`
		} `json:"response"`
	}
	
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse genius search response: %w", err)
	}
	
	if len(searchResp.Response.Hits) == 0 {
		return "", fmt.Errorf("no results found on genius")
	}
	
	bestMatch := lf.findBestGeniusMatch(title, artist, searchResp.Response.Hits)
	if bestMatch == nil {
		return "", fmt.Errorf("no suitable match found on genius")
	}
	
	return lf.scrapeGeniusLyrics(bestMatch.Result.URL)
}

func (lf *lyricsFetcher) findBestGeniusMatch(targetTitle, targetArtist string, hits []struct {
	Result struct {
		ID                int    `json:"id"`
		Title             string `json:"title"`
		PrimaryArtist     struct {
			Name string `json:"name"`
		} `json:"primary_artist"`
		URL               string `json:"url"`
	} `json:"result"`
}) *struct {
	Result struct {
		ID                int    `json:"id"`
		Title             string `json:"title"`
		PrimaryArtist     struct {
			Name string `json:"name"`
		} `json:"primary_artist"`
		URL               string `json:"url"`
	} `json:"result"`
} {
	if len(hits) == 0 {
		return nil
	}
	
	targetTitleNorm := lf.normalizeForSearch(targetTitle)
	targetArtistNorm := lf.normalizeForSearch(targetArtist)
	
	bestScore := 0.0
	var bestMatch *struct {
		Result struct {
			ID                int    `json:"id"`
			Title             string `json:"title"`
			PrimaryArtist     struct {
				Name string `json:"name"`
			} `json:"primary_artist"`
			URL               string `json:"url"`
		} `json:"result"`
	}
	
	for i := range hits {
		hit := &hits[i]
		hitTitleNorm := lf.normalizeForSearch(hit.Result.Title)
		hitArtistNorm := lf.normalizeForSearch(hit.Result.PrimaryArtist.Name)
		
		titleScore := lf.calculateSimilarity(targetTitleNorm, hitTitleNorm)
		artistScore := lf.calculateSimilarity(targetArtistNorm, hitArtistNorm)
		
		score := titleScore*0.7 + artistScore*0.3
		
		if score > bestScore {
			bestScore = score
			bestMatch = hit
		}
	}
	
	if bestScore > 0.4 {
		return bestMatch
	}
	
	return nil
}

func (lf *lyricsFetcher) scrapeGeniusLyrics(geniusURL string) (string, error) {
	req, err := http.NewRequest("GET", geniusURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	
	resp, err := lf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch genius page: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	html := string(body)
	
	patterns := []string{
		`<div[^>]*data-lyrics-container="true"[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*lyrics[^"]*"[^>]*>(.*?)</div>`,
		`<div[^>]*id="lyrics-root"[^>]*>(.*?)</div>`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?s)` + pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		
		if len(matches) > 0 {
			lyrics := ""
			for _, match := range matches {
				if len(match) > 1 {
					lyrics += match[1] + "\n"
				}
			}
			
			if lyrics != "" {
				lyrics = lf.cleanHTMLLyrics(lyrics)
				if len(strings.TrimSpace(lyrics)) > 50 {
					return lyrics, nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("could not extract lyrics from genius page")
}

func (lf *lyricsFetcher) cleanHTMLLyrics(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")
	
	replacements := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#x27;":   "'",
		"&#39;":    "'",
		"&nbsp;":   " ",
		"<br>":     "\n",
		"<br/>":    "\n",
		"<br />":   "\n",
	}
	
	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}
	
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	return strings.Join(cleanLines, "\n")
}

func (lf *lyricsFetcher) fetchFromAZLyrics(title, artist string) (string, error) {
	artistSlug := lf.createAZLyricsSlug(artist)
	titleSlug := lf.createAZLyricsSlug(title)
	
	azURL := fmt.Sprintf("https://www.azlyrics.com/lyrics/%s/%s.html", artistSlug, titleSlug)
	
	req, err := http.NewRequest("GET", azURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	
	resp, err := lf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("azlyrics returned status: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	html := string(body)
	
	startMarker := `<!-- Usage of azlyrics.com content -->`
	
	startIdx := strings.Index(html, startMarker)
	if startIdx == -1 {
		return "", fmt.Errorf("could not find lyrics start marker")
	}
	
	searchStart := startIdx + len(startMarker)
	divStart := strings.Index(html[searchStart:], "<div>")
	if divStart == -1 {
		return "", fmt.Errorf("could not find lyrics div")
	}
	
	lyricsStart := searchStart + divStart + 5
	
	divEnd := strings.Index(html[lyricsStart:], "</div>")
	if divEnd == -1 {
		return "", fmt.Errorf("could not find lyrics end")
	}
	
	lyricsHTML := html[lyricsStart : lyricsStart+divEnd]
	
	lyrics := lf.cleanHTMLLyrics(lyricsHTML)
	
	if len(strings.TrimSpace(lyrics)) < 20 {
		return "", fmt.Errorf("lyrics too short, probably not found")
	}
	
	return lyrics, nil
}

func (lf *lyricsFetcher) createAZLyricsSlug(s string) string {
	s = strings.ToLower(s)
	s=strings.TrimPrefix(s, "the ")
	
	re := regexp.MustCompile(`[^a-z0-9]`)
	s = re.ReplaceAllString(s, "")
	
	return s
}

func (lf *lyricsFetcher) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)
	
	if strings.Contains(s1Lower, s2Lower) || strings.Contains(s2Lower, s1Lower) {
		shorter := len(s1)
		longer := len(s2)
		if len(s2) < len(s1) {
			shorter = len(s2)
			longer = len(s1)
		}
		return float64(shorter) / float64(longer)
	}
	
	words1 := strings.Fields(s1Lower)
	words2 := strings.Fields(s2Lower)
	
	commonWords := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				commonWords++
				break
			}
		}
	}
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	return float64(commonWords) / float64(max(len(words1), len(words2)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (lf *lyricsFetcher) fetchFromLyricsOvh(title, artist string) (string, error) {
	cleanTitle := lf.normalizeForSearch(title)
	cleanArtist := lf.normalizeForSearch(artist)

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
	t := lf.normalizeForSearch(title)
	a := lf.normalizeForSearch(artist)
	if t == "" || a == "" {
		return "", fmt.Errorf("missing terms")
	}

	searchURL := fmt.Sprintf("http://api.chartlyrics.com/apiv1.asmx/SearchLyricDirect?artist=%s&song=%s", 
		url.QueryEscape(a), url.QueryEscape(t))
	
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