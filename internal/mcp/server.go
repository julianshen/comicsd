package mcp

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"comicsd/internal/downloader"
	"comicsd/internal/epub"
	"comicsd/internal/info"

	"github.com/chromedp/chromedp"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// SearchComicsArgs defines the arguments for searching comics
type SearchComicsArgs struct {
	Keyword string `json:"keyword" jsonschema:"required,description=Keyword to search for comics"`
}

// GetComicInfoArgs defines the arguments for getting comic information
type GetComicInfoArgs struct {
	ComicID string `json:"comic_id" jsonschema:"required,description=Comic ID to get information for"`
}

// DownloadComicArgs defines the arguments for downloading comics
type DownloadComicArgs struct {
	ComicID    string   `json:"comic_id" jsonschema:"required,description=Comic ID to download"`
	ChapterIDs []string `json:"chapter_ids" jsonschema:"required,description=List of chapter IDs to download"`
	Format     string   `json:"format" jsonschema:"required,description=Output format (cbz or epub)"`
	Title      string   `json:"title" jsonschema:"required,description=Comic title for filename"`
}

type pageTask struct {
	chapterID string
	pageID    string
	index     int
}

func workerCount() int {
	if v := os.Getenv("COMICSD_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 4
}

// MCPServer wraps the MCP functionality
type MCPServer struct {
	server *mcp_golang.Server
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer() *MCPServer {
	// Add debug output to stderr
	log.SetOutput(os.Stderr)
	log.Println("Creating MCP server...")

	transport := stdio.NewStdioServerTransport()
	server := mcp_golang.NewServer(transport)

	mcpServer := &MCPServer{
		server: server,
	}

	// Register tools
	log.Println("Registering MCP tools...")
	mcpServer.registerTools()
	log.Println("MCP server creation complete")

	return mcpServer
}

// registerTools registers all available tools with the MCP server
func (m *MCPServer) registerTools() {
	// Search comics tool
	log.Println("Registering search_comics tool...")
	err := m.server.RegisterTool(
		"search_comics",
		"Search for comics by keyword on manhuagui.com",
		m.searchComics,
	)
	if err != nil {
		log.Printf("Failed to register search_comics tool: %v", err)
	}

	// Get comic info tool
	log.Println("Registering get_comic_info tool...")
	err = m.server.RegisterTool(
		"get_comic_info",
		"Get detailed information about a specific comic including chapters",
		m.getComicInfo,
	)
	if err != nil {
		log.Printf("Failed to register get_comic_info tool: %v", err)
	}

	log.Println("All tools registered successfully")
}

// searchComics implements the search functionality for MCP
func (m *MCPServer) searchComics(args SearchComicsArgs) (*mcp_golang.ToolResponse, error) {
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	fetcher := info.NewComicInfoFetcher(ctx)
	results, err := fetcher.SearchComics(args.Keyword)
	if err != nil {
		return nil, fmt.Errorf("failed to search comics: %w", err)
	}

	// Format results for display
	var responseText string
	if len(results) == 0 {
		responseText = fmt.Sprintf("No comics found for keyword '%s'", args.Keyword)
	} else {
		responseText = fmt.Sprintf("Found %d comics for '%s':\n\n", len(results), args.Keyword)
		for i, result := range results {
			responseText += fmt.Sprintf("%d. %s (ID: %s)\n", i+1, result.Title, result.ID)
		}
	}

	// Also return structured data
	jsonData, _ := json.MarshalIndent(results, "", "  ")

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(responseText),
		mcp_golang.NewTextContent(fmt.Sprintf("Raw JSON data:\n```json\n%s\n```", string(jsonData))),
	), nil
}

// getComicInfo implements the comic info functionality for MCP
func (m *MCPServer) getComicInfo(args GetComicInfoArgs) (*mcp_golang.ToolResponse, error) {
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	fetcher := info.NewComicInfoFetcher(ctx)
	comicInfo, err := fetcher.GetComicInfo(args.ComicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic info: %w", err)
	}

	// Format basic info
	responseText := fmt.Sprintf("Comic Information:\n")
	responseText += fmt.Sprintf("ID: %s\n", comicInfo.ID)
	responseText += fmt.Sprintf("Title: %s\n", comicInfo.Title)
	if comicInfo.Author != "" {
		responseText += fmt.Sprintf("Author: %s\n", comicInfo.Author)
	}
	if comicInfo.Status != "" {
		responseText += fmt.Sprintf("Status: %s\n", comicInfo.Status)
	}
	responseText += fmt.Sprintf("Total Chapters: %d\n\n", len(comicInfo.Chapters))

	// List first 10 chapters as examples
	responseText += "Recent Chapters:\n"
	limit := len(comicInfo.Chapters)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		chapter := comicInfo.Chapters[i]
		responseText += fmt.Sprintf("  %d. [%s] %s\n", i+1, chapter.ID, chapter.Title)
	}

	if len(comicInfo.Chapters) > 10 {
		responseText += fmt.Sprintf("  ... and %d more chapters\n", len(comicInfo.Chapters)-10)
	}

	// Return structured data too
	jsonData, _ := json.MarshalIndent(comicInfo, "", "  ")

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(responseText),
		mcp_golang.NewTextContent(fmt.Sprintf("Complete data (JSON):\n```json\n%s\n```", string(jsonData))),
	), nil
}

