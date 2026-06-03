package processors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/net/html"

	"github.com/derfenix/webarchive/entity"
)

func NewHTML(client *http.Client, log *zap.Logger) *HTMLProcessor {
	return &HTMLProcessor{client: client, log: log}
}

type HTMLProcessor struct {
	client *http.Client
	log    *zap.Logger
}

func (h *HTMLProcessor) Process(ctx context.Context, page *entity.Page) ([]entity.File, error) {
	reader := page.Cache().Reader()

	if reader == nil {
		response, err := h.get(ctx, page, page.URL)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		reader = response.Body
	}

	htmlNode, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	baseURL, err := url.Parse(page.URL)
	if err != nil {
		return nil, fmt.Errorf("parse page url: %w", err)
	}

	var files []entity.File

	h.visit(ctx, htmlNode, baseURL, page, &files)

	buf := bytes.NewBuffer(nil)
	if err := html.Render(buf, htmlNode); err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	files = append(files, entity.NewFile("page.html", buf.Bytes()))

	return files, nil
}

func (h *HTMLProcessor) visit(ctx context.Context, n *html.Node, baseURL *url.URL, page *entity.Page, files *[]entity.File) {
	if err := h.processNode(ctx, n, baseURL, page, files); err != nil {
		h.log.Error("process node error", zap.Error(err))
	}

	if n.FirstChild != nil {
		h.visit(ctx, n.FirstChild, baseURL, page, files)
	}

	if n.NextSibling != nil {
		h.visit(ctx, n.NextSibling, baseURL, page, files)
	}
}

func (h *HTMLProcessor) processNode(ctx context.Context, node *html.Node, baseURL *url.URL, page *entity.Page, files *[]entity.File) error {
	var targetAttr string
	switch node.Data {
	case "link":
		targetAttr = "href"
		shouldProcess := false
		for _, attr := range node.Attr {
			if attr.Key == "rel" {
				switch attr.Val {
				case "stylesheet", "icon", "alternate icon", "shortcut icon", "manifest":
					shouldProcess = true
				}
			}
		}
		if !shouldProcess {
			return nil
		}
	case "script", "img":
		targetAttr = "src"
	default:
		return nil
	}

	var resourceURL string
	var attrIdx = -1

	for i, attr := range node.Attr {
		if attr.Key == targetAttr {
			resourceURL = attr.Val
			attrIdx = i
			break
		} else if attr.Key == "data-src" {
			resourceURL = attr.Val
			attrIdx = i
		}
	}

	if resourceURL == "" || attrIdx == -1 {
		return nil
	}

	normalized := normalizeHTMLURL(resourceURL, baseURL)
	if normalized == "" {
		return nil
	}

	resp, err := h.get(ctx, page, normalized)
	if err != nil {
		return fmt.Errorf("get resource %s: %w", normalized, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read resource: %w", err)
	}

	ext := "bin"
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "image/jpeg") {
		ext = "jpg"
	} else if strings.Contains(ct, "image/png") {
		ext = "png"
	} else if strings.Contains(ct, "image/gif") {
		ext = "gif"
	} else if strings.Contains(ct, "text/css") {
		ext = "css"
	} else if strings.Contains(ct, "javascript") {
		ext = "js"
	} else if strings.Contains(ct, "image/svg+xml") {
		ext = "svg"
	} else if strings.Contains(ct, "image/webp") {
		ext = "webp"
	} else if strings.Contains(ct, "font/woff2") {
		ext = "woff2"
	} else if strings.Contains(ct, "font/woff") {
		ext = "woff"
	} else if strings.Contains(ct, "font/ttf") {
		ext = "ttf"
	}

	fileName := fmt.Sprintf("%s.%s", uuid.New().String()[:8], ext)
	node.Attr[attrIdx].Val = fileName

	*files = append(*files, entity.NewFile(fileName, data))

	return nil
}

func normalizeHTMLURL(resourceURL string, base *url.URL) string {
	if strings.HasPrefix(resourceURL, "//") {
		return "https:" + resourceURL
	}
	if strings.HasPrefix(resourceURL, "about:") {
		return ""
	}
	if strings.HasPrefix(resourceURL, "data:") {
		return ""
	}
	parsed, err := url.Parse(resourceURL)
	if err != nil {
		return resourceURL
	}
	return base.ResolveReference(parsed).String()
}

func (h *HTMLProcessor) get(ctx context.Context, page *entity.Page, url string) (*http.Response, error) {
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
	response, err := h.client.Do(req)
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
