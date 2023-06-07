package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type url struct {
	repo domain.URLRepository
}

func NewURL(repo domain.URLRepository) *url {
	return &url{repo: repo}
}

func(s *url) ReadOriginal(ctx context.Context, shortened string) (string, error) {
	for _, url := range *state.GetCurrentURLsPtr().Urls {
		if url.ShortURL == shortened {
			return url.OriginalURL, nil
		}
	}
	return "", errors.New("no such value")
}

func(s *url) CreateShortened(ctx context.Context, original string) string {
	//TODO: seed should be an argument or be able to set from main in other way
	random := rand.New(rand.NewSource(time.Now().Unix()))

	for _, url := range *state.GetCurrentURLsPtr().Urls {
		if original == url.OriginalURL {
			return url.ShortURL
		}
	}

	var shortenedURL string

	shrtURLReqLen := 7

	shortenedURL = util.GenerateRandomString(shrtURLReqLen, random)

	for _, url := range *state.GetCurrentURLsPtr().Urls {
		for shortenedURL == url.ShortURL {
			shortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
		}
	}

	createdURLStruct := state.URLStringJSON{UUID: len(*state.GetCurrentURLsPtr().Urls), ShortURL: shortenedURL, OriginalURL: original}
	state.GetCurrentURLsPtr().Lock()
	*state.GetCurrentURLsPtr().Urls = append(*state.GetCurrentURLsPtr().Urls, createdURLStruct)
	state.GetCurrentURLsPtr().Unlock()


	s.repo.Create(ctx, []state.URLStringJSON{createdURLStruct})

	return shortenedURL
}