// downloadComic implements the download functionality for MCP
func (m *MCPServer) downloadComic(args DownloadComicArgs) (*mcp_golang.ToolResponse, error) {
	// Validate format
	if args.Format != "cbz" && args.Format != "epub" {
		return nil, fmt.Errorf("invalid format: %s. Use 'cbz' or 'epub'", args.Format)
	}

	if len(args.ChapterIDs) == 0 {
		return nil, fmt.Errorf("no chapters specified for download")
	}

	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	// Create output file
	filename := fmt.Sprintf("%s.%s", args.Title, args.Format)
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	var responseText string

	if args.Format == "cbz" {
		err = m.downloadToCBZ(ctx, args, file)
		if err != nil {
			return nil, fmt.Errorf("failed to download CBZ: %w", err)
		}
		responseText = fmt.Sprintf("Successfully downloaded %d chapters to %s (CBZ format)", len(args.ChapterIDs), filename)
	} else {
		err = m.downloadToEPUB(ctx, args, file)
		if err != nil {
			return nil, fmt.Errorf("failed to download EPUB: %w", err)
		}
		responseText = fmt.Sprintf("Successfully downloaded %d chapters to %s (EPUB format)", len(args.ChapterIDs), filename)
	}

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(responseText),
	), nil
}

// downloadToCBZ downloads comic chapters to CBZ format
func (m *MCPServer) downloadToCBZ(ctx context.Context, args DownloadComicArgs, file *os.File) error {
	cbz := zip.NewWriter(file)
	defer cbz.Close()

	// Build page tasks
	var tasks []pageTask
	for chn, chapterID := range args.ChapterIDs {
		log.Printf("Preparing chapter %s (%d/%d)", chapterID, chn+1, len(args.ChapterIDs))
		cc, err := downloader.NewDownload(ctx, args.ComicID, chapterID)
		if err != nil {
			return err
		}
		for _, p := range cc.Pages {
			tasks = append(tasks, pageTask{chapterID: chapterID, pageID: p, index: len(tasks)})
		}
	}

	total := len(tasks)
	if total == 0 {
		return nil
	}

	workerCnt := workerCount()
	log.Printf("Starting %d workers for %d pages", workerCnt, total)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([][]byte, total)
	taskCh := make(chan pageTask)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var done int32

	for i := 0; i < workerCnt; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			wctx, wcancel := chromedp.NewContext(ctx)
			defer wcancel()
			dlMap := make(map[string]*downloader.ComicsDL)
			for t := range taskCh {
				if ctx.Err() != nil {
					return
				}
				dl, ok := dlMap[t.chapterID]
				if !ok {
					var err error
					dl, err = downloader.NewDownload(wctx, args.ComicID, t.chapterID)
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
					dlMap[t.chapterID] = dl
				}
				var buf bytes.Buffer
				if err := dl.DownloadPageTo(t.pageID, &buf); err != nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
				results[t.index] = buf.Bytes()
				completed := atomic.AddInt32(&done, 1)
				log.Printf("Worker %d downloaded page %d/%d", id, completed, total)
			}
		}(i + 1)
	}

	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
	}

	for i, data := range results {
		w, err := cbz.Create(fmt.Sprintf("%d.jpg", i))
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// downloadToEPUB downloads comic chapters to EPUB format
func (m *MCPServer) downloadToEPUB(ctx context.Context, args DownloadComicArgs, file *os.File) error {
	epubWriter := epub.NewEPUBWriter(file, args.Title)
	defer epubWriter.Close()

	// Build page tasks
	var tasks []pageTask
	for chn, chapterID := range args.ChapterIDs {
		log.Printf("Preparing chapter %s (%d/%d)", chapterID, chn+1, len(args.ChapterIDs))
		cc, err := downloader.NewDownload(ctx, args.ComicID, chapterID)
		if err != nil {
			return err
		}
		for _, p := range cc.Pages {
			tasks = append(tasks, pageTask{chapterID: chapterID, pageID: p, index: len(tasks)})
		}
	}

	total := len(tasks)
	if total == 0 {
		return nil
	}

	workerCnt := workerCount()
	log.Printf("Starting %d workers for %d pages", workerCnt, total)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([][]byte, total)
	taskCh := make(chan pageTask)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var done int32

	for i := 0; i < workerCnt; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			wctx, wcancel := chromedp.NewContext(ctx)
			defer wcancel()
			dlMap := make(map[string]*downloader.ComicsDL)
			for t := range taskCh {
				if ctx.Err() != nil {
					return
				}
				dl, ok := dlMap[t.chapterID]
				if !ok {
					var err error
					dl, err = downloader.NewDownload(wctx, args.ComicID, t.chapterID)
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
					dlMap[t.chapterID] = dl
				}
				var buf bytes.Buffer
				if err := dl.DownloadPageTo(t.pageID, &buf); err != nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
				results[t.index] = buf.Bytes()
				completed := atomic.AddInt32(&done, 1)
				log.Printf("Worker %d downloaded page %d/%d", id, completed, total)
			}
		}(i + 1)
	}

	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
	}

	for i, data := range results {
		filename := fmt.Sprintf("%d.jpg", i)
		if err := epubWriter.AddPage(filename, data); err != nil {
			return err
		}
	}

	return nil
}

// Serve starts the MCP server
func (m *MCPServer) Serve() error {
	log.Println("Starting MCP server...")

	// Add recovery to catch any panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("MCP server panic: %v", r)
		}
	}()

	err := m.server.Serve()
	if err != nil {
		log.Printf("MCP server error: %v", err)
	}

	log.Println("MCP server stopped")
	return err
}
