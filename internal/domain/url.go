package domain

import (
	"bufio"
	"bytes"
	"encoding/json"
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

type OriginalURL struct {
	URL string `json:"url"`
	IsSet bool `json:"-"`
}

type ShortenedURL struct {
	Result string `json:"result"`
}

func (u *URL) String() string {
	return fmt.Sprintf("%s %s", u.Original, u.Shortened)
}

func (u *URL) GenerateShortURLFromJSONHandler(context ctx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request)  {
		var orig OriginalURL
		if err := json.NewDecoder(r.Body).Decode(&orig); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
		orig.IsSet = true

		context.json = orig

		u.GenerateShortURL(w, r, context)
	}
}

func (u *URL) GenerateShortURL(w http.ResponseWriter, r *http.Request, context ctx) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") || r.Header.Get("Content-Type") == "application/json" || r.ContentLength == 0 {
		var originalURL string
		if context.json.IsSet {
			originalURL = context.json.URL
		} else {
			scanner := bufio.NewScanner(r.Body)
			scanner.Scan()
			originalURL = scanner.Text()
		}

		shortenedURL, err := u.ShortenRawURL(originalURL, context.urls, context.randomSeed, context.db)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if !context.json.IsSet {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			if !strings.HasPrefix(context.address, "http://") {
				context.address = "http://" + context.address
			}
			if !strings.HasSuffix(context.address, "/") {
				context.address = context.address + "/"
			}
			w.Write([]byte(context.address + shortenedURL))
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if !strings.HasPrefix(context.address, "http://") {
				context.address = "http://" + context.address
			}
			if !strings.HasSuffix(context.address, "/") {
				context.address = context.address + "/"
			}
			var shortenedJSONBytes []byte
			buf := bytes.NewBuffer(shortenedJSONBytes)
			shortened := ShortenedURL{Result: context.address + shortenedURL}
			err = json.NewEncoder(buf).Encode(shortened)
			if err != nil {
				fmt.Println(err)
				return
			}
			w.Write(buf.Bytes())
			return
		}

	}
	w.WriteHeader(http.StatusBadRequest)
}

func (u *URL) GenerateShortURLHandler(context ctx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GenerateShortURL(w, r, context)
	}
}

func (u *URL) GetOriginalURL(w http.ResponseWriter, r *http.Request, context ctx) {
	shortenedURL := chi.URLParam(r, "short")

	savedUrls, err := context.db.getUrls()
	if err != nil {
		savedUrls = *context.urls
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

func (u *URL) GetOriginalURLHandler(context ctx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GetOriginalURL(w, r, context)
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
