package info

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/chromedp/chromedp"
	"go.uber.org/multierr"
)

type ComicInfo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Chapters    []Chapter `json:"chapters"`
}

type Chapter struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type SearchResult struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type ComicInfoFetcher struct {
	ctx context.Context
}

func NewComicInfoFetcher(ctx context.Context) *ComicInfoFetcher {
	return &ComicInfoFetcher{ctx: ctx}
}

// textContent extracts text content using chromedp. Defined as a variable for tests.
var textContent = func(ctx context.Context, sel string, res *string) error {
	return chromedp.Text(sel, res, chromedp.ByQuery).Do(ctx)
}

// evalJS evaluates JavaScript using chromedp. Defined as a variable for tests.
var evalJS = func(ctx context.Context, expr string, res interface{}) error {
	return chromedp.Evaluate(expr, res).Do(ctx)
}

// fillComicInfo fills the ComicInfo struct by scraping the page.
func (c *ComicInfoFetcher) fillComicInfo(info *ComicInfo) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var err error

		// Get comic title
		var title string
		if e := textContent(ctx, `.book-title h1`, &title); e != nil {
			err = multierr.Append(err, fmt.Errorf("get title: %w", e))
		} else {
			info.Title = strings.TrimSpace(title)
		}

		// Get author and status from detail list
		var detailText string
		if e := textContent(ctx, `.book-detail .detail-list`, &detailText); e != nil {
			err = multierr.Append(err, fmt.Errorf("get detail: %w", e))
		} else {
			if strings.Contains(detailText, "作者") || strings.Contains(detailText, "漫畫作者") {
				re := regexp.MustCompile(`作者[：:]\s*([^\n\r]+)`)
				matches := re.FindStringSubmatch(detailText)
				if len(matches) > 1 {
					info.Author = strings.TrimSpace(matches[1])
				}
			}
			if strings.Contains(detailText, "狀態") || strings.Contains(detailText, "状态") {
				re := regexp.MustCompile(`狀態[：:]\s*([^\n\r]+)`)
				matches := re.FindStringSubmatch(detailText)
				if len(matches) > 1 {
					info.Status = strings.TrimSpace(matches[1])
				}
			}
		}

		// Get description
		var description string
		if e := textContent(ctx, `#intro-all`, &description); e != nil {
			err = multierr.Append(err, fmt.Errorf("get description: %w", e))
		} else {
			info.Description = strings.TrimSpace(description)
		}

		// Get chapters - use evaluate to get href attributes and titles
		var chapterData []map[string]string
		if e := evalJS(ctx, `Array.from(document.querySelectorAll('.chapter-list li a')).map(link => ({href: link.getAttribute('href'), title: link.textContent.trim(),}))`, &chapterData); e != nil {
			err = multierr.Append(err, fmt.Errorf("get chapters: %w", e))
		} else {
			for _, data := range chapterData {
				link := data["href"]
				title := data["title"]

				// Extract chapter ID from URL
				re := regexp.MustCompile(`/comic/\d+/(\d+)\.html`)
				matches := re.FindStringSubmatch(link)
				chapterID := ""
				if len(matches) > 1 {
					chapterID = matches[1]
				}

				chapter := Chapter{
					ID:    chapterID,
					Title: title,
					URL:   link,
				}
				info.Chapters = append(info.Chapters, chapter)
			}
		}

		return err
	})
}

func (c *ComicInfoFetcher) GetComicInfo(comicID string) (*ComicInfo, error) {
	comicURL := fmt.Sprintf("https://tw.manhuagui.com/comic/%s/", comicID)

	info := &ComicInfo{
		ID:       comicID,
		Chapters: make([]Chapter, 0),
	}

	err := chromedp.Run(c.ctx,
		chromedp.Navigate(comicURL),
		chromedp.WaitVisible(`.book-title`),
		c.fillComicInfo(info),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get comic info: %w", err)
	}

	return info, nil
}

func (info *ComicInfo) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(jsonData), nil
}

func (info *ComicInfo) ToPlainText() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Comic ID: %s\n", info.ID))
	sb.WriteString(fmt.Sprintf("Title: %s\n", info.Title))
	if info.Author != "" {
		sb.WriteString(fmt.Sprintf("Author: %s\n", info.Author))
	}
	if info.Status != "" {
		sb.WriteString(fmt.Sprintf("Status: %s\n", info.Status))
	}
	if info.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", info.Description))
	}
	sb.WriteString(fmt.Sprintf("Chapters: %d\n", len(info.Chapters)))
	sb.WriteString("\nChapter List:\n")

	for i, chapter := range info.Chapters {
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, chapter.ID, chapter.Title))
	}

	return sb.String()
}

func (c *ComicInfoFetcher) SearchComics(keyword string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://tw.manhuagui.com/s/%s.html", keyword)

	var results []SearchResult

	err := chromedp.Run(c.ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`.book-result`),
		c.fillSearchResults(&results),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to search comics: %w", err)
	}

	return results, nil
}

// fillSearchResults fills the search results slice by scraping the page.
func (c *ComicInfoFetcher) fillSearchResults(results *[]SearchResult) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var err error

		var searchData []map[string]string
		if e := evalJS(ctx, `Array.from(document.querySelectorAll('.book-result .book-detail dt a')).map(link => ({href: link.getAttribute('href'), title: link.textContent.trim(),}))`, &searchData); e != nil {
			err = multierr.Append(err, fmt.Errorf("get search results: %w", e))
		} else {
			for _, data := range searchData {
				link := data["href"]
				title := data["title"]

				// Extract comic ID from URL
				re := regexp.MustCompile(`/comic/(\d+)/`)
				matches := re.FindStringSubmatch(link)
				comicID := ""
				if len(matches) > 1 {
					comicID = matches[1]
				}

				if comicID != "" {
					result := SearchResult{
						ID:    comicID,
						Title: title,
						URL:   link,
					}
					*results = append(*results, result)
				}
			}
		}

		return err
	})
}
