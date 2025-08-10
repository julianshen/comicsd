package mcp

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"comicsd/internal/downloader"
	"comicsd/internal/epub"
	"comicsd/internal/info"

	"github.com/chromedp/chromedp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchParams represents the parameters for the search tool
type SearchParams struct {
	Keyword string `json:"keyword"`
}

// InfoParams represents the parameters for the info tool
type InfoParams struct {
	ComicID string `json:"comic_id"`
}

// GenerateConfigParams represents the parameters for the config generation tool
type GenerateConfigParams struct {
	ComicID    string   `json:"comic_id"`
	Chapters   []string `json:"chapters"`
	Title      string   `json:"title"`
	Format     string   `json:"format"`
	ConfigName string   `json:"config_name"`
}

// SummarizeParams represents the parameters for the summarize tool
type SummarizeParams struct {
	ComicID  string   `json:"comic_id"`
	Chapters []string `json:"chapters"`
	Title    string   `json:"title"`
	Format   string   `json:"format"`
}

// NewOfficialMCPServer creates a new MCP server using the official SDK
func NewOfficialMCPServer() *mcp.Server {
	log.SetOutput(os.Stderr)
	log.Println("Creating official MCP server...")

	server := mcp.NewServer("comicsd", "1.0.0", nil)

	// Add search tool
	log.Println("Adding search tool...")
	server.AddTools(
		mcp.NewServerTool("search_comics", "Search for comics by keyword", searchComicsOfficial, mcp.Input(
			mcp.Property("keyword", mcp.Description("Keyword to search for comics")),
		)),
	)

	// Add info tool
	log.Println("Adding info tool...")
	server.AddTools(
		mcp.NewServerTool("get_comic_info", "Get comic information", getComicInfoOfficial, mcp.Input(
			mcp.Property("comic_id", mcp.Description("Comic ID to get information for")),
		)),
	)

	// Add config generation tool
	log.Println("Adding config generation tool...")
	server.AddTools(
		mcp.NewServerTool("generate_config", "Generate summarization configuration for specified comic and chapters", generateConfigOfficial, mcp.Input(
			mcp.Property("comic_id", mcp.Description("Comic ID to summarize")),
			mcp.Property("chapters", mcp.Description("List of chapter IDs to include")),
			mcp.Property("title", mcp.Description("Comic title for the configuration")),
			mcp.Property("format", mcp.Description("Output format (cbz or epub)")),
			mcp.Property("config_name", mcp.Description("Name for this configuration entry")),
		)),
	)

	// Add summarize tool
	log.Println("Adding summarize tool...")
	server.AddTools(
		mcp.NewServerTool("summarize_comic", "Summarize specific chapters of a comic in CBZ or EPUB format", summarizeComicOfficial, mcp.Input(
			mcp.Property("comic_id", mcp.Description("Comic ID to summarize")),
			mcp.Property("chapters", mcp.Description("List of chapter IDs to summarize")),
			mcp.Property("title", mcp.Description("Comic title for filename")),
			mcp.Property("format", mcp.Description("Output format (cbz or epub)")),
		)),
	)

	log.Println("Official MCP server created successfully")
	return server
}

