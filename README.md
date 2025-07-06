# Comic Downloader

A Go-based comic downloader that supports downloading comics from manhuagui.com in CBZ and EPUB formats, with both CLI and MCP (Model Context Protocol) server capabilities.

## Features

- **Search Comics**: Search for comics by keyword
- **Download Comics**: Download comics in CBZ (for comic readers) or EPUB (for e-readers) format
- **Multiple Interfaces**:
  - Command-line interface (CLI)
  - MCP server for AI assistant integration
- **Format Support**:
  - CBZ: Standard comic book archive format
  - EPUB: E-book format with centered, full-page image layout

## Installation

```bash
git clone <repository-url>
cd comicsd
go build -o comicsd ./cmd/comicsd
```

## Usage

### Command Line Interface

#### Search for Comics
```bash
./comicsd search <keyword>
./comicsd search <keyword> -format json
```

#### Get Comic Information
```bash
./comicsd info <comic_id>
./comicsd info <comic_id> -format json
```

#### Download Comics
Create a `download.toml` file with your comic configuration:

```toml
[manga_name]
title = "Comic Title"
mangaid = "12345"
chapters = ["chapter1", "chapter2", "chapter3"]
format = "epub"  # optional, defaults to "cbz"
```

Then run:
```bash
./comicsd download
./comicsd download -format epub
```

### MCP Server Mode

Run as an MCP server for AI assistant integration:

```bash
./comicsd mcp
```

See `docs/MCP_README.md` for detailed MCP integration instructions.

## Project Structure

```
comicsd/
├── cmd/
│   └── comicsd/          # Main application entry point
│       └── main.go
├── internal/             # Private application code
│   ├── downloader/       # Comic downloading logic
│   ├── epub/            # EPUB generation
│   ├── info/            # Comic information fetching
│   └── mcp/             # MCP server implementation
├── docs/                # Documentation
│   ├── MCP_README.md    # MCP integration guide
│   └── claude_desktop_config.json
├── go.mod
├── go.sum
└── README.md
```

## Requirements

- Go 1.21+
- Chrome/Chromium browser (for web scraping)
- Internet connection

## Dependencies

- `github.com/chromedp/chromedp` - Web scraping
- `github.com/spf13/viper` - Configuration management
- `github.com/metoro-io/mcp-golang` - MCP server support
- `github.com/rivo/tview` - Terminal UI components
- `github.com/gdamore/tcell/v2` - Terminal handling

## Examples

### Search and Download Workflow

1. Search for comics:
   ```bash
   ./comicsd search "manga title"
   ```

2. Get detailed information:
   ```bash
   ./comicsd info 24332
   ```

3. Create download configuration and download:
   ```bash
   # Edit download.toml with desired chapters
   ./comicsd download -format epub
   ```

### MCP Integration

When integrated with Claude Desktop or other MCP-compatible AI assistants, you can use natural language:

- "Search for comics about dragons"
- "Get information about comic 24332"
- "Download the first 5 chapters of comic 24332 in EPUB format"

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license information here]

## Disclaimer

This tool is for educational purposes. Please respect copyright laws and the terms of service of the websites you're accessing.