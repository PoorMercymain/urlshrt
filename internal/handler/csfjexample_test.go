package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

func routerExampleCreateShortenedFromJSON() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))

	return r
}

func ExampleUrl_CreateShortenedFromJSON() {
	ts := httptest.NewServer(routerExampleCreateShortenedFromJSON())
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/shorten", "{\"url\":\"https://ya.ru\"}", "application/json", http.MethodPost},
	}

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 201
}