package info

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFillComicInfoMissingElements(t *testing.T) {
	origText := textContent
	origEval := evalJS
	defer func() { textContent = origText; evalJS = origEval }()

	textErrors := map[string]error{
		`.book-title h1`:            errors.New("title missing"),
		`.book-detail .detail-list`: errors.New("detail missing"),
	}
	textContent = func(ctx context.Context, sel string, res *string) error {
		if err, ok := textErrors[sel]; ok {
			return err
		}
		return nil
	}
	evalJS = func(ctx context.Context, expr string, res interface{}) error {
		return errors.New("chapter missing")
	}

	info := &ComicInfo{ID: "1"}
	fetcher := &ComicInfoFetcher{}
	err := fetcher.fillComicInfo(info).Do(context.Background())
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "title missing") || !strings.Contains(msg, "detail missing") || !strings.Contains(msg, "chapter missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFillSearchResultsMissingElements(t *testing.T) {
	origEval := evalJS
	defer func() { evalJS = origEval }()

	evalJS = func(ctx context.Context, expr string, res interface{}) error {
		return errors.New("search missing")
	}

	var results []SearchResult
	fetcher := &ComicInfoFetcher{}
	err := fetcher.fillSearchResults(&results).Do(context.Background())
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "search missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}
