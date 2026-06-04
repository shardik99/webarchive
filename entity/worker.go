package entity

import (
	"context"
	"net/url"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/net/html"
)

type Pages interface {
	Save(ctx context.Context, page *Page) error
	ListUnprocessed(ctx context.Context) ([]Page, error)
	ListAll(ctx context.Context, owner string, tags []string) ([]*Page, error)
}

func NewWorker(ch chan *Page, pages Pages, processor Processor, log *zap.Logger) *Worker {
	return &Worker{pages: pages, processor: processor, log: log, ch: ch}
}

type Worker struct {
	ch        chan *Page
	pages     Pages
	processor Processor
	log       *zap.Logger
}

func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.log.Info("starting")

	wg.Add(1)
	go func() {
		defer wg.Done()

		unprocessed, err := w.pages.ListUnprocessed(ctx)
		if err != nil {
			w.log.Error("failed to get unprocessed pages", zap.Error(err))
		} else {
			for i := range unprocessed {
				w.ch <- &unprocessed[i]
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case page, open := <-w.ch:
			if !open {
				w.log.Warn("channel closed")
				return
			}

			log := w.log.With(zap.Stringer("page_id", page.ID), zap.String("page_url", page.URL))

			log.Info("got new page")

			wg.Add(1)
			go w.do(ctx, wg, page, log)
		}
	}
}

func (w *Worker) do(ctx context.Context, wg *sync.WaitGroup, page *Page, log *zap.Logger) {
	defer wg.Done()

	page.SetProcessing()
	if err := w.pages.Save(ctx, page); err != nil {
		w.log.Error(
			"failed to save processing page",
			zap.String("page_id", page.ID.String()),
			zap.String("page_url", page.URL),
			zap.Error(err),
		)
	}

	page.Process(ctx, w.processor)

	log.Debug("page processed")

	if page.Depth > 0 {
		links := w.extractLinks(ctx, page)
		if len(links) > 0 {
			existing, err := w.pages.ListAll(ctx, page.Owner, nil)
			existMap := make(map[string]struct{})
			if err == nil {
				for _, p := range existing {
					if p.CollectionID != nil && page.CollectionID != nil && *p.CollectionID == *page.CollectionID {
						existMap[p.URL] = struct{}{}
					} else if p.CollectionID == nil && page.CollectionID == nil {
						existMap[p.URL] = struct{}{}
					}
				}
			}

			for _, l := range links {
				if _, ok := existMap[l]; ok {
					continue
				}

				child := NewPage(l, page.Description, page.Tags, page.Headers, page.Cookies, page.CollectionID, page.Depth-1, page.Formats...)
				child.Owner = page.Owner
				child.Status = StatusNew
				child.Prepare(ctx, w.processor)

				if err := w.pages.Save(ctx, child); err == nil {
					go func(c *Page) {
						w.ch <- c
					}(child)
				}
			}
		}
	}

	if err := w.pages.Save(ctx, page); err != nil {
		w.log.Error(
			"failed to save processed page",
			zap.String("page_id", page.ID.String()),
			zap.String("page_url", page.URL),
			zap.Error(err),
		)
	}
}

func (w *Worker) extractLinks(_ context.Context, page *Page) []string {
	if page.Depth <= 0 || page.Cache().Reader() == nil {
		return nil
	}

	pageURL, err := url.Parse(page.URL)
	if err != nil {
		w.log.Error("failed to parse page url", zap.Error(err))
		return nil
	}

	doc, err := html.Parse(page.Cache().Reader())
	if err != nil {
		w.log.Error("failed to parse html", zap.Error(err))
		return nil
	}

	links := make(map[string]struct{})
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, err := url.Parse(a.Val)
					if err == nil {
						resolved := pageURL.ResolveReference(u)
						if resolved.Host == pageURL.Host && (resolved.Scheme == "http" || resolved.Scheme == "https") {
							resolved.Fragment = ""
							links[resolved.String()] = struct{}{}
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var result []string
	for l := range links {
		result = append(result, l)
	}
	return result
}
