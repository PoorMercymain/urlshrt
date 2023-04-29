package main

import (
	"fmt"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/domain"
)

func main() {
	url := domain.Url{}

	mux := http.NewServeMux()
	mux.Handle(`/`, http.HandlerFunc(url.ShortenUrlHandler))

	err := http.ListenAndServe(":8080", mux)
    if err != nil {
        fmt.Println(err)
		return
    }
}
