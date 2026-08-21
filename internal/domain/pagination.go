package domain

import "fmt"

type ListQuery struct {
	Page    int               `form:"page"`
	PerPage int               `form:"per_page"`
	Sort    string            `form:"sort"`
	Filters map[string]string `form:"-"`
}

func (q *ListQuery) Normalize(allowedSorts, allowedFilters map[string]bool) error {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
	if q.Page < 1 || q.PerPage < 1 || q.PerPage > 100 {
		return fmt.Errorf("invalid pagination")
	}
	if q.Sort == "" {
		q.Sort = "updated_at:desc"
	}
	if !allowedSorts[q.Sort] {
		return fmt.Errorf("sort is not allowed")
	}
	for key := range q.Filters {
		if !allowedFilters[key] {
			return fmt.Errorf("filter %q is not allowed", key)
		}
	}
	return nil
}

type Page[T any] struct {
	Items   []T `json:"items"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}