// searchComicsOfficial implements search using the official SDK
func searchComicsOfficial(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchParams]) (*mcp.CallToolResultFor[any], error) {
	log.Printf("Search called with keyword: %s", params.Arguments.Keyword)

	chromectx, cancel := chromedp.NewContext(ctx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	fetcher := info.NewComicInfoFetcher(chromectx)
	results, err := fetcher.SearchComics(params.Arguments.Keyword)
	if err != nil {
		log.Printf("search comics error: %v", err)
		return nil, fmt.Errorf("failed to search comics: %w", err)
	}

	// Return pure JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}, nil
}

// getComicInfoOfficial implements info retrieval using the official SDK
func getComicInfoOfficial(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[InfoParams]) (*mcp.CallToolResultFor[any], error) {
	log.Printf("Info called with comic ID: %s", params.Arguments.ComicID)

	chromectx, cancel := chromedp.NewContext(ctx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	fetcher := info.NewComicInfoFetcher(chromectx)
	comicInfo, err := fetcher.GetComicInfo(params.Arguments.ComicID)
	if err != nil {
		log.Printf("get comic info error: %v", err)
		return nil, fmt.Errorf("failed to get comic info: %w", err)
	}

	// Return pure JSON
	jsonData, err := json.MarshalIndent(comicInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comic info: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}, nil
}

// generateConfigOfficial implements config generation using the official SDK
func generateConfigOfficial(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GenerateConfigParams]) (*mcp.CallToolResultFor[any], error) {
	log.Printf("Generate config called with comic ID: %s, chapters: %v, format: %s",
		params.Arguments.ComicID, params.Arguments.Chapters, params.Arguments.Format)

	// Validate format
	format := params.Arguments.Format
	if format != "cbz" && format != "epub" && format != "" {
		return nil, fmt.Errorf("invalid format: %s. Use 'cbz' or 'epub'", format)
	}
	if format == "" {
		format = "cbz" // default
	}

	// Validate inputs
	if params.Arguments.ComicID == "" {
		return nil, fmt.Errorf("comic_id is required")
	}
	if len(params.Arguments.Chapters) == 0 {
		return nil, fmt.Errorf("at least one chapter is required")
	}
	if params.Arguments.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if params.Arguments.ConfigName == "" {
		return nil, fmt.Errorf("config_name is required")
	}

	// Generate TOML configuration
	var tomlConfig strings.Builder
	tomlConfig.WriteString(fmt.Sprintf("[%s]\n", params.Arguments.ConfigName))
	tomlConfig.WriteString(fmt.Sprintf("title = \"%s\"\n", params.Arguments.Title))
	tomlConfig.WriteString(fmt.Sprintf("mangaid = \"%s\"\n", params.Arguments.ComicID))

	// Format chapters array
	tomlConfig.WriteString("chapters = [")
	for i, chapter := range params.Arguments.Chapters {
		if i > 0 {
			tomlConfig.WriteString(", ")
		}
		tomlConfig.WriteString(fmt.Sprintf("\"%s\"", chapter))
	}
	tomlConfig.WriteString("]\n")

	// Add format if not default
	if format != "cbz" {
		tomlConfig.WriteString(fmt.Sprintf("format = \"%s\"\n", format))
	}

	configContent := tomlConfig.String()

	// Create response
	responseText := fmt.Sprintf("Generated summarization configuration for comic '%s':\n\n", params.Arguments.Title)
	responseText += "Configuration content:\n"
	responseText += "```toml\n"
	responseText += configContent
	responseText += "```\n\n"
	responseText += fmt.Sprintf("Summary:\n")
	responseText += fmt.Sprintf("- Comic ID: %s\n", params.Arguments.ComicID)
	responseText += fmt.Sprintf("- Title: %s\n", params.Arguments.Title)
	responseText += fmt.Sprintf("- Format: %s\n", format)
	responseText += fmt.Sprintf("- Chapters: %d selected\n", len(params.Arguments.Chapters))
	responseText += fmt.Sprintf("- Config name: %s\n\n", params.Arguments.ConfigName)
	responseText += "To use this configuration:\n"
	responseText += "1. Save the TOML content to a file named 'summarize.toml'\n"
	responseText += "2. Run: ./comicsd summarize\n"
	responseText += fmt.Sprintf("3. Or run with specific format: ./comicsd summarize -format %s", format)

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: responseText}},
	}, nil
}

