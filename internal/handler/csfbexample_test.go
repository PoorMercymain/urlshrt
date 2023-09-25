package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/go-chi/chi/v5"
)

func routerExampleCreateShortenedFromBatch() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	var wg sync.WaitGroup
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatchAdapter(&wg)))

	return r
}

func ExampleURL_CreateShortenedFromBatchAdapter() {
	ts := httptest.NewServer(routerExampleCreateShortenedFromBatch())
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/shorten/batch", "[{\"correlation_id\": \"1\",\"original_url\": \"https://ya.ru\"}]", "application/json", http.MethodPost},
	}

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 201
}
