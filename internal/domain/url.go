package domain

import (
	"context"
	"sync"

	"github.com/PoorMercymain/urlshrt/internal/state"
)

// URLService is an interface which defines what functions does an object which will operate on service layer should implement.
//go:generate mockgen -destination=mocks/srv_mock.gen.go -package=mocks . URLService
type URLService interface {
	ReadOriginal(ctx context.Context, shortened string, errChan chan error) (string, error)
	CreateShortened(ctx context.Context, original string) (string, error)
	CreateShortenedFromBatch(ctx context.Context, batch []*BatchElement) ([]BatchElementResult, error)
	PingPg(ctx context.Context) error
	ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error)
	DeleteUserURLs(ctx context.Context, short []URLWithID, shortURLsChan *MutexChanString, once *sync.Once)
}

// URLRepository is an interface which defines what functions does an object which will operate on repository layer should implement.
//go:generate mockgen -destination=mocks/repo_mock.gen.go -package=mocks . URLRepository
type URLRepository interface {
	ReadAll(ctx context.Context) ([]state.URLStringJSON, error)
	Create(ctx context.Context, urls []state.URLStringJSON) (string, error)
	CreateBatch(ctx context.Context, batch []*state.URLStringJSON) error
	PingPg(ctx context.Context) error
	ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error)
	DeleteUserURLs(ctx context.Context, shortURLs []string, uid []int64) error
	IsURLDeleted(ctx context.Context, shortened string) (bool, error)
}
