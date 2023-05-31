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
	URL   string `json:"url"`
	IsSet bool   `json:"-"`
}

type ShortenedURL struct {
	Result string `json:"result"`
}

func (u *URL) String() string {
	return fmt.Sprintf("%s %s", u.Original, u.Shortened)
}

func (u *URL) GenerateShortURLFromJSONHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var orig OriginalURL

		if err := json.NewDecoder(r.Body).Decode(&orig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		orig.IsSet = true

		data.Lock()
		data.json = orig
		data.Unlock()

		u.GenerateShortURL(w, r, data)
	}
}

func (u *URL) GenerateShortURL(w http.ResponseWriter, r *http.Request, data *Data) {
	contentTypeOk := false
	for _, val := range r.Header.Values("Content-Type") {
		if val == "text/plain" {
			contentTypeOk = true
			break
		}
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") || r.Header.Get("Content-Type") == "application/json" || r.ContentLength == 0 || contentTypeOk {
		var originalURL string
		if data.json.IsSet {
			originalURL = data.json.URL
		} else {
			scanner := bufio.NewScanner(r.Body)
			scanner.Scan()
			originalURL = scanner.Text()
		}

		shortenedURL, err := u.ShortenRawURL(originalURL, data)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if !data.json.IsSet {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(data.address + "/" + shortenedURL))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		var shortenedJSONBytes []byte
		buf := bytes.NewBuffer(shortenedJSONBytes)
		shortened := ShortenedURL{Result: data.address + "/" + shortenedURL}
		err = json.NewEncoder(buf).Encode(shortened)
		if err != nil {
			fmt.Println(err)
			return
		}
		w.Write(buf.Bytes())
		return

	}
	w.WriteHeader(http.StatusBadRequest)
}

func (u *URL) GenerateShortURLHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GenerateShortURL(w, r, data)
	}
}

func (u *URL) GetOriginalURL(w http.ResponseWriter, r *http.Request, data *Data) {
	shortenedURL := chi.URLParam(r, "short")

	for _, url := range *data.urls {
		if url.ShortURL == shortenedURL {
			w.Header().Set("Location", url.OriginalURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
}

func (u *URL) GetOriginalURLHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.GetOriginalURL(w, r, data)
	}
}

func (u *URL) ShortenRawURL(rawURL string, data *Data) (string, error) {
	random := rand.New(rand.NewSource(data.randomSeed))

	u.Original = rawURL

	for _, url := range *data.urls {
		if u.Original == url.OriginalURL {
			return url.ShortURL, nil
		}
	}

	var shortenedURL string

	shrtURLReqLen := 7

	shortenedURL = generateRandomString(shrtURLReqLen, random)

	for _, url := range *data.urls {
		for shortenedURL == url.ShortURL {
			shortenedURL = generateRandomString(shrtURLReqLen, random)
		}
	}

	u.Shortened = shortenedURL

	createdURL := URLStringJSON{UUID: len(*data.urls), ShortURL: u.Shortened, OriginalURL: u.Original}
	data.Lock()
	*data.urls = append(*data.urls, createdURL)
	data.Unlock()

	if data.db.location != "" {
		data.db.saveStrings([]URLStringJSON{createdURL})
	}

	return u.Shortened, nil
}

func generateRandomString(length int, random *rand.Rand) string {
	randStrBytes := make([]byte, length)
	shiftToSkipSymbols := 6

	for i := 0; i < length; i++ {
		symbolCodeLimiter := 'z' - 'A' - shiftToSkipSymbols
		symbolCode := random.Intn(symbolCodeLimiter)
		if symbolCode > 'Z'-'A' {
			symbolCode += shiftToSkipSymbols
		}
		randStrBytes[i] = byte('A' + symbolCode)
	}

	return string(randStrBytes)
}
