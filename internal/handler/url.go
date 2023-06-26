package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/go-chi/chi/v5"
)

type url struct {
	srv domain.URLService
}

func NewURL(srv domain.URLService) *url {
	return &url{srv: srv}
}

func (h *url) PingPg(w http.ResponseWriter, r *http.Request) {
	err := h.srv.PingPg(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *url) ReadOriginal(w http.ResponseWriter, r *http.Request) {
	shortenedURL := chi.URLParam(r, "short")

	orig, err := h.srv.ReadOriginal(r.Context(), shortenedURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", orig)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *url) CreateShortened(w http.ResponseWriter, r *http.Request) {
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
	if err != nil && errors.As(err, &uErr) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(addr + shortenedURL))
		return
	} else if err != nil {
		cookie, err := r.Cookie("auth")
		if err != nil {
			util.GetLogger().Infoln(err)
		} else {
			http.SetCookie(w, cookie)
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cookie, err := r.Cookie("auth")
	if err != nil {
		util.GetLogger().Infoln(err)
	} else {
		http.SetCookie(w, cookie)
	}
	w.Header().Set("Content-Type", "text/plain")
	if unauthorized := ctx.Value(domain.Key("auth")); unauthorized != nil {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	w.Write([]byte(addr + shortenedURL))
}

func (h *url) CreateShortenedFromJSON(w http.ResponseWriter, r *http.Request) {
	var orig OriginalURL

	if len(r.Header.Values("Content-Type")) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for contentTypeCurrentIndex, contentType := range r.Header.Values("Content-Type") {
		if contentType == "application/json" {
			break
		}
		if contentTypeCurrentIndex == len(r.Header.Values("Content-Type"))-1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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
	w.Write(buf.Bytes())
}

func (h *url) CreateShortenedFromBatch(w http.ResponseWriter, r *http.Request) {
	orig := make([]*domain.BatchElement, 0)

	if len(r.Header.Values("Content-Type")) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for contentTypeCurrentIndex, contentType := range r.Header.Values("Content-Type") {
		if contentType == "application/json" {
			break
		}
		if contentTypeCurrentIndex == len(r.Header.Values("Content-Type"))-1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if err := json.NewDecoder(r.Body).Decode(&orig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(orig) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//origPtrs := make([]*domain.BatchElement, 0)


	shortened, err := h.srv.CreateShortenedFromBatch(r.Context(), orig)
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
	w.Write(buf.Bytes())
}


func (h *url) ReadUserURLs(w http.ResponseWriter, r *http.Request) {
	UserURLs, err := h.srv.ReadUserURLs(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

	if unauthorized := r.Context().Value(domain.Key("auth")); unauthorized != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	UserURLsOutput := make([]domain.UserOutput, 0)

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
	w.Write(buf.Bytes())
}