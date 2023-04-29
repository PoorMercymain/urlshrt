package domain

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type Url struct {
	Original string
	Shortened string
}

func (u Url) String() string {
	return u.Original + " " + u.Shortened
}

func (u Url) ShortenUrlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "text/plain" {
		scanner := bufio.NewScanner(r.Body)
		scanner.Scan()
		originalUrl := scanner.Text()
		fmt.Println("orig", originalUrl)
		shortenedUrl, err := u.ShortenRawUrl(originalUrl)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(201)
		w.Write([]byte(shortenedUrl))
		return
	} else if r.Method == http.MethodGet {
		var shortenedUrl string
		if len(r.URL.String()) > 1 {
			shortenedUrl = r.URL.String()[1:]
		} else {
			shortenedUrl = ""
		}

		fmt.Println("u", shortenedUrl)

		db := NewDB("txt", "testTxtDB.txt")

		savedUrls, err := db.getUrls()
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		fmt.Println("сокр", shortenedUrl)

		for _, url := range savedUrls {
			if url.Shortened == shortenedUrl {
				w.Header().Set("Location", url.Original)
				w.WriteHeader(307)
				return
			}
		}
	}
	w.WriteHeader(400)
}

func (u Url) ShortenRawUrl(rawUrl string) (string, error) {
	rand.Seed(time.Now().Unix())

	db := NewDB("txt", "testTxtDB.txt")
	
	u.Original = rawUrl

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