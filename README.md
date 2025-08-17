# 🎵 tagTonic

A powerful MP3 tag editor with a beautiful TUI interface, built with Go, Cobra, and Bubble Tea.

## ✨ Features

- **Manual MP3 Tag Editing** - Edit title, artist, album, genre, year, lyrics, and artwork
- **Automatic Content Fetching** - Fetch lyrics and artwork from multiple APIs
- **Batch Processing** - Process multiple MP3 files at once
- **Beautiful TUI Interface** - Modern, animated interface with purple neumorphic design
- **Image Preview** - View and manage artwork with preview capabilities
- **CLI & TUI** - Both command-line and terminal user interface

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/sumit_pathak/tagTonic.git
cd tagTonic

# Install dependencies
go mod tidy

# Build the application
go build -o tagTonic

# Run the TUI
./tagTonic tui
```

### CLI Usage

```bash
# Launch TUI interface
tagTonic tui

# Edit tags manually
tagTonic edit song.mp3 --title "New Title" --artist "New Artist"

# Fetch lyrics and artwork
tagTonic fetch song.mp3 --lyrics --artwork

# Batch process files
tagTonic batch --dir ./music/ --lyrics --artwork
```

## 📖 Documentation

### CLI Commands

#### `edit` - Manual Tag Editing
```bash
tagTonic edit [file] [flags]
```

**Flags:**
- `--title` - Set title
- `--artist` - Set artist  
- `--album` - Set album
- `--genre` - Set genre
- `--year` - Set year
- `--lyrics` - Path to lyrics file
- `--artwork` - Path to artwork file

#### `fetch` - Fetch Content from APIs
```bash
tagTonic fetch [file] [flags]
```

**Flags:**
- `--lyrics` - Fetch lyrics
- `--artwork` - Fetch artwork
- `--force` - Force overwrite existing content

#### `batch` - Batch Processing
```bash
tagTonic batch [flags]
```

**Flags:**
- `--dir` - Directory containing MP3 files (required)
- `--pattern` - File pattern to match (default: "*.mp3")
- `--lyrics` - Fetch lyrics for all files
- `--artwork` - Fetch artwork for all files
- `--recursive` - Search recursively in subdirectories

#### `tui` - Terminal User Interface
```bash
tagTonic tui
```

Launches the beautiful TUI interface with multiple views:
- **File Browser** - Navigate and select MP3 files
- **Tag Editor** - Edit tags with form interface
- **Artwork View** - Manage and preview artwork
- **Lyrics View** - Edit and view lyrics
- **Help** - Documentation and shortcuts

### TUI Navigation

**View Shortcuts:**
- `1` - File Browser
- `2` - Tag Editor  
- `3` - Artwork View
- `4` - Lyrics View
- `H` - Help
- `Tab` - Cycle through views
- `Ctrl+C` - Quit

**File Browser:**
- `↑/↓` - Navigate files
- `Enter` - Open directory or select file
- `Backspace` - Go to parent directory
- `Ctrl+O` - Open file dialog

**Tag Editor:**
- `Tab` - Next field
- `Shift+Tab` - Previous field
- `Ctrl+S` - Save tags
- `Ctrl+R` - Reload tags
- `Ctrl+F` - Fetch tags from APIs

**Artwork View:**
- `F` - Fetch artwork from APIs
- `R` - Reload artwork
- `S` - Save artwork
- `D` - Delete artwork

**Lyrics View:**
- `F` - Fetch lyrics from APIs
- `R` - Reload lyrics
- `S` - Save lyrics
- `E` - Edit lyrics
- `Escape` - Exit edit mode

## 🎨 Design

tagTonic features a beautiful purple, dark neumorphic theme with:

- **Purple Color Scheme** - Primary purple (#8B5CF6) with dark backgrounds
- **Neumorphic Design** - Subtle shadows and rounded corners
- **Smooth Animations** - Engaging but not distracting animations
- **Responsive Layout** - Adapts to different terminal sizes
- **Modern Typography** - Clean, readable text with proper contrast

## 🔌 API Integration

### Lyrics Sources
- **Lyrics.ovh** - Free, no API key required
- **Genius** - Requires API key
- **Musixmatch** - Requires API key

### Artwork Sources  
- **iTunes** - Free, no API key required
- **Discogs** - Free, no API key required
- **Last.fm** - Requires API key
- **Cover Art Archive** - Requires MusicBrainz integration

## 🏗️ Architecture

```
tagTonic/
├── cmd/                # Cobra CLI commands
│   ├── root.go        # Root command
│   ├── edit.go        # Edit command
│   ├── fetch.go       # Fetch command
│   ├── batch.go       # Batch command
│   └── tui.go         # TUI command
├── tui/               # Bubble Tea TUI
│   ├── app.go         # Main application
│   ├── styles/        # Lipgloss styles
│   └── views/         # TUI views
├── mp3/               # MP3 tag operations
│   └── tags.go        # Tag editor interface
├── fetcher/           # API clients
│   ├── lyrics.go      # Lyrics fetcher
│   └── artwork.go     # Artwork fetcher
├── utils/             # Utility functions
├── config/            # Configuration
└── main.go            # Entry point
```

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Git

### Setup
```bash
# Clone repository
git clone https://github.com/sumit_pathak/tagTonic.git
cd tagTonic

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build
go build -o tagTonic

# Run
./tagTonic tui
```

### Dependencies

**CLI Framework:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management

**TUI Framework:**
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - UI components

**MP3 Processing:**
- `github.com/dhowden/tag` - MP3 tag reading/writing
- `github.com/h2non/filetype` - File type detection

**HTTP & Image:**
- Standard library for HTTP clients
- Standard library for image processing

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./mp3/
go test ./fetcher/
```

## 📝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [tag](https://github.com/dhowden/tag) - MP3 tag library

## 🐛 Issues

If you encounter any issues or have feature requests, please [open an issue](https://github.com/sumit_pathak/tagTonic/issues).

## 📊 Roadmap

- [ ] Support for more audio formats (FLAC, M4A, etc.)
- [ ] Plugin system for custom fetchers
- [ ] Audio preview functionality
- [ ] Tag templates and presets
- [ ] Export/import tag data
- [ ] Advanced artwork editing
- [ ] MusicBrainz integration
- [ ] Undo/redo functionality
- [ ] Drag & drop support (future GUI)
- [ ] Cloud storage integration

---

**Made with ❤️ and Go** 