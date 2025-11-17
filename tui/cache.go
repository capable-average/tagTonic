package tui

type Cache struct {
	artworkCache map[string][]byte
}

func NewCache(maxSizeMB int) *Cache {
	return &Cache{
		artworkCache: make(map[string][]byte),
	}
}

func (c *Cache) GetArtwork(filePath string) ([]byte, bool) {
	data, exists := c.artworkCache[filePath]
	return data, exists
}

func (c *Cache) SetArtwork(filePath string, data []byte) {
	if len(data) == 0 {
		return
	}
	c.artworkCache[filePath] = data
}

func (c *Cache) Clear() {
	c.artworkCache = make(map[string][]byte)
}
