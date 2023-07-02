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
	err := s.repo.PingPg(ctx)
	return err
}

func (s *url) CreateShortenedFromBatch(ctx context.Context, batch []*domain.BatchElement) ([]domain.BatchElementResult, error) {
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
	for _, urlFromCurURLs := range (*curURLsPtr.Urls) {
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
						UUID: len(*curURLsPtr.Urls)+uuidShift,
						ShortURL: batch[j].ShortenedURL,
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

func (s *url) ReadOriginal(ctx context.Context, shortened string) (string, error) {
	curURLsPtr, err := state.GetCurrentURLsPtr()
	if err != nil {
		return "", err
	}

	if deleted, err := s.repo.IsURLDeleted(ctx, shortened); !deleted {
		if err != nil {
			util.GetLogger().Infoln(err)
		}
		for _, url := range *curURLsPtr.Urls {
			//util.GetLogger().Infoln(url.ShortURL)
			if url.ShortURL == shortened {
				return url.OriginalURL, nil
			}
		}
		return "", errors.New("no such value")
	} else {
		return "", domain.NewErrorDeleted(errors.New("the requested url was deleted"))
	}
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


func (s *url) DeleteUserURLs(ctx context.Context, short []string, shortURLsChan *domain.MutexChanString, once *sync.Once) {
	shortURLs := struct{
		URLs []string
		uid []int64
		*sync.Mutex
	}{
		URLs: make([]string, 0),
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
					shortURLs.URLs = append(shortURLs.URLs, shrt)
					shortURLs.uid = append(shortURLs.uid, ctx.Value(domain.Key("id")).(int64))
					util.GetLogger().Infoln("добавил", shrt)
					shortURLs.Unlock()
				default:
					if len(shortURLs.URLs) == 10 || (time.Since(timer) > time.Second) && len(shortURLs.URLs) > 0 {
						util.GetLogger().Infoln("удаляю...", shortURLs.URLs)
						deleteErr = s.repo.DeleteUserURLs(ctx, shortURLs.URLs, shortURLs.uid)
						util.GetLogger().Infoln(deleteErr)
						time.Sleep(time.Second)
						g, erro := s.repo.IsURLDeleted(ctx, shortURLs.URLs[0])
						util.GetLogger().Infoln("удалил ли? вот ответ -", g, erro)
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
				util.GetLogger().Infoln("туть")
				shortURLsChan.Channel<-url
			}
			shortURLsChan.Unlock()
			util.GetLogger().Infoln("пум-пум", short)
			short = short[:0]
			util.GetLogger().Infoln("не", short)
		}
	}()
}