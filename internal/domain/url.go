package domain

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type URL struct {
	Original  string
	Shortened string
}

func (u *URL) String() string {
	return fmt.Sprintf("%s %s", u.Original, u.Shortened)
}

func (u *URL) GenerateShortURL(w http.ResponseWriter, r *http.Request, urls *[]URL, addr string, randSeed int64, db *Database) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") || r.ContentLength == 0 {
		scanner := bufio.NewScanner(r.Body)
		scanner.Scan()
		originalURL := scanner.Text()

		shortenedURL, err := u.ShortenRawURL(originalURL, urls, randSeed, db)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		if !strings.HasPrefix(addr, "http://") {
			addr = "http://" + addr
		}
		if !strings.HasSuffix(addr, "/") {
			addr = addr + "/"
		}
		w.Write([]byte(addr + shortenedURL))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (u *URL) GenerateShortURLHandler(urls *[]URL, addr string, randSeed int64, db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GenerateShortURL(w, r, urls, addr, randSeed, db)
	}
}

func (u *URL) GetOriginalURL(w http.ResponseWriter, r *http.Request, urls []URL, db *Database) {
	shortenedURL := chi.URLParam(r, "short")

	savedUrls, err := db.getUrls()
	if err != nil {
		savedUrls = urls
	}

	for _, url := range savedUrls {
		if url.Shortened == shortenedURL {
			w.Header().Set("Location", url.Original)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
}

func (u *URL) GetOriginalURLHandler(urls []URL, db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GetOriginalURL(w, r, urls, db)
	}
}

func (u *URL) ShortenRawURL(rawURL string, urls *[]URL, randSeed int64, db *Database) (string, error) {
	random := rand.New(rand.NewSource(randSeed))

	u.Original = rawURL

	savedUrls, errDB := db.getUrls()
	if errDB != nil {
		savedUrls = *urls
	}

	for _, url := range savedUrls {
		if u.Original == url.Original {
			return url.Shortened, nil
		}
	}

	var shortenedURL string

	shrtURLReqLen := 7

	shortenedURL = generateRandomString(shrtURLReqLen, random)

	for _, url := range savedUrls {
		for shortenedURL == url.Shortened {
			shortenedURL = generateRandomString(shrtURLReqLen, random)
		}
	}

	u.Shortened = shortenedURL

	urlStrArr := []string{ u.String() }

	if errDB == nil {
		db.saveStrings(urlStrArr)
	}
	*urls = append(*urls, *u)

	return u.Shortened, nil
}

func generateRandomString(length int, random *rand.Rand) string {
	randStrBytes := make([]byte, length)
	shiftToSkipSymbols := 6

	for i := 0; i < length; i++ {
		symbolCodeLimiter := 'z'-'A' - shiftToSkipSymbols
		symbolCode := random.Intn(symbolCodeLimiter)
		if symbolCode > 'Z'-'A' {
			symbolCode += shiftToSkipSymbols
		}
		randStrBytes[i] = byte('A' + symbolCode)
	}

	return string(randStrBytes)
}
