package processors

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/html"

	"github.com/derfenix/webarchive/config"
	"github.com/derfenix/webarchive/entity"
)

const defaultEncoding = "utf-8"

type processor interface {
	Process(ctx context.Context, page *entity.Page) ([]entity.File, error)
}

func NewProcessors(cfg config.Config, log *zap.Logger) (*Processors, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Second * 10,
				KeepAlive: time.Second * 10,
			}).DialContext,
			MaxIdleConns:           20,
			MaxIdleConnsPerHost:    5,
			MaxConnsPerHost:        10,
			IdleConnTimeout:        time.Second * 60,
			ResponseHeaderTimeout:  time.Second * 20,
			MaxResponseHeaderBytes: 1024 * 1024 * 50,
			WriteBufferSize:        256,
			ReadBufferSize:         1024 * 64,
			ForceAttemptHTTP2:      true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 3 {
				return fmt.Errorf("too many redirects")
			}

			return nil
		},
		Jar:     jar,
		Timeout: time.Second * 30,
	}

	procs := Processors{
		client: httpClient,
		processors: map[entity.Format]processor{
			entity.FormatHeaders:    NewHeaders(httpClient),
			entity.FormatPDF:        NewPDF(cfg.PDF),
			entity.FormatSingleFile: NewSingleFile(httpClient, log),
			entity.FormatHTML:       NewHTML(httpClient, log),
			entity.FormatMarkdown:   NewMarkdown(httpClient, log),
		},
	}

	return &procs, nil
}

type Processors struct {
	processors map[entity.Format]processor
	client     *http.Client
}

func (p *Processors) Process(ctx context.Context, format entity.Format, page *entity.Page) entity.Result {
	result := entity.Result{Format: format}

	proc, ok := p.processors[format]
	if !ok {
		result.Err = fmt.Errorf("no processor registered")

		return result
	}

	files, err := proc.Process(ctx, page)
	if err != nil {
		result.Err = fmt.Errorf("process: %w", err)

		return result
	}

	result.Files = files

	return result
}

func (p *Processors) OverrideProcessor(format entity.Format, proc processor) error {
	p.processors[format] = proc

	return nil
}

func (p *Processors) GetMeta(ctx context.Context, page *entity.Page) (entity.Meta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, page.URL, nil)
	if err != nil {
		return entity.Meta{}, fmt.Errorf("new request: %w", err)
	}

	for k, v := range page.Headers {
		req.Header.Add(k, v)
	}
	for k, v := range page.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	response, err := p.client.Do(req)
	if err != nil {
		return entity.Meta{}, fmt.Errorf("do request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return entity.Meta{}, fmt.Errorf("want status 200, got %d", response.StatusCode)
	}

	if response.Body == nil {
		return entity.Meta{}, fmt.Errorf("empty response body")
	}

	defer func() {
		_ = response.Body.Close()
	}()

	tee := io.TeeReader(response.Body, page.Cache())

	htmlNode, err := html.Parse(tee)
	if err != nil {
		return entity.Meta{}, fmt.Errorf("parse response body: %w", err)
	}

	var fc *html.Node
	for fc = htmlNode.FirstChild; fc != nil && fc.Data != "html"; fc = fc.NextSibling {
	}

	if fc == nil {
		return entity.Meta{}, fmt.Errorf("failed to find html tag")
	}

	fc = fc.NextSibling
	if fc == nil {
		return entity.Meta{}, fmt.Errorf("failed to find html tag")
	}

	for fc = fc.FirstChild; fc != nil && fc.Data != "head"; fc = fc.NextSibling {
		fmt.Println(fc.Data)
	}

	if fc == nil {
		return entity.Meta{}, fmt.Errorf("failed to find html tag")
	}

	meta := entity.Meta{}
	getMetaData(fc, &meta)
	meta.Encoding = encodingFromHeader(response.Header)

	return meta, nil
}

func getMetaData(n *html.Node, meta *entity.Meta) {
	if n == nil {
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "title" {
			meta.Title = c.FirstChild.Data
		}
		if c.Type == html.ElementNode && c.Data == "meta" {
			attrs := make(map[string]string)
			for _, attr := range c.Attr {
				attrs[attr.Key] = attr.Val
			}

			name, ok := attrs["name"]
			if ok && name == "description" {
				meta.Description = attrs["content"]
			}
		}

		getMetaData(c, meta)
	}
}

func encodingFromHeader(headers http.Header) string {
	var foundEncoding bool
	var encoding string

	_, encoding, foundEncoding = strings.Cut(headers.Get("Content-Type"), "; ")
	if foundEncoding {
		_, encoding, foundEncoding = strings.Cut(encoding, "=")
	}

	if !foundEncoding {
		encoding = defaultEncoding
	}

	return encoding
}
