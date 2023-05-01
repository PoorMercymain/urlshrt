package main

import (
	"fmt"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/go-chi/chi/v5"
)

func main() {
	url := domain.URL{}

	urls := make([]domain.URL, 0)

	r := chi.NewRouter()

	r.Post("/", url.GenerateShortURLHandler(urls))
	r.Get("/{short}", url.GetOriginalURLHandler(urls))

	err := http.ListenAndServe(":8080", r)
    if err != nil {
        fmt.Println(err)
		return
    }
}
