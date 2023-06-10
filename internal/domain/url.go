package domain

import (
	"context"

	"github.com/PoorMercymain/urlshrt/internal/state"
)

type URLService interface {
	ReadOriginal(ctx context.Context, shortened string) (string, error)
	CreateShortened(ctx context.Context, original string) (string, error)
	PingPg(ctx context.Context) error
}

type URLRepository interface {
	ReadAll(ctx context.Context) ([]state.URLStringJSON, error)
	Create(ctx context.Context, urls []state.URLStringJSON) error
	PingPg(ctx context.Context) error
}
