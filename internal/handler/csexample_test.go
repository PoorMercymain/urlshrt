package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

func routerExample() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	r.Post("/", WrapHandler(uh.CreateShortened))

	return r
}

func ExampleURL_CreateShortened() {
	ts := httptest.NewServer(routerExample())
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

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 201
	//201
	//201
	//201
}
