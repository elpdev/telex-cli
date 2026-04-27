package api

import (
	"context"
	"fmt"
	"net/url"
)

type Getter interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
}

func List[T any](client Getter, ctx context.Context, path string, query url.Values) ([]T, *Pagination, error) {
	body, _, err := client.Get(ctx, path, query)
	if err != nil {
		return nil, nil, err
	}
	envelope, err := DecodeEnvelope[[]T](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, DecodePagination(envelope.Meta), nil
}

func SetString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func SetInt(query url.Values, key string, value int) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func SetInt64(query url.Values, key string, value int64) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func SetBool(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, fmt.Sprintf("%t", *value))
	}
}
