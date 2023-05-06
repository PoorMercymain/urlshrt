package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/go-chi/chi/v5"
)

func main() {
	var conf config.Config

	flag.Var(&conf.HttpAddr, "a", "адрес http-сервера")
	flag.Var(&conf.ShortAddr, "b", "базовый адрес сокращенного URL")

	url := domain.URL{}

	urls := make([]domain.URL, 0)

	r := chi.NewRouter()

	flag.Parse()

	if !conf.HttpAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: "localhost:8080", WasSet: true}
		conf.HttpAddr = conf.ShortAddr
	} else if !conf.HttpAddr.WasSet {
		conf.HttpAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HttpAddr
	}

	r.Post("/", url.GenerateShortURLHandler(urls, conf.ShortAddr.Addr))
	r.Get("/{short}", url.GetOriginalURLHandler(urls))

	fmt.Println(conf)
	err := http.ListenAndServe(conf.HttpAddr.Addr, r)
    if err != nil {
        fmt.Println(err)
		return
    }
}
