package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/go-chi/chi/v5"
)

func main() {
	var conf config.Config

	httpEnv, httpSet := os.LookupEnv("SERVER_ADDRESS")
	shortEnv, shortSet := os.LookupEnv("BASE_URL")

	if !httpSet {
		flag.Var(&conf.HTTPAddr, "a", "адрес http-сервера")
	}
	if !shortSet {
		flag.Var(&conf.ShortAddr, "b", "базовый адрес сокращенного URL")
	}

	url := domain.URL{}

	urls := make([]domain.URL, 1)

	r := chi.NewRouter()

	if !httpSet || !shortSet {
		flag.Parse()
	}

	if httpSet {
		conf.HTTPAddr = config.AddrWithCheck{Addr: httpEnv, WasSet: true}
	}

	if shortSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: shortEnv, WasSet: true}
	}

	if !conf.HTTPAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: "localhost:8080", WasSet: true}
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.HTTPAddr.WasSet {
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HTTPAddr
	}

	db := domain.NewDB("txt", "testTxtDB.txt")

	r.Post("/", url.GenerateShortURLHandler(&urls, conf.ShortAddr.Addr, time.Now().Unix(), db))
	r.Get("/{short}", url.GetOriginalURLHandler(urls, db))

	fmt.Println(conf)
	err := http.ListenAndServe(conf.HTTPAddr.Addr, r)
	if err != nil {
		fmt.Println(err)
		return
	}
}
