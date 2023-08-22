package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/domain/mocks"
	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/repository"
	"github.com/PoorMercymain/urlshrt/internal/service"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

func testRequest(t *testing.T, ts *httptest.Server, code int, body, method, path string) (*http.Response, string) {

	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, ts.URL+path, nil)
		util.GetLogger().Infoln(req)
		jwt, _, err := middleware.BuildJWTString()
		require.NoError(t, err)
		cookie := &http.Cookie{Name: "auth", Value: jwt}
		req.AddCookie(cookie)
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
	} else if path != "/api/user/urls" {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, _ := client.Get(ts.URL + path)
		resp.Body.Close()
		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	} else {
		assert.Equal(t, code, resp.StatusCode)
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

func router( /*fmem *os.File*/ ) chi.Router {
	r := chi.NewRouter()

	urls := []state.URLStringJSON{{UUID: 1, ShortURL: "aBcDeFg", OriginalURL: "https://ya.ru"}}

	host := "http://localhost:8080"

	util.InitLogger()

	defer util.GetLogger().Sync()

	pg := &state.Postgres{}

	ur := repository.NewURL("", pg)
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

	r.Post("/", WrapHandler(uh.CreateShortened /*, fmem*/))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal /*, fmem*/))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON /*, fmem*/))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))

	return r
}

func WrapHandler(h http.HandlerFunc /*, fmem *os.File*/) http.HandlerFunc {
	return middleware.GzipHandle(middleware.Authorize(middleware.WithLogging(h /*, fmem*/)))
}

func TestRouter(t *testing.T) {
	/*fmem, err := os.Create(`profiles\base.pprof`)
		if err != nil {
	    	util.GetLogger().Infoln(err)
	    }*/

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
		//{url: "/api/user/urls", status: 200, body: "", want: "[{\"short_url\": \"http://localhost:8080/aBcDeFg\",\"original_url\": \"https://ya.ru\"}]"},
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
	//re, _ = testRequest(t, ts, testTable[3].status, testTable[3].body, "GET", testTable[3].url)
	//re.Body.Close()
}

func benchmarkRequest(b *testing.B, ts *httptest.Server, body, method, path, contentType string) string {
	util.GetLogger().Infoln("a")

	var req *http.Request
	var err error
	util.GetLogger().Infoln(method)
	if body == "" {
		req, err = http.NewRequest(method, ts.URL+path, nil)
	} else {
		req, err = http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		util.GetLogger().Infoln(req)
	}
	if err != nil {
		util.GetLogger().Infoln(err)
		return ""
	}

	util.GetLogger().Infoln(req)
	req.Header.Set("Content-Type", contentType)

	resp, err := ts.Client().Do(req)
	util.GetLogger().Infoln(resp)
	if err != nil && err != http.ErrUseLastResponse {
		util.GetLogger().Infoln(err)
		return ""
	}

	if resp.StatusCode != http.StatusNoContent {
		defer resp.Body.Close()
	}

	var respBody []byte

	if contentType != "application/json" && resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
	} else if path == "/api/shorten" {
		var short = struct {
			Result string `json:"result"`
		}{}

		err = json.NewDecoder(resp.Body).Decode(&short)
		respBody = []byte(short.Result)
	} else if path == "/api/shorten/batch" {
		var batch = []struct {
			Correlation string `json:"correlation_id"`
			Short       string `json:"short_url"`
		}{}

		err = json.NewDecoder(resp.Body).Decode(&batch)
		respBody = []byte(batch[0].Correlation + " " + batch[0].Short)
	}
	util.GetLogger().Infoln("be")

	if err != nil {
		util.GetLogger().Infoln(err)
		return ""
	}

	return string(respBody)
}

