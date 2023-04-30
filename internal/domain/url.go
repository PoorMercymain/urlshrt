package domain

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type URL struct {
	Original string
	Shortened string
}

func (u URL) String() string {
	return u.Original + " " + u.Shortened
}

func (u URL) generateShortURL(w http.ResponseWriter, r *http.Request, urls []URL) {
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
		w.WriteHeader(201)
		w.Write([]byte("http://localhost:8080/" + shortenedURL))
		return
	}
	w.WriteHeader(400)
}

func (u URL) getOriginalURL(w http.ResponseWriter, r *http.Request, urls []URL) {
	var shortenedURL string
			if len(r.URL.String()) > 1 {
				shortenedURL = r.URL.String()[1:]
			} else {
				shortenedURL = ""
			}

			db := NewDB("txt", "testTxtDB.txt")

			savedUrls, err := db.getUrls()
			if err != nil {
				savedUrls = urls
			}

			for _, url := range savedUrls {
				if url.Shortened == shortenedURL {
					w.Header().Set("Location", url.Original)
					w.WriteHeader(307)
					return
				}
			}

			w.WriteHeader(400)
}

func (u URL) ShortenURLHandler(urls []URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			u.generateShortURL(w, r, urls)
		case http.MethodGet:
			u.getOriginalURL(w, r, urls)
		default:
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
	}
}

func (u URL) ShortenRawURL(rawURL string, urls []URL) (string, error) {
	rand.Seed(time.Now().Unix())

	db := NewDB("txt", "testTxtDB.txt")

	u.Original = rawURL

	savedUrls, errDB := db.getUrls()
	if errDB != nil {
		savedUrls = urls
	}

	for _, url := range savedUrls {
		if u.Original == url.Original {
			return url.Shortened, nil
		}
	}

	var shortenedURL string

	length := 7

    shortenedURL = generateRandomString(length)

	for _, url := range savedUrls {
		for shortenedURL == url.Shortened {
			shortenedURL = generateRandomString(length)
		}
	}

	u.Shortened = shortenedURL

	var urlStrArr []string

	urlStrArr = append(urlStrArr, u.String())
	if errDB == nil {
		db.saveStrings(urlStrArr)
	}
	urls = append(urls, u)
	fmt.Println(urls)

	return u.Shortened, nil
}

func generateRandomString(length int) string {
	randStrBytes := make([]byte, length)

    for i := 0; i < length; i++ {
		symbolCode := rand.Intn(53)
		if symbolCode > 25 {
			symbolCode += 6
		}
        randStrBytes[i] = byte(65 + symbolCode)
    }

	return string(randStrBytes)
}