// summarizeComicOfficial implements comic summarization (downloading) using the official SDK
func summarizeComicOfficial(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[SummarizeParams]) (*mcp.CallToolResultFor[any], error) {
	log.Printf("Summarize called with comic ID: %s, chapters: %v, format: %s",
		params.Arguments.ComicID, params.Arguments.Chapters, params.Arguments.Format)

	// Validate format
	format := params.Arguments.Format
	if format != "cbz" && format != "epub" && format != "" {
		return nil, fmt.Errorf("invalid format: %s. Use 'cbz' or 'epub'", format)
	}
	if format == "" {
		format = "cbz" // default
	}

	// Validate inputs
	if params.Arguments.ComicID == "" {
		return nil, fmt.Errorf("comic_id is required")
	}
	if len(params.Arguments.Chapters) == 0 {
		return nil, fmt.Errorf("at least one chapter is required")
	}
	if params.Arguments.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Create chromedp context for downloading
	chromectx, cancel := chromedp.NewContext(ctx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	// Create output file
	filename := fmt.Sprintf("%s.%s", params.Arguments.Title, format)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	var responseText string

	if format == "cbz" {
		err = summarizeToCBZ(chromectx, params.Arguments, file)
		if err != nil {
			return nil, fmt.Errorf("failed to summarize to CBZ: %w", err)
		}
		responseText = fmt.Sprintf("Successfully summarized %d chapters to %s (CBZ format)", len(params.Arguments.Chapters), filename)
	} else {
		err = summarizeToEPUB(chromectx, params.Arguments, file)
		if err != nil {
			return nil, fmt.Errorf("failed to summarize to EPUB: %w", err)
		}
		responseText = fmt.Sprintf("Successfully summarized %d chapters to %s (EPUB format)", len(params.Arguments.Chapters), filename)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: responseText}},
	}, nil
}

// summarizeToCBZ downloads comic chapters to CBZ format
func summarizeToCBZ(ctx context.Context, params SummarizeParams, file *os.File) error {
	cbz := zip.NewWriter(file)
	defer cbz.Close()

	page := 0
	for chn, chapterID := range params.Chapters {
		log.Printf("Summarizing chapter %s (%d/%d)", chapterID, chn+1, len(params.Chapters))
		cc, err := downloader.NewDownload(ctx, params.ComicID, chapterID)
		if err != nil {
			return err
		}

		for n := range cc.Pages {
			log.Printf("Summarizing page %d/%d/%d", n, len(cc.Pages), chn)
			w, err := cbz.Create(fmt.Sprintf("%d.jpg", page))
			if err != nil {
				return err
			}

			err = cc.DownloadPageTo(cc.Pages[n], w)
			if err != nil {
				return err
			}
			page++
		}
	}

	return nil
}

// summarizeToEPUB downloads comic chapters to EPUB format
func summarizeToEPUB(ctx context.Context, params SummarizeParams, file *os.File) error {
	epubWriter := epub.NewEPUBWriter(file, params.Title)
	defer epubWriter.Close()

	page := 0
	for chn, chapterID := range params.Chapters {
		log.Printf("Summarizing chapter %s (%d/%d)", chapterID, chn+1, len(params.Chapters))
		cc, err := downloader.NewDownload(ctx, params.ComicID, chapterID)
		if err != nil {
			return err
		}

		for n := range cc.Pages {
			log.Printf("Summarizing page %d/%d/%d", n, len(cc.Pages), chn)

			// Download image data to memory
			var buf bytes.Buffer
			err := cc.DownloadPageTo(cc.Pages[n], &buf)
			if err != nil {
				return err
			}

			// Add page to EPUB
			filename := fmt.Sprintf("%d.jpg", page)
			err = epubWriter.AddPage(filename, buf.Bytes())
			if err != nil {
				return err
			}
			page++
		}
	}

	return nil
}

// ServeOfficial runs the official MCP server
func ServeOfficial() error {
	log.Println("Starting official MCP server...")
	server := NewOfficialMCPServer()

	transport := mcp.NewStdioTransport()
	err := server.Run(context.Background(), transport)
	if err != nil {
		log.Printf("Official MCP server error: %v", err)
	}

	log.Println("Official MCP server stopped")
	return err
}
