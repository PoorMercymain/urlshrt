package domain

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type URL struct {
	Original  string
	Shortened string
}

func (u *URL) String() string {
	return fmt.Sprintf("%s %s", u.Original, u.Shortened)
}

func (u *URL) GenerateShortURL(w http.ResponseWriter, r *http.Request, urls *[]URL, addr string) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") || r.ContentLength == 0 {
		scanner := bufio.NewScanner(r.Body)
		scanner.Scan()
		originalURL := scanner.Text()

		shortenedURL, err := u.ShortenRawURL(originalURL, urls)
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

func (u *URL) GenerateShortURLHandler(urls *[]URL, addr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GenerateShortURL(w, r, urls, addr)
	}
}

func (u *URL) GetOriginalURL(w http.ResponseWriter, r *http.Request, urls []URL) {
	shortenedURL := chi.URLParam(r, "short")

	db := NewDB("txt", "testTxtDB.txt")

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

func (u *URL) GetOriginalURLHandler(urls []URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GetOriginalURL(w, r, urls)
	}
}

func (u *URL) ShortenRawURL(rawURL string, urls *[]URL) (string, error) {
	rand.Seed(time.Now().Unix())

	db := NewDB("txt", "testTxtDB.txt")

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

	shortenedURL = generateRandomString(shrtURLReqLen)

	for _, url := range savedUrls {
		for shortenedURL == url.Shortened {
			shortenedURL = generateRandomString(shrtURLReqLen)
		}
	}

	u.Shortened = shortenedURL

	urlStrArr := make([]string, 0)

	urlStrArr = append(urlStrArr, u.String())

	if errDB == nil {
		db.saveStrings(urlStrArr)
	}
	*urls = append(*urls, *u)

	return u.Shortened, nil
}

func generateRandomString(length int) string {
	randStrBytes := make([]byte, length)
	shiftToSkipSymbols := 6

	for i := 0; i < length; i++ {
		symbolCodeLimiter := 'z'-'A' - shiftToSkipSymbols
		symbolCode := rand.Intn(symbolCodeLimiter)
		if symbolCode > 'Z'-'A' {
			symbolCode += shiftToSkipSymbols
		}
		randStrBytes[i] = byte('A' + symbolCode)
	}

	return string(randStrBytes)
}
