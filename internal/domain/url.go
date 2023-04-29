package domain

import (
	"bufio"
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

func (u URL) ShortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if strings.HasPrefix("text/plain", r.Header.Get("Content-Type")) || r.ContentLength == 0 {
			scanner := bufio.NewScanner(r.Body)
			scanner.Scan()
			originalURL := scanner.Text()

			shortenedURL, err := u.ShortenRawURL(originalURL)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(201)
			w.Write([]byte(shortenedURL))
			return
		}
	} else if r.Method == http.MethodGet {
		var shortenedURL string
		if len(r.URL.String()) > 1 {
			shortenedURL = r.URL.String()[1:]
		} else {
			shortenedURL = ""
		}

		db := NewDB("txt", "testTxtDB.txt")

		savedUrls, err := db.getUrls()
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		for _, url := range savedUrls {
			if url.Shortened == shortenedURL {
				w.Header().Set("Location", url.Original)
				w.WriteHeader(307)
				return
			}
		}
	}
	w.WriteHeader(400)
}

func (u URL) ShortenRawURL(rawURL string) (string, error) {
	rand.Seed(time.Now().Unix())

	db := NewDB("txt", "testTxtDB.txt")

	u.Original = rawURL

	savedUrls, err := db.getUrls()
	if err != nil {
		return "", err
	}

	for _, url := range savedUrls {
		if u.Original == url.Original {
			return url.Shortened, nil
		}
	}

	var shortenedUrl string

	length := 7

    shortenedUrl = generateRandomString(length)

	for _, url := range savedUrls {
		for shortenedUrl == url.Shortened {
			shortenedUrl = generateRandomString(length)
		}
	}

	u.Shortened = shortenedUrl

	var urlStrArr []string

	urlStrArr = append(urlStrArr, u.String())

	db.saveStrings(urlStrArr)

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