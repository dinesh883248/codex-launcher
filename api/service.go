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
		offset = (page - 1) * pageSize
		items, _, err = s.store.ListRequests(ctx, offset, pageSize)
		if err != nil {
			return Page{}, err
		}
	}
	return Page{Requests: items, Page: page, Pages: pages, Total: total}, nil
}

func (s *Service) GetProcessingRequest(ctx context.Context) (Request, bool, error) {
	return s.store.GetProcessingRequest(ctx)
}

func (s *Service) GetRequest(ctx context.Context, id int64) (Request, bool, error) {
	return s.store.GetRequest(ctx, id)
}

func (s *Service) GetOutputLines(ctx context.Context, requestID int64, limit, offset int) ([]OutputLine, int, error) {
	return s.store.GetOutputLines(ctx, requestID, limit, offset)
}
