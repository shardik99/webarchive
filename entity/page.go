package entity

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Processor interface {
	Process(ctx context.Context, format Format, page *Page) Result
	GetMeta(ctx context.Context, page *Page) (Meta, error)
}

type Format uint8

const (
	FormatHeaders Format = iota
	FormatSingleFile
	FormatPDF
	FormatHTML
)

var AllFormats = []Format{
	FormatHeaders,
	FormatPDF,
	FormatSingleFile,
	FormatHTML,
}

type Status uint8

const (
	StatusNew Status = iota
	StatusProcessing
	StatusDone
	StatusFailed
	StatusWithErrors
)

type Meta struct {
	Title       string
	Description string
	Encoding    string
	Error       string
}

type PageBase struct {
	ID          uuid.UUID
	URL         string
	Description string
	Created     time.Time
	Formats     []Format
	Version     uint16
	Status      Status
	Meta        Meta
	Owner       string
	Tags        []string
	Headers     map[string]string
	Cookies     map[string]string
}

func NewPage(url string, description string, tags []string, headers map[string]string, cookies map[string]string, formats ...Format) *Page {
	normalizedTags := make([]string, 0, len(tags))
	for _, t := range tags {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			normalizedTags = append(normalizedTags, strings.ToLower(trimmed))
		}
	}

	return &Page{
		PageBase: PageBase{
			ID:          uuid.New(),
			URL:         url,
			Description: description,
			Formats:     formats,
			Created:     time.Now(),
			Version:     1,
			Tags:        normalizedTags,
			Headers:     headers,
			Cookies:     cookies,
		},
		cache: NewCache(),
	}
}

type Page struct {
	PageBase
	Results ResultsRO
	cache   *Cache
}

func (p *Page) Cache() *Cache {
	return p.cache
}

func (p *Page) SetProcessing() {
	p.Status = StatusProcessing
}

func (p *Page) Prepare(ctx context.Context, processor Processor) {
	meta, err := processor.GetMeta(ctx, p)
	if err != nil {
		p.Meta.Error = err.Error()
	} else {
		p.Meta = meta
	}
}

func (p *Page) Process(ctx context.Context, processor Processor) {
	innerWG := sync.WaitGroup{}
	innerWG.Add(len(p.Formats))

	results := Results{}

	for _, format := range p.Formats {
		go func(format Format) {
			defer innerWG.Done()

			defer func() {
				if err := recover(); err != nil {
					results.Add(Result{Format: format, Err: fmt.Errorf("recovered from panic: %v (%s)", err, string(debug.Stack()))})
				}
			}()

			result := processor.Process(ctx, format, p)
			results.Add(result)
		}(format)
	}

	innerWG.Wait()

	var hasResultWithOutErrors bool
	for _, result := range results.Results() {
		if result.Err != nil {
			p.Status = StatusWithErrors
		} else {
			hasResultWithOutErrors = true
		}
	}

	if !hasResultWithOutErrors {
		p.Status = StatusFailed
	}

	if p.Status == StatusProcessing {
		p.Status = StatusDone
	}

	p.Results = results.RO()
}
