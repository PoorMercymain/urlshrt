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
	var short = struct{Result string `json:"result"`}{}

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

	//var url URL

	urls := []state.URLStringJSON{{UUID: 1, ShortURL: "aBcDeFg", OriginalURL: "https://ya.ru"}}

	host := "http://localhost:8080"

	//db := NewDB("txt", "testTxtDB.txt")

	middleware.InitLogger()

	defer middleware.GetLogger().Sync()

	ur := repository.NewURL("")
	us := service.NewURL(ur)
	uh := NewURL(us)

	state.InitCurrentURLs(&urls)
	state.InitShortAddress(host)

	//data := NewData(&urls, host, time.Now().Unix(), db, "", false, new(sync.Mutex))

	r.Post("/", middleware.GzipHandle(http.HandlerFunc(uh.CreateShortened)))
	r.Get("/{short}", middleware.GzipHandle(http.HandlerFunc(uh.ReadOriginal)))
	r.Post("/api/shorten", middleware.GzipHandle(http.HandlerFunc(uh.CreateShortenedFromJSON)))

	return r
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

	re, post := testRequest(t, ts, testTable[0].status, testTable[0].body, "POST", testTable[0].url)
	assert.Equal(t, testTable[0].want, post)
	re.Body.Close()
	re, _ = testRequest(t, ts, testTable[1].status, testTable[1].body, "GET", testTable[1].url)
	re.Body.Close()
	re, postJSON := testRequest(t, ts, testTable[2].status, testTable[2].body, "POST with JSON", testTable[2].url)
	assert.Equal(t, testTable[2].want, postJSON)
	re.Body.Close()
}
