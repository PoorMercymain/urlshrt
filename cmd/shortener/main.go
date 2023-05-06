package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/go-chi/chi/v5"
)

func main() {
	var conf config.Config

	HTTPEnv, HTTPSet := os.LookupEnv("SERVER_ADDRESS")
	ShortEnv, ShortSet := os.LookupEnv("BASE_URL")
	//conf.HTTPAddr = config.AddrWithCheck{Addr: }

	if !HTTPSet {
		flag.Var(&conf.HTTPAddr, "a", "адрес http-сервера")
	}
	if !ShortSet {
		flag.Var(&conf.ShortAddr, "b", "базовый адрес сокращенного URL")
	}

	url := domain.URL{}

	urls := make([]domain.URL, 0)

	r := chi.NewRouter()
	
	if !HTTPSet || !ShortSet {
		flag.Parse()
	}

	if HTTPSet {
		conf.HTTPAddr = config.AddrWithCheck{Addr: HTTPEnv, WasSet: true}
	}

	if ShortSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: ShortEnv, WasSet: true}
	}

	if !conf.HTTPAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: "localhost:8080", WasSet: true}
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.HTTPAddr.WasSet {
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HTTPAddr
	}

	r.Post("/", url.GenerateShortURLHandler(urls, conf.ShortAddr.Addr))
	r.Get("/{short}", url.GetOriginalURLHandler(urls))

	fmt.Println(conf)
	err := http.ListenAndServe(conf.HTTPAddr.Addr, r)
    if err != nil {
        fmt.Println(err)
		return
    }
}
