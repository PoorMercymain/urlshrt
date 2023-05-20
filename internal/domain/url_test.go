package domain

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testRequest(t *testing.T, ts *httptest.Server, code int, body, method, path string) (*http.Response, string) {
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, ts.URL+path, nil)
	} else {
		req, err = http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "text/plain")
	}

	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	t.Log(req)
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
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func router() chi.Router {
	r := chi.NewRouter()

	var url URL
	urls := make([]URL, 0)
	urls = append(urls, URL{Original: "https://ya.ru", Shortened: "aBcDeFg"})

	host := "http://localhost:8080/"

	db := NewDB("txt", "testTxtDB.txt")

	logger, err := zap.NewDevelopment()
    if err != nil {
		fmt.Println(err)
        return nil
    }
    defer logger.Sync()

	sugar := *logger.Sugar()

	postContext := NewContext(&urls, host, time.Now().Unix(), db)
	getContext := NewContext(&urls, "", 0, db)

	r.Post("/", WithLogging(url.GenerateShortURLHandler(*postContext), &sugar))
	r.Get("/{short}", WithLogging(url.GetOriginalURLHandler(*getContext), &sugar))

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
	}

	re, post := testRequest(t, ts, testTable[0].status, testTable[0].body, "POST", testTable[0].url)
	assert.Equal(t, testTable[0].want, post)
	re.Body.Close()
	re, _ = testRequest(t, ts, testTable[1].status, testTable[1].body, "GET", testTable[1].url)
	re.Body.Close()
}
