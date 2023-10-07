package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/go-chi/chi/v5"
)

func routerExampleDeleteUserURLs() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	var wg sync.WaitGroup

	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(nil, nil, &wg)))

	return r
}

func ExampleURL_DeleteUserURLsAdapter() {
	ts := httptest.NewServer(routerExampleDeleteUserURLs())
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/api/user/urls", "[\"GqKWdrE\"]", "application/json", http.MethodDelete},
	}

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 202
}
