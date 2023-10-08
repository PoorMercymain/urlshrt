// handler package contains handler functions for urlshrt project.
package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

type URL struct {
	srv domain.URLService
}

// NewURL creates object to operate handler functions.
func NewURL(srv domain.URLService) *URL {
	return &URL{srv: srv}
}

// PingPg - handler to check connection to Postgres.
func (h *URL) PingPg(w http.ResponseWriter, r *http.Request) {
	err := h.srv.PingPg(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ReadOriginal - handler to get original URL from shortened.
func (h *URL) ReadOriginal(w http.ResponseWriter, r *http.Request) {
	shortenedURL := chi.URLParam(r, "short")

	errChan := make(chan error, 1)
	orig, err := h.srv.ReadOriginal(r.Context(), shortenedURL, errChan)
	select {
	case errDeleted := <-errChan:
		util.GetLogger().Infoln(errDeleted)
		w.WriteHeader(http.StatusGone)
		return
	default:
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Location", orig)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// CreateShortened - handler to create short URL from original.
func (h *URL) CreateShortened(w http.ResponseWriter, r *http.Request) {
	if len(r.Header.Values("Content-Type")) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for contentTypeCurrentIndex, contentType := range r.Header.Values("Content-Type") {
		if contentType == "text/plain" || contentType == "text/plain; charset=utf-8" {
			break
		}
		if contentTypeCurrentIndex == len(r.Header.Values("Content-Type"))-1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	var originalURL string

	scanner := bufio.NewScanner(r.Body)
	scanner.Scan()
	originalURL = scanner.Text()

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	ctx := r.Context()
	randSeed, err := strconv.Atoi(r.Header.Get("RandSeed"))
	if err != nil {
		util.GetLogger().Infoln("RandSeed not provided through request headers or incorrect")
	} else {
		ctx = context.WithValue(r.Context(), domain.Key("seed"), int64(randSeed))
		util.GetLogger().Infoln("RandSeed provided", randSeed)
	}
	util.GetLogger().Infoln(ctx)
	shortenedURL, err := h.srv.CreateShortened(ctx, originalURL)
	var uErr *domain.UniqueError
	util.GetLogger().Infoln("here")
	if err != nil && errors.As(err, &uErr) {
		util.GetLogger().Infoln("and here")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusConflict)
		_, err = w.Write([]byte(addr + shortenedURL))
		if err != nil {
			return
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(addr + shortenedURL))
	if err != nil {
		return
	}
}

// CreateShortenedFromJSON - handler to create short URL from original which is in JSON.
func (h *URL) CreateShortenedFromJSON(w http.ResponseWriter, r *http.Request) {
	var orig OriginalURL

	if !IsJSONContentTypeCorrect(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&orig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	shortened, err := h.srv.CreateShortened(r.Context(), orig.URL)
	var uErr *domain.UniqueError
	if err != nil && errors.As(err, &uErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
	}

	var shortenedJSONBytes []byte
	buf := bytes.NewBuffer(shortenedJSONBytes)

	shortenedResponse := struct {
		Result string `json:"result"`
	}{
		Result: addr + shortened,
	}
	util.GetLogger().Infoln("sending...", shortenedResponse)
	err = json.NewEncoder(buf).Encode(shortenedResponse)
	if err != nil {
		util.GetLogger().Errorln(err)
		return
	}
	_, err = w.Write(buf.Bytes())
	if err != nil {
		return
	}
}

// CreateShortenedFromBatchAdapter - adapter for handler to create shortened URLs from batch in JSON.
func (h *URL) CreateShortenedFromBatchAdapter(wg *sync.WaitGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orig := make([]*domain.BatchElement, 0, 1)

		if !IsJSONContentTypeCorrect(r) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&orig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(orig) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		shortened, err := h.srv.CreateShortenedFromBatch(r.Context(), orig, wg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		var shortenedJSONBytes []byte
		buf := bytes.NewBuffer(shortenedJSONBytes)

		addr := state.GetBaseShortAddress()
		if addr[len(addr)-1] != '/' {
			addr = addr + "/"
		}

		for i, shrt := range shortened {
			shortened[i].ShortenedURL = addr + shrt.ShortenedURL
		}

		err = json.NewEncoder(buf).Encode(shortened)
		if err != nil {
			util.GetLogger().Errorln(err)
			return
		}
		_, err = w.Write(buf.Bytes())
		if err != nil {
			return
		}
	}
}

// ReadUserURLs - handler to get all user's URLs.
func (h *URL) ReadUserURLs(w http.ResponseWriter, r *http.Request) {
	UserURLs, err := h.srv.ReadUserURLs(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		util.GetLogger().Infoln(err)
		return
	}

	if len(UserURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cookie, err := r.Cookie("auth")
	if err != nil {
		util.GetLogger().Infoln(err)
		return
	} else {
		http.SetCookie(w, cookie)
	}

	if unauthorized := r.Context().Value(domain.Key("unauthorized")); unauthorized != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	UserURLsOutput := make([]domain.UserOutput, 0, len(UserURLs))

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	for _, usrURL := range UserURLs {
		UserURLsOutput = append(UserURLsOutput, domain.UserOutput{ShortURL: addr + usrURL.ShortURL, OriginalURL: usrURL.OriginalURL})
	}

	var JSONBytes []byte
	buf := bytes.NewBuffer(JSONBytes)

	err = json.NewEncoder(buf).Encode(UserURLsOutput)
	if err != nil {
		util.GetLogger().Errorln(err)
		return
	}
	_, err = w.Write(buf.Bytes())
	if err != nil {
		return
	}
}

func (h *URL) ReadAmountOfURLsAndUsers(w http.ResponseWriter, r *http.Request) {
	var amounts domain.Amounts
	var err error
	amounts.URLs, amounts.Users, err = h.srv.CountURLsAndUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var amountsJSONBytes []byte
	buf := bytes.NewBuffer(amountsJSONBytes)
	err = json.NewEncoder(buf).Encode(amounts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(buf.Bytes())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// DeleteUserURLsAdapter - adapter for closure function to mark URL as deleted.
func (h *URL) DeleteUserURLsAdapter(shortURLsChan *domain.MutexChanString, once *sync.Once, wg *sync.WaitGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		short := make([]string, 0, 1)

		if !IsJSONContentTypeCorrect(r) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&short); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		shortURLWithID := make([]domain.URLWithID, 0, len(short))
		for _, url := range short {
			shortURLWithID = append(shortURLWithID, domain.URLWithID{URL: url, ID: r.Context().Value(domain.Key("id")).(int64)})
		}

		util.GetLogger().Infoln("попытка удалить", short)
		util.GetLogger().Infoln(len(short))

		if len(short) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		go func() {
			h.srv.DeleteUserURLs(r.Context(), shortURLWithID, shortURLsChan, once, wg)
		}()

		w.WriteHeader(http.StatusAccepted)
	}
}

// IsJSONContentTypeCorrect - function to check content type of an http request.
func IsJSONContentTypeCorrect(r *http.Request) bool {
	if len(r.Header.Values("Content-Type")) == 0 {
		return false
	}

	for contentTypeCurrentIndex, contentType := range r.Header.Values("Content-Type") {
		if contentType == "application/json" {
			break
		}
		if contentTypeCurrentIndex == len(r.Header.Values("Content-Type"))-1 {
			return false
		}
	}

	return true
}
