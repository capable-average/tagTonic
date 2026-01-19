# tagTonic Usage Guide

## Interactive TUI

Launch the terminal user interface:

```bash
tagTonic tui
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate files/fields |
| `Enter` | Edit selected field |
| `Tab` | Switch between panels |
| `Ctrl+S` | Save changes |
| `Ctrl+F` | Search/filter |
| `?` | Show help |
| `q` | Quit |

**Features:** File browser, tag editor, artwork preview (Kitty protocol), lyrics viewer, batch operations, bulk tag editing.

### Bulk Tag Editor (Batch Mode)

Edit tags across multiple files at once with selective field updates.

**Entering Batch Mode:**

1. Press `b` to enable batch mode
2. Use `Space` to select/deselect files
3. Press `Tab` or `Enter` to open the bulk tag editor

**In the Bulk Tag Editor:**

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate fields |
| `Enter` or `e` | Edit field value |
| `s` or `Ctrl+S` | Save/apply tags to all selected files |
| `f` or `Ctrl+F` | Fetch lyrics and artwork for all |
| `Ctrl+L` | Fetch lyrics only |
| `Ctrl+A` | Fetch artwork only |
| `Tab` | Switch to file browser |
| `Esc` | Exit bulk tag editor |

**How it works:**

- Fields display common values if all selected files share the same value
- Fields show `<multiple values>` if values differ across files
- Only fields marked with `[✓]` (enabled and modified) are applied
- Unchanged fields preserve their individual values in each file
- Supported fields: Title, Artist, Album, Year, Genre (Lyrics not supported in bulk edit)

**Example workflow:**

```bash
# In the TUI:
# 1. Press 'b' to enter batch mode
# 2. Navigate and press Space to select multiple MP3s
# 3. Press Tab to open bulk tag editor
# 4. Edit the "Album" or "Artist" field (sets same value for all)
# 5. Press 's' to save - only modified fields are updated
```

---

## CLI Commands

### edit

Edit MP3 tags manually.

```bash
tagTonic edit <file.mp3> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--title` | Set song title |
| `--artist` | Set artist name |
| `--album` | Set album name |
| `--genre` | Set music genre |
| `--year` | Set release year |
| `--lyrics` | Set lyrics from file path |
| `--artwork` | Set artwork from image path |

**Examples:**

```bash
# Edit basic tags
tagTonic edit song.mp3 --title "Song Title" --artist "Artist Name"

# Add lyrics and artwork
tagTonic edit song.mp3 --lyrics ./lyrics.txt --artwork ./cover.jpg
```

---

### fetch

Automatically fetch lyrics and artwork from online sources.

```bash
tagTonic fetch <file.mp3> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--lyrics` | Fetch lyrics |
| `--artwork` | Fetch artwork |
| `--force` | Overwrite existing data |

**Examples:**

```bash
# Fetch lyrics and artwork
tagTonic fetch song.mp3 --lyrics --artwork

# Force overwrite existing data
tagTonic fetch song.mp3 --lyrics --artwork --force
```

---

### batch

Process multiple MP3 files in a directory.

```bash
tagTonic batch [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dir` | Directory to process (required) |
| `--pattern` | File pattern (default: `*.mp3`) |
| `--recursive` | Process subdirectories |
| `--lyrics` | Fetch lyrics for all files |
| `--artwork` | Fetch artwork for all files |
| `--force` | Overwrite existing data |

**Examples:**

```bash
# Fetch metadata for all MP3s in directory
tagTonic batch --dir ./music --lyrics --artwork

# Recursive processing
tagTonic batch --dir ./music --recursive --lyrics --artwork

# Overwrite existing artwork
tagTonic batch --dir ./music --recursive --artwork --force
```

---

### show

Display metadata for an MP3 file without editing.

```bash
tagTonic show <file.mp3>
```

Displays title, artist, album, genre, year, lyrics, and artwork information.

---

## Configuration

Create an optional configuration file at `~/.config/tagTonic/config.yaml`:

```yaml
# Genius API token (recommended for better rate limits)
genius_api_key: "your_token_here"

# Preferred sources
preferred_lyrics_source: "genius"      # Options: genius, lyrics.ovh, auto
preferred_artwork_source: "itunes"     # Options: itunes, musicbrainz, archive, auto

# Performance settings
max_artwork_size: 500                  # Maximum artwork size in pixels
batch_concurrency: 5                   # Number of concurrent batch operations
```

### Getting a Genius API Token

1. Create a free account at [genius.com](https://genius.com)
2. Go to [genius.com/api-clients](https://genius.com/api-clients) and click "New API Client"
3. Fill in the form (use `http://localhost` for URL fields)
4. Click "Generate Access Token" and copy the token
5. Add to config: `genius_api_key: "your_token_here"`

Configuration is optional. The application functions without it but may encounter rate limits or not-so-good results on free APIs.

Note: An example config file is present for reference [config.yaml](/config.example.yaml)

---

## Common Workflows

### Single File

```bash
# Auto-fetch metadata
tagTonic fetch song.mp3 --lyrics --artwork

# Manual editing if fetch fails
tagTonic edit song.mp3 --title "Title" --artist "Artist"
```

### Entire Library

```bash
# Batch process all files
tagTonic batch --dir ~/Music --recursive --lyrics --artwork

# Fine-tune individual files
tagTonic tui
```

### Metadata Inspection

```bash
tagTonic show song.mp3
```

---

## Help

```bash
tagTonic --help              # General help
tagTonic [command] --help    # Command-specific help
```

[Back to README](../README.md)