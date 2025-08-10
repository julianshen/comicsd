package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"comicsd/internal/downloader"
	"comicsd/internal/epub"
	"comicsd/internal/info"
	"comicsd/internal/mcp"

	"github.com/chromedp/chromedp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: comicsd <command> [args]\ncommands: search, info, download, mcp")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		format := searchCmd.String("format", "text", "output format (text or json)")
		searchCmd.Parse(os.Args[2:])
		if searchCmd.NArg() < 1 {
			log.Fatal("keyword required")
		}
		keyword := searchCmd.Arg(0)
		ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
		defer cancel()
		fetcher := info.NewComicInfoFetcher(ctx)
		results, err := fetcher.SearchComics(keyword)
		if err != nil {
			log.Fatal(err)
		}
		if *format == "json" {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		} else {
			for _, r := range results {
				fmt.Printf("%s %s\n", r.ID, r.Title)
			}
		}

	case "info":
		infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
		format := infoCmd.String("format", "text", "output format (text or json)")
		infoCmd.Parse(os.Args[2:])
		if infoCmd.NArg() < 1 {
			log.Fatal("comic id required")
		}
		comicID := infoCmd.Arg(0)
		ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
		defer cancel()
		fetcher := info.NewComicInfoFetcher(ctx)
		ci, err := fetcher.GetComicInfo(comicID)
		if err != nil {
			log.Fatal(err)
		}
		if *format == "json" {
			j, _ := ci.ToJSON()
			fmt.Println(j)
		} else {
			fmt.Print(ci.ToPlainText())
		}

	case "download":
		dlCmd := flag.NewFlagSet("download", flag.ExitOnError)
		format := dlCmd.String("format", "cbz", "output format (cbz or epub)")
		dlCmd.Parse(os.Args[2:])
		args := dlCmd.Args()
		if len(args) < 3 {
			log.Fatal("usage: comicsd download [-format cbz|epub] <comic_id> <title> <chapter_ids...>")
		}
		comicID := args[0]
		title := args[1]
		chapterIDs := args[2:]
		ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(string, ...interface{}) {}))
		defer cancel()
		file, err := os.Create(fmt.Sprintf("%s.%s", title, *format))
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		if *format == "cbz" {
			if err := downloadToCBZ(ctx, comicID, chapterIDs, file); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := downloadToEPUB(ctx, title, comicID, chapterIDs, file); err != nil {
				log.Fatal(err)
			}
		}

	case "mcp":
		server := mcp.NewMCPServer()
		if err := server.Serve(); err != nil {
			log.Fatal(err)
		}

	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func downloadToCBZ(ctx context.Context, comicID string, chapters []string, file *os.File) error {
	cbz := zip.NewWriter(file)
	defer cbz.Close()
	page := 0
	for _, chapterID := range chapters {
		cc, err := downloader.NewDownload(ctx, comicID, chapterID)
		if err != nil {
			return err
		}
		for _, p := range cc.Pages {
			w, err := cbz.Create(fmt.Sprintf("%d.jpg", page))
			if err != nil {
				return err
			}
			if err := cc.DownloadPageTo(p, w); err != nil {
				return err
			}
			page++
		}
	}
	return nil
}

func downloadToEPUB(ctx context.Context, title, comicID string, chapters []string, file *os.File) error {
	writer := epub.NewEPUBWriter(file, title)
	defer writer.Close()
	page := 0
	for _, chapterID := range chapters {
		cc, err := downloader.NewDownload(ctx, comicID, chapterID)
		if err != nil {
			return err
		}
		for _, p := range cc.Pages {
			var buf bytes.Buffer
			if err := cc.DownloadPageTo(p, &buf); err != nil {
				return err
			}
			fname := fmt.Sprintf("%d.jpg", page)
			if err := writer.AddPage(fname, buf.Bytes()); err != nil {
				return err
			}
			page++
		}
	}
	return nil
}
