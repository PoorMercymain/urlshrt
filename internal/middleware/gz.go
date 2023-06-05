package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandle(h http.Handler) http.HandlerFunc {
	gzipFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" && r.Header.Get("Content-Type") != "text/html" && r.Header.Get("Content-Type") != "application/x-gzip" {
			h.ServeHTTP(w, r)
			return
		}

		for _, val := range r.Header.Values("Content-Encoding") {
			if val == "gzip" {
				gzipReader, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				r.Body = gzipReader

				r.Body.Close()
				if r.URL.String() == "/api/shorten" {
					r.Header.Set("Content-Type", "application/json")
				} else {
					r.Header.Set("Content-Type", "text/plain")
				}

			}
		}

		if len(r.Header.Values("Accept-Encoding")) == 0 {
			h.ServeHTTP(w, r)
			return
		}

		for i, v := range r.Header.Values("Accept-Encoding") {
			if strings.Contains(v, "gzip") {
				break
			}

			if i == len(r.Header.Values("Accept-Encoding"))-1 {
				h.ServeHTTP(w, r)
				return
			}
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")

		h.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
	return WithLogging(gzipFunc)
}
