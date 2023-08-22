package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

func routerExampleReadOriginal() chi.Router {
	us := GetExampleMockSrv()

	r := chi.NewRouter()

	uh := NewURL(us)

	r.Get("/{short}", WrapHandler(uh.ReadOriginal))

	return r
}

func ExampleUrl_ReadOriginal() {
	ts := httptest.NewServer(routerExampleReadOriginal())
	defer ts.Close()

	var testTable = []struct {
		url         string
		body        string
		contentType string
		method      string
	}{
		{"/GqKWdrE", "", "", http.MethodGet},
	}

	for j := range testTable {
		status, _ := exampleRequest(ts, testTable[j].body, testTable[j].method, testTable[j].url, testTable[j].contentType)
		fmt.Println(status)
	}
	// Output: 200
}