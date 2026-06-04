package processors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/derfenix/webarchive/entity"
	"github.com/go-shiori/go-readability"
	"go.uber.org/zap"
)

func NewMarkdown(client *http.Client, log *zap.Logger) *MarkdownProcessor {
	return &MarkdownProcessor{client: client, log: log}
}

type MarkdownProcessor struct {
	client *http.Client
	log    *zap.Logger
}

func (m *MarkdownProcessor) Process(ctx context.Context, page *entity.Page) ([]entity.File, error) {
	var reader io.Reader
	if len(page.InlinedHTML) > 0 {
		reader = bytes.NewReader(page.InlinedHTML)
	} else {
		reader = page.Cache().Reader()
		if reader == nil {
			response, err := m.get(ctx, page, page.URL)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			reader = response.Body
		}
	}

	pageURL, err := url.Parse(page.URL)
	if err != nil {
		return nil, fmt.Errorf("parse page url: %w", err)
	}

	article, err := readability.FromReader(reader, pageURL)
	if err != nil {
		return nil, fmt.Errorf("readability extraction: %w", err)
	}

	markdown, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		return nil, fmt.Errorf("html to markdown conversion: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("# ")
	buf.WriteString(article.Title)
	buf.WriteString("\n\n")
	buf.WriteString(markdown)

	file := entity.NewFile("article.md", buf.Bytes())
	return []entity.File{file}, nil
}

func (m *MarkdownProcessor) get(ctx context.Context, page *entity.Page, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	for k, v := range page.Headers {
		req.Header.Add(k, v)
	}
	for k, v := range page.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	response, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("want status 200, got %d", response.StatusCode)
	}
	if response.Body == nil {
		return nil, fmt.Errorf("empty response body")
	}
	return response, nil
}
