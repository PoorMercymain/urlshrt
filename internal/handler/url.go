package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/go-chi/chi/v5"
)

type url struct {
	srv domain.URLService
}

func NewURL(srv domain.URLService) *url {
	return &url{srv: srv}
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
		if contentType == "text/plain" || contentType == "text/plain; charset=utf-8"{
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

	//ctx := r.Context()
	//ctx = context.WithValue(ctx, "rand_seed", )
	shortenedURL := h.srv.CreateShortened(r.Context(), originalURL)

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
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

	shortened := h.srv.CreateShortened(r.Context(), orig.URL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	var shortenedJSONBytes []byte
	buf := bytes.NewBuffer(shortenedJSONBytes)

	addr := state.GetBaseShortAddress()
	if addr[len(addr)-1] != '/' {
		addr = addr + "/"
	}

	shortenedResponse := struct{
			Result string `json:"result"`
		}{
			Result: addr + shortened,
		}
	err := json.NewEncoder(buf).Encode(shortenedResponse)
	if err != nil {
		middleware.GetLogger().Errorln(err)
		return
	}
	w.Write(buf.Bytes())
}