package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type ComicsDL struct {
	url    string
	urlMap map[string]network.RequestID
	ctx    context.Context
	Pages  []string
}

func NewDownload(ctx context.Context, id1, id2 string) (*ComicsDL, error) {
	baseUrl := fmt.Sprintf("https://tw.manhuagui.com/comic/%s/%s.html", id1, id2)
	dl := &ComicsDL{
		baseUrl,
		make(map[string]network.RequestID),
		ctx,
		make([]string, 0),
	}

	//setup listeners
	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			unEscaped, err := url.PathUnescape(ev.Request.URL)
			dl.urlMap[ev.Request.URL] = ev.RequestID

			if err == nil {
				dl.urlMap[unEscaped] = ev.RequestID
			}
		}
	})

	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseUrl),
		chromedp.WaitVisible(`#mangaBox`),
	); err != nil {
		return nil, err
	}

	if err := dl.GetPages(); err != nil {
		return nil, err
	}

	return dl, nil
}

func (dl *ComicsDL) GetPages() error {
	var nodes []*cdp.Node
	if err := chromedp.Run(dl.ctx,
		chromedp.Nodes("#pageSelect", &nodes),
		chromedp.ActionFunc(func(ctx context.Context) error {
			dom.RequestChildNodes(nodes[0].NodeID).WithDepth(1).Do(ctx)
			for _, n := range nodes[0].Children {
				if page, existed := n.Attribute("value"); existed {
					dl.Pages = append(dl.Pages, page)
				}
			}
			return nil
		}),
	); err != nil {
		return err
	}
	return nil
}

func (dl *ComicsDL) findRequestID(src string) (network.RequestID, error) {
	if v, b := dl.urlMap[src]; b {
		return v, nil
	}

	if unEscaped, e := url.PathUnescape(src); e == nil {
		if v, b := dl.urlMap[unEscaped]; b {
			return v, nil
		}
	}

	return "", errors.New("no such url: " + src)
}

func (dl *ComicsDL) DownloadPageTo(pageNo string, writer io.Writer) error {
	var src string
	var b bool
	return chromedp.Run(dl.ctx,
		chromedp.Navigate(fmt.Sprintf(`%s#p=%s`, dl.url, pageNo)),
		chromedp.Reload(),
		chromedp.WaitVisible(`#mangaFile`),
		chromedp.AttributeValue(`#mangaFile`, "src", &src, &b),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if b {
				if v, err := dl.findRequestID(src); err == nil {
					log.Println(v)

					data, err := network.GetResponseBody(v).Do(ctx)
					if err == nil {
						if _, err := writer.Write(data); err != nil {
							return err
						}
					} else {
						return err
					}
				} else {
					return err
				}
			} else {
				return errors.New("no such image")
			}
			return nil
		}),
	)
}
