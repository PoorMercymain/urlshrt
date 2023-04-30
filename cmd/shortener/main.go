package main

import (
	"fmt"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/domain"
)

func main() {
	url := domain.URL{}

	urls := make([]domain.URL, 0)

	mux := http.NewServeMux()
	mux.Handle(`/`, http.HandlerFunc(url.ShortenURLHandler(urls)))

	err := http.ListenAndServe(":8080", mux)
    if err != nil {
        fmt.Println(err)
		return
    }
}
