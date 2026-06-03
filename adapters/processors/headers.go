package processors

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/derfenix/webarchive/entity"
)

func NewHeaders(client *http.Client) *Headers {
	return &Headers{client: client}
}

type Headers struct {
	client *http.Client
}

func (h *Headers) Process(ctx context.Context, page *entity.Page) ([]entity.File, error) {
	var (
		headersFile entity.File
		err         error
	)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodHead, page.URL, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("create request: %w", reqErr)
	}

	for k, v := range page.Headers {
		req.Header.Add(k, v)
	}
	for k, v := range page.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	resp, doErr := h.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("call url: %w", doErr)
	}

	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	headersFile, err = h.newFile(resp.Header)

	if err != nil {
		return nil, fmt.Errorf("new file from headers: %w", err)
	}

	return []entity.File{headersFile}, nil
}

func (h *Headers) newFile(headers http.Header) (entity.File, error) {
	buf := bytes.NewBuffer(nil)

	if err := headers.Write(buf); err != nil {
		return entity.File{}, fmt.Errorf("write headers: %w", err)
	}

	return entity.NewFile("headers", buf.Bytes()), nil
}
