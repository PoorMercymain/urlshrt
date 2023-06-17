package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/repository"
	"github.com/PoorMercymain/urlshrt/internal/service"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, code int, body, method, path string) (*http.Response, string) {

	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, ts.URL+path, nil)
	} else if method == "POST" {
		req, err = http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	} else if method == "POST with JSON" {
		req, err = http.NewRequest("POST", ts.URL+path, strings.NewReader(body))
	}
	if method == "POST" {
		req.Header.Set("Content-Type", "text/plain")
	} else if method == "POST with JSON" {
		req.Header.Set("Content-Type", "application/json")
	}

	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	if err != http.ErrUseLastResponse {
		require.NoError(t, err)
	}

	defer resp.Body.Close()

	if method != "GET" {
		assert.Equal(t, code, resp.StatusCode)
	} else {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, _ := client.Get(ts.URL + path)
		resp.Body.Close()
		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	}

	var respBody []byte
	var short = struct {
		Result string `json:"result"`
	}{}

	if method != "POST with JSON" {
		respBody, err = io.ReadAll(resp.Body)
	} else {
		err = json.NewDecoder(resp.Body).Decode(&short)
		respBody = []byte(short.Result)
	}

	require.NoError(t, err)

	return resp, string(respBody)
}

func router() chi.Router {
	r := chi.NewRouter()

	urls := []state.URLStringJSON{{UUID: 1, ShortURL: "aBcDeFg", OriginalURL: "https://ya.ru"}}

	host := "http://localhost:8080"

	util.InitLogger()

	defer util.GetLogger().Sync()

	ur := repository.NewURL("")
	us := service.NewURL(ur)
	uh := NewURL(us)

	urlsMap := make(map[string]state.URLStringJSON)
	for _, u := range urls {
		urlsMap[u.OriginalURL] = u
	}

	util.GetLogger().Infoln(urlsMap)
	state.InitCurrentURLs(&urlsMap)
	state.InitShortAddress(host)

	u, err := state.GetCurrentURLsPtr()
	if err != nil {
		util.GetLogger().Infoln(err)
	}
	util.GetLogger().Infoln(u.Urls)

	r.Post("/", WrapHandler(uh.CreateShortened))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))

	return r
}

func WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return middleware.GzipHandle(middleware.WithLogging(h))
}

func TestRouter(t *testing.T) {
	ts := httptest.NewServer(router())

	defer ts.Close()

	var testTable = []struct {
		url    string
		status int
		body   string
		want   string
	}{
		{"/", 201, "https://ya.ru", "http://localhost:8080/aBcDeFg"},
		{"/aBcDeFg", 307, "", "https://ya.ru"},
		{url: "/api/shorten", status: 201, body: "{\"url\":\"https://ya.ru\"}", want: "http://localhost:8080/aBcDeFg"},
	}

	re, _ := testRequest(t, ts, testTable[0].status, testTable[0].body, "POST", testTable[0].url)
	//assert.Equal(t, testTable[0].want, post)
	re.Body.Close()

	u, err := state.GetCurrentURLsPtr()
	if err != nil {
		util.GetLogger().Infoln(err)
	}

	util.GetLogger().Infoln(u.Urls)
	re, _ = testRequest(t, ts, testTable[1].status, testTable[1].body, "GET", testTable[1].url)
	re.Body.Close()
	re, _ = testRequest(t, ts, testTable[2].status, testTable[2].body, "POST with JSON", testTable[2].url)
	//assert.Equal(t, testTable[2].want, postJSON)
	re.Body.Close()
}
