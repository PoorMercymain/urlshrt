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

func (s *url) PingPg(ctx context.Context) error {
	err := s.repo.PingPg(ctx)
	return err
}

func (s *url) CreateShortenedFromBatch(ctx context.Context, batch []*domain.BatchElement) ([]domain.BatchElementResult, error) {
	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return nil, err
	}

	var random *rand.Rand
	if rSeed := ctx.Value("seed"); rSeed != nil {
		random = rand.New(rand.NewSource(ctx.Value("seed").(int64)))
	} else {
		util.GetLogger().Infoln("seed not found in context, default value used")
		random = rand.New(rand.NewSource(time.Now().Unix()))
	}

	const shrtURLReqLen = 7

	notYetWritten := make([]*state.URLStringJSON, 0)

	var counter int
	util.GetLogger().Infoln("its them", *curURLsPtr.Urls, "len", len(*curURLsPtr.Urls))

	for j, batchURL := range batch {
		if len(*curURLsPtr.Urls) == 0 {
			batch[j].ShortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
			notYetWritten = append(notYetWritten, &(state.URLStringJSON{UUID: len(*curURLsPtr.Urls)+len(batch)-counter, ShortURL: batch[j].ShortenedURL, OriginalURL: batchURL.OriginalURL}))
			continue
		}

		if foundURL, ok := (*curURLsPtr.Urls)[batchURL.OriginalURL]; ok {
			batch[j].ShortenedURL = foundURL.ShortURL
		} else {
			batch[j].ShortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
			notYetWritten = append(notYetWritten, &(state.URLStringJSON{
				UUID: len(*curURLsPtr.Urls)+len(batch)-counter,
				ShortURL: batch[j].ShortenedURL,
				OriginalURL: batchURL.OriginalURL,
			}))
		}
	}



	util.GetLogger().Infoln("res", batch)
	batchToReturn := make([]domain.BatchElementResult, 0)
	for _, res := range batch {
		batchToReturn = append(batchToReturn, domain.BatchElementResult{ID: res.ID, ShortenedURL: res.ShortenedURL})
	}
	if counter == len(batch) {
		return batchToReturn, nil
	}

	util.GetLogger().Infoln("not written", notYetWritten)
	err = s.repo.CreateBatch(ctx, notYetWritten)
	if err != nil {
		return nil, err
	}

	curURLsPtr.Lock()
	for _, url := range notYetWritten {
		(*curURLsPtr.Urls)[url.OriginalURL] = *url
	}

	curURLsPtr.Unlock()

	return batchToReturn, nil
}

func (s *url) ReadOriginal(ctx context.Context, shortened string) (string, error) {
	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
	}

	for _, url := range *curURLsPtr.Urls {
		util.GetLogger().Infoln(url.ShortURL)
		if url.ShortURL == shortened {
			return url.OriginalURL, nil
		}
	}
	return "", errors.New("no such value")
}

func (s *url) CreateShortened(ctx context.Context, original string) (string, error) {
	var random *rand.Rand
	if rSeed := ctx.Value(domain.Key("seed")); rSeed != nil {
		util.GetLogger().Infoln(rSeed)
		random = rand.New(rand.NewSource(rSeed.(int64)))
	} else {
		util.GetLogger().Infoln("seed not found in context, default value used")
		random = rand.New(rand.NewSource(time.Now().Unix()))
	}

	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
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

	shrt, err := s.repo.Create(ctx, []state.URLStringJSON{createdURLStruct})
	if err != nil {
		return shrt, err
	}

	curURLsPtr.Lock()
	if _, ok := (*curURLsPtr.Urls)[createdURLStruct.OriginalURL]; !ok {
		(*curURLsPtr.Urls)[createdURLStruct.OriginalURL] = createdURLStruct
	} else {
		shortenedURL = (*curURLsPtr.Urls)[createdURLStruct.OriginalURL].ShortURL
	}

	curURLsPtr.Unlock()
	util.GetLogger().Infoln(curURLsPtr.Urls)

	return shortenedURL, nil
}
