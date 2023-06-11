package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"

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

	shortenedURL, err := h.srv.CreateShortened(r.Context(), originalURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	shortened, err := h.srv.CreateShortened(r.Context(), orig.URL)
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

	shortenedResponse := struct {
		Result string `json:"result"`
	}{
		Result: addr + shortened,
	}
	err = json.NewEncoder(buf).Encode(shortenedResponse)
	if err != nil {
		util.GetLogger().Errorln(err)
		return
	}
	w.Write(buf.Bytes())
}

func (h *url) CreateShortenedFromBatch(w http.ResponseWriter, r *http.Request) {
	orig := make([]domain.BatchElement, 0)

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
	util.GetLogger().Infoln("handler", orig)
	shortened, err := h.srv.CreateShortenedFromBatch(r.Context(), &orig)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	util.GetLogger().Infoln("still handler", shortened)

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
