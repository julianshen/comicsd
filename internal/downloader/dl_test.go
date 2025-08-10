package downloader_test

import (
	"archive/zip"
	"comicsd/internal/downloader"
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/chromedp/chromedp"
)

func TestGetPages(t *testing.T) {
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	file, err := os.Create("東大特訓班2-7.cbz")
	if err != nil {
		panic(err)
	}
	cbz := zip.NewWriter(file)
	defer cbz.Close()
	cc, err := downloader.NewDownload(ctx, "26964", "718179")
	if err != nil {
		t.Fatal(err)
	}
	for n := range cc.Pages {
		log.Printf("Downloading page %d\n", n)
		w, err := cbz.Create(fmt.Sprintf("%d.jpg", n))
		if err != nil {
			panic(err)
		}
		err = cc.DownloadPageTo(cc.Pages[n], w)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("DONE")
}
