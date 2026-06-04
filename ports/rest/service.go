package rest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/derfenix/webarchive/api/openapi"
	"github.com/derfenix/webarchive/entity"
)

type Pages interface {
	ListAll(ctx context.Context, owner string, tags []string) ([]*entity.Page, error)
	Save(ctx context.Context, site *entity.Page) error
	Get(ctx context.Context, id uuid.UUID) (*entity.Page, error)
	GetFile(ctx context.Context, pageID, fileID uuid.UUID) (*entity.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, page *entity.Page) error
}

type Collections interface {
	ListAll(ctx context.Context, owner string) ([]*entity.Collection, error)
	Save(ctx context.Context, col *entity.Collection) error
	Get(ctx context.Context, id uuid.UUID) (*entity.Collection, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewService(pages Pages, cols Collections, ch chan *entity.Page, processor entity.Processor) *Service {
	return &Service{
		pages:       pages,
		collections: cols,
		ch:          ch,
		processor:   processor,
	}
}

type Service struct {
	openapi.UnimplementedHandler
	processor   entity.Processor
	pages       Pages
	collections Collections
	ch          chan *entity.Page
}

func (s *Service) GetPage(ctx context.Context, params openapi.GetPageParams) (openapi.GetPageRes, error) {
	page, err := s.pages.Get(ctx, params.ID)
	if err != nil {
		return &openapi.GetPageNotFound{}, nil
	}

	if page.Owner != OwnerFromContext(ctx) {
		return &openapi.GetPageNotFound{}, nil
	}

	restPage := PageToRestWithResults(page)

	return &restPage, nil
}

func (s *Service) AddPage(ctx context.Context, req openapi.OptAddPageReq, params openapi.AddPageParams) (openapi.AddPageRes, error) {
	url := params.URL.Or(req.Value.URL)
	description := params.Description.Or(req.Value.Description.Value)

	formats := req.Value.Formats
	if len(formats) == 0 {
		formats = params.Formats
	}
	if len(formats) == 0 {
		formats = []openapi.Format{"all"}
	}

	tags := req.Value.Tags
	if len(tags) == 0 {
		tags = params.Tags
	}

	switch {
	case req.Value.URL != "":
		url = req.Value.URL
	case params.URL.IsSet():
		url = params.URL.Value
	}

	if url == "" {
		return &openapi.AddPageBadRequest{
			Field: "url",
			Error: "Value is required",
		}, nil
	}

	domainFormats, err := FormatFromRest(formats)
	if err != nil {
		return &openapi.AddPageBadRequest{
			Field: "formats",
			Error: err.Error(),
		}, nil
	}

	var headers, cookies map[string]string
	if req.Value.Headers.IsSet() {
		headers = req.Value.Headers.Value
	}
	if req.Value.Cookies.IsSet() {
		cookies = req.Value.Cookies.Value
	}

	var collectionID *uuid.UUID
	if req.Value.CollectionID.IsSet() && !req.Value.CollectionID.Null {
		cid := req.Value.CollectionID.Value
		collectionID = &cid
	}

	depth := 0
	if req.Value.Depth.IsSet() {
		depth = req.Value.Depth.Value
	}

	if depth > 0 && collectionID == nil {
		col := entity.NewCollection(url, "Auto-created collection for links from "+url, OwnerFromContext(ctx))
		if err := s.collections.Save(ctx, col); err != nil {
			return nil, fmt.Errorf("create auto collection: %w", err)
		}
		collectionID = &col.ID
	}

	var sublinkFormats []entity.Format
	if len(req.Value.SublinkFormats) > 0 {
		sf, err := FormatFromRest(req.Value.SublinkFormats)
		if err != nil {
			return &openapi.AddPageBadRequest{
				Field: "sublink_formats",
				Error: err.Error(),
			}, nil
		}
		sublinkFormats = sf
	}

	page := entity.NewPage(url, description, tags, headers, cookies, collectionID, depth, sublinkFormats, domainFormats...)
	page.Owner = OwnerFromContext(ctx)
	page.Status = entity.StatusNew
	page.Prepare(ctx, s.processor)

	if err := s.pages.Save(ctx, page); err != nil {
		return nil, fmt.Errorf("save page: %w", err)
	}

	res := BasePageToRest(&page.PageBase)

	s.ch <- page

	return &res, nil
}

func (s *Service) GetPages(ctx context.Context, params openapi.GetPagesParams) (openapi.Pages, error) {
	sites, err := s.pages.ListAll(ctx, OwnerFromContext(ctx), params.Tags)
	if err != nil {
		return nil, fmt.Errorf("list all: %w", err)
	}

	res := make(openapi.Pages, len(sites))
	for i := range res {
		res[i] = PageToRest(sites[i])
	}

	return res, nil
}

func (s *Service) GetFile(ctx context.Context, params openapi.GetFileParams) (openapi.GetFileRes, error) {
	page, err := s.pages.Get(ctx, params.ID)
	if err != nil || page.Owner != OwnerFromContext(ctx) {
		return &openapi.GetFileNotFound{}, nil
	}

	file, err := s.pages.GetFile(ctx, params.ID, params.FileID)
	if err != nil {
		return &openapi.GetFileNotFound{}, nil
	}

	switch {
	case file.MimeType == "application/pdf":
		return &openapi.GetFileOKApplicationPdf{Data: bytes.NewReader(file.Data)}, nil

	case strings.HasPrefix(file.MimeType, "text/plain"):
		return &openapi.GetFileOKTextPlainCharsetUtf8{Data: bytes.NewReader(file.Data)}, nil

	case strings.HasPrefix(file.MimeType, "text/html"):
		return &openapi.GetFileOKTextHTMLCharsetUtf8{Data: bytes.NewReader(file.Data)}, nil

	default:
		return nil, fmt.Errorf("unsupported mimetype: %s", file.MimeType)
	}
}

func (s *Service) DeletePage(ctx context.Context, params openapi.DeletePageParams) (openapi.DeletePageRes, error) {
	page, err := s.pages.Get(ctx, params.ID)
	if err != nil || page.Owner != OwnerFromContext(ctx) {
		return &openapi.DeletePageNotFound{}, nil
	}

	if err := s.pages.Delete(ctx, params.ID); err != nil {
		return nil, fmt.Errorf("delete page: %w", err)
	}

	return &openapi.DeletePageNoContent{}, nil
}

func (s *Service) UpdatePage(ctx context.Context, req openapi.OptUpdatePageReq, params openapi.UpdatePageParams) (openapi.UpdatePageRes, error) {
	page, err := s.pages.Get(ctx, params.ID)
	if err != nil || page.Owner != OwnerFromContext(ctx) {
		return &openapi.UpdatePageNotFound{}, nil
	}

	if req.Value.Title.IsSet() {
		page.Meta.Title = req.Value.Title.Value
	}
	if req.Value.Description.IsSet() {
		page.Meta.Description = req.Value.Description.Value
	}
	if req.Value.Tags != nil {
		page.Tags = req.Value.Tags
	}
	if req.Value.CollectionID.IsSet() {
		if req.Value.CollectionID.Null {
			page.CollectionID = nil
		} else {
			uid := req.Value.CollectionID.Value
			page.CollectionID = &uid
		}
	}

	if err := s.pages.Update(ctx, page); err != nil {
		return nil, fmt.Errorf("update page: %w", err)
	}

	restPage := PageToRest(page)
	return &restPage, nil
}

func (s *Service) AddCollection(ctx context.Context, req openapi.OptAddCollectionReq) (*openapi.Collection, error) {
	name := req.Value.Name
	desc := ""
	if req.Value.Description.IsSet() {
		desc = req.Value.Description.Value
	}

	col := entity.NewCollection(name, desc, OwnerFromContext(ctx))

	if err := s.collections.Save(ctx, col); err != nil {
		return nil, fmt.Errorf("save collection: %w", err)
	}

	res := CollectionToRest(col)
	return &res, nil
}

func (s *Service) GetCollections(ctx context.Context) ([]openapi.Collection, error) {
	cols, err := s.collections.ListAll(ctx, OwnerFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}

	res := make([]openapi.Collection, len(cols))
	for i := range res {
		res[i] = CollectionToRest(cols[i])
	}
	return res, nil
}

func (s *Service) GetCollection(ctx context.Context, params openapi.GetCollectionParams) (openapi.GetCollectionRes, error) {
	col, err := s.collections.Get(ctx, params.ID)
	if err != nil || col.Owner != OwnerFromContext(ctx) {
		return &openapi.GetCollectionNotFound{}, nil
	}

	res := CollectionToRest(col)
	return &res, nil
}

func (s *Service) DeleteCollection(ctx context.Context, params openapi.DeleteCollectionParams) (openapi.DeleteCollectionRes, error) {
	col, err := s.collections.Get(ctx, params.ID)
	if err != nil || col.Owner != OwnerFromContext(ctx) {
		return &openapi.DeleteCollectionNotFound{}, nil
	}

	if err := s.collections.Delete(ctx, params.ID); err != nil {
		return nil, fmt.Errorf("delete collection: %w", err)
	}

	return &openapi.DeleteCollectionNoContent{}, nil
}

func (s *Service) NewError(_ context.Context, err error) *openapi.ErrorStatusCode {
	return &openapi.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: openapi.Error{
			Message:   err.Error(),
			Localized: openapi.OptString{},
		},
	}
}
