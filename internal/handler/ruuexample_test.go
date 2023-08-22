package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

func routerExampleReadUserURLs() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))

	return r
}

func ExampleURL_ReadUserURLs() {
	ts := httptest.NewServer(routerExampleReadUserURLs())
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/user/urls", "", "", http.MethodGet},
	}

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 200
}
