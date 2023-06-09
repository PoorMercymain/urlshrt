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

func (s *url) ReadOriginal(ctx context.Context, shortened string) (string, error) {
	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
	}

	for _, url := range *curURLsPtr.Urls {
		if url.ShortURL == shortened {
			return url.OriginalURL, nil
		}
	}
	return "", errors.New("no such value")
}

func (s *url) CreateShortened(ctx context.Context, original string) (string, error) {
	var random *rand.Rand
	if rSeed := ctx.Value("seed"); rSeed != nil {
		random = rand.New(rand.NewSource(ctx.Value("seed").(int64)))
	} else {
		util.GetLogger().Infoln("seed not found in context, default value used")
		random = rand.New(rand.NewSource(time.Now().Unix()))
	}

	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
	}

	for _, url := range *curURLsPtr.Urls {
		if original == url.OriginalURL {
			return url.ShortURL, nil
		}
	}

	var shortenedURL string

	const shrtURLReqLen = 7

	shortenedURL = util.GenerateRandomString(shrtURLReqLen, random)

	for _, url := range *curURLsPtr.Urls {
		for shortenedURL == url.ShortURL {
			shortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
		}
	}

	createdURLStruct := state.URLStringJSON{UUID: len(*curURLsPtr.Urls), ShortURL: shortenedURL, OriginalURL: original}
	curURLsPtr.Lock()
	*curURLsPtr.Urls = append(*curURLsPtr.Urls, createdURLStruct)
	curURLsPtr.Unlock()

	s.repo.Create(ctx, []state.URLStringJSON{createdURLStruct})

	return shortenedURL, nil
}
