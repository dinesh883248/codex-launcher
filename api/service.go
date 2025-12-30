package api

import "context"

type Service struct {
	store *Store
}

type Page struct {
	Requests []Request
	Page     int
	Pages    int
	Total    int
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateRequest(ctx context.Context, prompt string) (Request, error) {
	return s.store.CreateRequest(ctx, prompt)
}

func (s *Service) ListRequests(ctx context.Context, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	items, total, err := s.store.ListRequests(ctx, offset, pageSize)
	if err != nil {
		return Page{}, err
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	return Page{Requests: items, Page: page, Pages: pages, Total: total}, nil
}

func (s *Service) GetProcessingRequest(ctx context.Context) (Request, bool, error) {
	return s.store.GetProcessingRequest(ctx)
}
