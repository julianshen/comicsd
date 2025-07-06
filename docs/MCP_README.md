# Comic Downloader MCP Server

This comic downloader now supports running as an MCP (Model Context Protocol) server, allowing AI assistants like Claude to interact with it through natural language prompts.

## What is MCP?

MCP (Model Context Protocol) is a standard that allows AI assistants to connect to external tools and data sources. When running as an MCP server, this comic downloader can be controlled by AI assistants through natural language conversations.

## Available Tools

The MCP server provides four main tools:

### 1. `search_comics`
- **Purpose**: Search for comics by keyword on manhuagui.com
- **Parameters**: 
  - `keyword` (string, required): Keyword to search for comics
- **Returns**: List of matching comics with IDs and titles

### 2. `get_comic_info`
- **Purpose**: Get detailed information about a specific comic
- **Parameters**:
  - `comic_id` (string, required): Comic ID to get information for
- **Returns**: Comic details including title, author, status, and chapter list

### 3. `generate_config`
- **Purpose**: Generate summarization configuration file for specified comic and chapters
- **Parameters**:
  - `comic_id` (string, required): Comic ID to summarize
  - `chapters` (array of strings, required): List of chapter IDs to include
  - `title` (string, required): Comic title for the configuration
  - `format` (string, required): Output format ("cbz" or "epub")
  - `config_name` (string, required): Name for this configuration entry
- **Returns**: Generated TOML configuration content

### 4. `summarize_comic`
- **Purpose**: Directly summarize specific chapters of a comic in CBZ or EPUB format
- **Parameters**:
  - `comic_id` (string, required): Comic ID to summarize
  - `chapters` (array of strings, required): List of chapter IDs to summarize
  - `title` (string, required): Comic title for filename
  - `format` (string, required): Output format ("cbz" or "epub")
- **Returns**: Success message with filename

## Usage

### Starting the MCP Server

```bash
./comicsd mcp
```

The server will start and listen for MCP requests via stdio.

### Claude Desktop Integration

1. Copy the provided `claude_desktop_config.json` to your Claude Desktop configuration directory:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

2. Update the `command` path in the config to point to your compiled `comicsd` binary:

```json
{
  "mcpServers": {
    "comicsd": {
      "command": "/path/to/your/comicsd",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

3. Restart Claude Desktop

### Example Conversations with Claude

Once configured, you can have natural language conversations with Claude like:

**User**: "Search for comics about 藥屋"

**Claude**: *Uses search_comics tool and shows results*

**User**: "Get more information about comic ID 24332"

**Claude**: *Uses get_comic_info tool to show detailed information*

**User**: "Generate a summarization config for comic 24332 with chapters 566271, 564917, and 562492 in EPUB format"

**Claude**: *Uses generate_config tool to create a TOML configuration*

**User**: "Summarize the first 3 chapters of comic 35937 in EPUB format"

**Claude**: *Uses get_comic_info to find chapter IDs, then uses summarize_comic to directly download them*

## Features

- **Natural Language Interface**: Control the comic downloader through conversational AI
- **Smart Chapter Selection**: AI can help you select specific chapters or ranges
- **Format Flexibility**: Choose between CBZ (for comic readers) or EPUB (for e-readers)
- **Progress Feedback**: Real-time download progress and status updates
- **Error Handling**: Descriptive error messages for troubleshooting

## Requirements

- Compiled `comicsd` binary
- Claude Desktop or other MCP-compatible AI assistant
- Internet connection for accessing manhuagui.com
- Chrome/Chromium browser (for web scraping)

## Troubleshooting

1. **Server won't start**: Ensure the binary path in the config is correct
2. **No tools available**: Check that Claude Desktop has been restarted after config changes
3. **Download failures**: Verify internet connection and that the target comic/chapters exist
4. **Permission errors**: Ensure the binary has execute permissions and write access to the current directory

## Security Note

This MCP server downloads content from manhuagui.com. Ensure you comply with the website's terms of service and local copyright laws when using this tool.