func benchmarkRouter(b *testing.B) chi.Router {
	r := chi.NewRouter()

	host := "http://localhost:8080"

	util.InitLogger()

	defer util.GetLogger().Sync()

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ur := mocks.NewMockURLRepository(ctrl)

	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	ur.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	ur.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().ReadAll(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()
	ur.EXPECT().ReadUserURLs(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()

	us := service.NewURL(ur)
	uh := NewURL(us)

	urlsMap := make(map[string]state.URLStringJSON)

	util.GetLogger().Infoln(urlsMap)
	state.InitCurrentURLs(&urlsMap)
	state.InitShortAddress(host)

	u, err := state.GetCurrentURLsPtr()
	if err != nil {
		util.GetLogger().Infoln(err)
	}
	util.GetLogger().Infoln(u.Urls)

	shortURLsChan := domain.NewMutexChanString(make(chan domain.URLWithID, 10))
	var once sync.Once

	r.Post("/", WrapHandler(uh.CreateShortened))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatch))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))
	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(shortURLsChan, &once)))

	return r
}

func BenchmarkHandlers(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/", "https://ya.ru", "text/plain", http.MethodPost},
		{"/GqKWdrE", "", "", http.MethodGet},
		{"/api/shorten", "{\"url\":\"https://ya.ru\"}", "application/json", http.MethodPost},
		{"/api/shorten/batch", "[{\"correlation_id\": \"тего\",\"original_url\": \"https://hh.ru\"}]", "application/json", http.MethodPost},
		{"/api/user/urls", "", "", http.MethodGet},
		{"/api/user/urls", "[\"GqKWdrE\"]", "application/json", http.MethodDelete},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkShorten(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/", "https://ya.ru", "text/plain", http.MethodPost},
		{"/", "https://practicum.yandex.ru", "text/plain", http.MethodPost},
		{"/", "https://eda.yandex.ru", "text/plain", http.MethodPost},
		{"/", "https://music.yandex.ru", "text/plain", http.MethodPost},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkReadOriginal(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/GqKWdrE", "", "", http.MethodGet},
		{"/AbCdEfG", "", "", http.MethodGet},
		{"/Qwertyu", "", "", http.MethodGet},
		{"/noooooo", "", "", http.MethodGet},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkShortenJSON(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/shorten", "{\"url\":\"https://ya.ru\"}", "application/json", http.MethodPost},
		{"/api/shorten", "{\"url\":\"https://practicum.yandex.ru\"}", "application/json", http.MethodPost},
		{"/api/shorten", "{\"url\":\"https://eda.yandex.ru\"}", "application/json", http.MethodPost},
		{"/api/shorten", "{\"url\":\"https://music.yandex.ru\"}", "application/json", http.MethodPost},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkShortenBatch(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/shorten/batch", "[{\"correlation_id\": \"тего\",\"original_url\": \"https://hh.ru\"}]", "application/json", http.MethodPost},
		{"/api/shorten/batch", "[{\"correlation_id\": \"тего1\",\"original_url\": \"https://practicum.yandex.ru\"}]", "application/json", http.MethodPost},
		{"/api/shorten/batch", "[{\"correlation_id\": \"тего2\",\"original_url\": \"https://eda.yandex.ru\"}]", "application/json", http.MethodPost},
		{"/api/shorten/batch", "[{\"correlation_id\": \"тего3\",\"original_url\": \"https://music.yandex.ru\"}]", "application/json", http.MethodPost},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkReadURLs(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/user/urls", "", "", http.MethodGet},
		{"/api/user/urls", "", "", http.MethodGet},
		{"/api/user/urls", "", "", http.MethodGet},
		{"/api/user/urls", "", "", http.MethodGet},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	ts := httptest.NewServer(benchmarkRouter(b))
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/user/urls", "[\"GqKWdrE\"]", "application/json", http.MethodDelete},
		{"/api/user/urls", "[\"AbCdEfG\"]", "application/json", http.MethodDelete},
		{"/api/user/urls", "[\"Qwertyu\"]", "application/json", http.MethodDelete},
		{"/api/user/urls", "[\"noooooo\"]", "application/json", http.MethodDelete},
	}

	for i := 0; i < b.N; i++ {
		for j := range testTable {
			benchmarkRequest(b, ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		}
	}
}

type testReporter struct {
}

func (tr testReporter) Errorf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	fmt.Println(err)
}

func (tr testReporter) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func GetExampleMockRepo() *mocks.MockURLRepository {
	var tr testReporter
	ctrl := gomock.NewController(tr)
	defer ctrl.Finish()

	ur := mocks.NewMockURLRepository(ctrl)

	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	ur.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	ur.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().ReadAll(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()
	ur.EXPECT().ReadUserURLs(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()

	return ur
}

func GetExampleMockSrv() *mocks.MockURLService {
	var tr testReporter
	ctrl := gomock.NewController(tr)
	defer ctrl.Finish()

	us := mocks.NewMockURLService(ctrl)

	ber := make([]domain.BatchElementResult, 1)
	ber = append(ber, domain.BatchElementResult{ID: "1", ShortenedURL: "http://localhost:8080/GqKWdrE"})

	usj := make([]state.URLStringJSON, 1)
	usj = append(usj, state.URLStringJSON{UUID: 1, ShortURL: "http://localhost:8080/GqKWdrE", OriginalURL: "https://ya.ru"})

	us.EXPECT().CreateShortened(gomock.Any(), gomock.Any()).Return("GqKWdrE", nil).AnyTimes()
	us.EXPECT().ReadOriginal(gomock.Any(), gomock.Any(), gomock.Any()).Return("https://ya.ru", nil).AnyTimes()
	us.EXPECT().CreateShortenedFromBatch(gomock.Any(), gomock.Any()).Return(ber, nil).AnyTimes()
	us.EXPECT().ReadUserURLs(gomock.Any()).Return(usj, nil).AnyTimes()
	us.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return().AnyTimes()

	return us
}

/*func exampleRouter() chi.Router {
	r := chi.NewRouter()

	host := "http://localhost:8080"

	util.InitLogger()

	defer util.GetLogger().Sync()

	var tr testReporter
	ctrl := gomock.NewController(tr)
	defer ctrl.Finish()

	ur := mocks.NewMockURLRepository(ctrl)

	ur.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	ur.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().IsURLDeleted(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	ur.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ur.EXPECT().ReadAll(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()
	ur.EXPECT().ReadUserURLs(gomock.Any()).Return(make([]state.URLStringJSON, 0), nil).AnyTimes()

	us := service.NewURL(ur)
	uh := NewURL(us)

	urlsMap := make(map[string]state.URLStringJSON)

	util.GetLogger().Infoln(urlsMap)
	state.InitCurrentURLs(&urlsMap)
	state.InitShortAddress(host)

	u, err := state.GetCurrentURLsPtr()
	if err != nil {
		util.GetLogger().Infoln(err)
	}
	util.GetLogger().Infoln(u.Urls)

	shortURLsChan := domain.NewMutexChanString(make(chan domain.URLWithID, 10))
	var once sync.Once

	r.Post("/", WrapHandler(uh.CreateShortened))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatch))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))
	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(shortURLsChan, &once)))

	return r
}*/

func exampleRequest(ts *httptest.Server, body, method, path, contentType string) (int, string) {
	util.GetLogger().Infoln("a")

	var req *http.Request
	var err error
	util.GetLogger().Infoln(method)
	if body == "" {
		req, err = http.NewRequest(method, ts.URL+path, nil)
	} else {
		req, err = http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		util.GetLogger().Infoln(req)
	}
	if err != nil {
		util.GetLogger().Infoln(err)
		return 0, ""
	}

	util.GetLogger().Infoln(req)
	req.Header.Set("Content-Type", contentType)

	resp, err := ts.Client().Do(req)
	util.GetLogger().Infoln(resp)
	if err != nil && err != http.ErrUseLastResponse {
		util.GetLogger().Infoln(err)
		return resp.StatusCode, ""
	}

	if resp.StatusCode != http.StatusNoContent {
		defer resp.Body.Close()
	}

	var respBody []byte

	if contentType != "application/json" && resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
	} else if path == "/api/shorten" {
		var short = struct {
			Result string `json:"result"`
		}{}

		err = json.NewDecoder(resp.Body).Decode(&short)
		respBody = []byte(short.Result)
	} else if path == "/api/shorten/batch" {
		var batch = []struct {
			Correlation string `json:"correlation_id"`
			Short       string `json:"short_url"`
		}{}

		err = json.NewDecoder(resp.Body).Decode(&batch)
		respBody = []byte(batch[0].Correlation + " " + batch[0].Short)
	}
	util.GetLogger().Infoln("be")

	if err != nil {
		util.GetLogger().Infoln(err)
		return resp.StatusCode, ""
	}

	return resp.StatusCode, string(respBody)
}
