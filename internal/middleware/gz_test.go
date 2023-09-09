package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUseGzipReader(t *testing.T) {
	util.InitLogger()

	c := chi.NewRouter()

	ts := httptest.NewServer(c)
	defer ts.Close()

	buf := bytes.NewBuffer([]byte(""))
	w := gzip.NewWriter(buf)
	w.Write([]byte("12345"))
	w.Close()
	r := bytes.NewBuffer(buf.Bytes())

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/test", r)
	require.NoError(t, err)

	req.Header.Set("Content-Encoding", "gzip")

	req2, err := http.NewRequest(http.MethodPost, ts.URL+"/test", r)
	require.NoError(t, err)

	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			util.GetLogger().Infoln("Content-Encoding is not gzip")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		w.WriteHeader(http.StatusOK)
	}

	c.Post("/test", GzipHandle(http.HandlerFunc(fn)))

	util.GetLogger().Infoln("1")
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	util.GetLogger().Infoln("2")
	resp, err = ts.Client().Do(req2)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	req4, err := http.NewRequest(http.MethodPost, ts.URL+"/test", r)
	req4.Header.Set("Content-Encoding", "compress,deflate,br")
	require.NoError(t, err)

	util.GetLogger().Infoln("3")
	resp, err = ts.Client().Do(req4)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}