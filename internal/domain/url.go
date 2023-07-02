package domain

import (
	"context"
	"sync"

	"github.com/PoorMercymain/urlshrt/internal/state"
)

type URLService interface {
	ReadOriginal(ctx context.Context, shortened string) (string, error)
	CreateShortened(ctx context.Context, original string) (string, error)
	CreateShortenedFromBatch(ctx context.Context, batch []*BatchElement) ([]BatchElementResult, error)
	PingPg(ctx context.Context) error
	ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error)
	DeleteUserURLs(ctx context.Context, short []URLWithID, shortURLsChan *MutexChanString, once *sync.Once)
}

type URLRepository interface {
	ReadAll(ctx context.Context) ([]state.URLStringJSON, error)
	Create(ctx context.Context, urls []state.URLStringJSON) (string, error)
	CreateBatch(ctx context.Context, batch []*state.URLStringJSON) error
	PingPg(ctx context.Context) error
	ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error)
	DeleteUserURLs(ctx context.Context, shortURLs []string, uid []int64) error
	IsURLDeleted(ctx context.Context, shortened string) (bool, error)
}
