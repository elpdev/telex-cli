package api

import (
	"encoding/json"
	"fmt"
)

type Envelope[T any] struct {
	Data    T                   `json:"data"`
	Meta    map[string]any      `json:"meta,omitempty"`
	Error   string              `json:"error,omitempty"`
	Details map[string][]string `json:"details,omitempty"`
}

type Pagination struct {
	Page       int
	PerPage    int
	TotalCount int
}

func DecodeEnvelope[T any](body []byte) (*Envelope[T], error) {
	var envelope Envelope[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode response envelope: %w", err)
	}
	return &envelope, nil
}

func DecodePagination(meta map[string]any) *Pagination {
	if len(meta) == 0 {
		return nil
	}
	page, okPage := intFromMeta(meta["page"])
	perPage, okPerPage := intFromMeta(meta["per_page"])
	totalCount, okTotalCount := intFromMeta(meta["total_count"])
	if !okPage || !okPerPage || !okTotalCount {
		return nil
	}
	return &Pagination{Page: page, PerPage: perPage, TotalCount: totalCount}
}

func intFromMeta(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}
