// service is a package which contains some functions to use on service layer of the app.
package service

import (
	"context"
	"errors"
	"math/rand"
	"sync"
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

func (s *url) ReadUserURLs(ctx context.Context) ([]state.URLStringJSON, error) {
	return s.repo.ReadUserURLs(ctx)
}

func (s *url) PingPg(ctx context.Context) error {
	return s.repo.PingPg(ctx)
}

// CreateShortenedFromBatch creates shorten URLs from batch elements and calls repository level to save it to database.
func (s *url) CreateShortenedFromBatch(ctx context.Context, batch []*domain.BatchElement, wg *sync.WaitGroup) ([]domain.BatchElementResult, error) {
	wg.Add(1)
	defer wg.Done()

	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return nil, err
	}

	var random *rand.Rand
	if rSeed := ctx.Value(domain.Key("seed")); rSeed != nil {
		random = rand.New(rand.NewSource(ctx.Value(domain.Key("seed")).(int64)))
	} else {
		util.GetLogger().Infoln("seed not found in context, default value used")
		random = rand.New(rand.NewSource(time.Now().Unix()))
	}

	const shrtURLReqLen = 7

	notYetWritten := make([]*state.URLStringJSON, 0)

	util.GetLogger().Infoln("its them", *curURLsPtr.Urls, "len", len(*curURLsPtr.Urls))
	allShortURLs := make(map[string]bool)
	for _, urlFromCurURLs := range *curURLsPtr.Urls {
		allShortURLs[urlFromCurURLs.ShortURL] = true
	}

	var uuidShift int
	for j, batchURL := range batch {
		if foundURL, ok := (*curURLsPtr.Urls)[batchURL.OriginalURL]; ok {
			batch[j].ShortenedURL = foundURL.ShortURL
		} else {
			uuidShift += 1
			for {
				batch[j].ShortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
				if _, shortExists := allShortURLs[batch[j].ShortenedURL]; !shortExists {
					notYetWritten = append(notYetWritten, &(state.URLStringJSON{
						UUID:        len(*curURLsPtr.Urls) + uuidShift,
						ShortURL:    batch[j].ShortenedURL,
						OriginalURL: batch[j].OriginalURL,
					}))
					allShortURLs[batch[j].ShortenedURL] = true
					break
				}
			}
		}
	}

	util.GetLogger().Infoln("res", batch)
	batchToReturn := make([]domain.BatchElementResult, 0)
	for _, res := range batch {
		batchToReturn = append(batchToReturn, domain.BatchElementResult{ID: res.ID, ShortenedURL: res.ShortenedURL})
	}
	if uuidShift == 0 {
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

// ReadOriginal gets original URL using shortened.
func (s *url) ReadOriginal(ctx context.Context, shortened string, errChan chan error) (string, error) {
	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
	}

	if deleted, err := s.repo.IsURLDeleted(ctx, shortened); !deleted {
		if err != nil {
			util.GetLogger().Infoln(err)
		}
		for _, url := range *curURLsPtr.Urls {
			if url.ShortURL == shortened {
				return url.OriginalURL, nil
			}
		}
		return "", errors.New("no such value")
	} else if err != nil {
		util.GetLogger().Infoln(err)
		return "", err
	} else {
		errDeleted := errors.New("the requested url was deleted")
		errChan <- errDeleted
		return "", errDeleted
	}
}

// CreateShortened creates shorten URL and calls repository level to save it to database.
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

	curShrtURLs := make(map[string]bool, 0)

	for _, curURL := range *curURLsPtr.Urls {
		curShrtURLs[curURL.ShortURL] = true
	}

	for {
		shortenedURL = util.GenerateRandomString(shrtURLReqLen, random)
		if _, shortExists := curShrtURLs[shortenedURL]; !shortExists {
			break
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

func (s *url) DeleteUserURLs(ctx context.Context, short []domain.URLWithID, shortURLsChan *domain.MutexChanString, once *sync.Once, wg *sync.WaitGroup) {
	shortURLs := struct {
		URLs []string
		uid  []int64
		*sync.Mutex
	}{
		URLs:  make([]string, 0),
		Mutex: &sync.Mutex{},
	}

	var deleteErr error
	go func() {
		once.Do(func() {
			timer := time.Now()
			for {
				select {
				case shrt := <-shortURLsChan.Channel:
					shortURLs.Lock()
					shortURLs.URLs = append(shortURLs.URLs, shrt.URL)
					shortURLs.uid = append(shortURLs.uid, shrt.ID)
					wg.Add(1)
					for len(shortURLsChan.Channel) > 0 {
						shrt = <-shortURLsChan.Channel
						shortURLs.URLs = append(shortURLs.URLs, shrt.URL)
						util.GetLogger().Infoln(ctx.Value(domain.Key("id")).(int64))
						shortURLs.uid = append(shortURLs.uid, shrt.ID)
						util.GetLogger().Infoln("добавил", shrt)
						wg.Add(1)
					}
					shortURLs.Unlock()
				default:
					if len(shortURLs.URLs) >= 10 || (time.Since(timer) > time.Millisecond*450) && len(shortURLs.URLs) > 0 {
						util.GetLogger().Infoln("удаляю...", shortURLs.URLs)
						deleteErr = s.repo.DeleteUserURLs(ctx, shortURLs.URLs, shortURLs.uid)
						if deleteErr != nil {
							util.GetLogger().Infoln(deleteErr)
						}
						g, erro := s.repo.IsURLDeleted(ctx, shortURLs.URLs[0])
						for range shortURLs.uid {
							wg.Done()
						}
						util.GetLogger().Infoln("удалил ли? вот ответ -", g)
						if erro != nil {
							util.GetLogger().Infoln(erro)
						}
						shortURLs.Lock()
						shortURLs.URLs = shortURLs.URLs[:0]
						shortURLs.uid = shortURLs.uid[:0]
						shortURLs.Unlock()
						util.GetLogger().Infoln(len(shortURLs.URLs))
						timer = time.Now()
					}
				}
			}
		})
	}()

	go func() {
		if len(short) != 0 {
			util.GetLogger().Infoln("len short", len(short))
			shortURLsChan.Lock()
			for _, url := range short {
				shortURLsChan.Channel <- url
			}
			shortURLsChan.Unlock()
			short = short[:0]
		}
	}()
}

func (s *url) CountURLsAndUsers(ctx context.Context) (int, int, error) {
	return s.repo.CountURLsAndUsers(ctx)
}
