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
	"go.uber.org/zap"
)

func main() {
	var conf config.Config

	httpEnv, httpSet := os.LookupEnv("SERVER_ADDRESS")
	shortEnv, shortSet := os.LookupEnv("BASE_URL")
	jsonFileEnv, jsonFileSet := os.LookupEnv("FILE_STORAGE_PATH")

	fmt.Println("serv", httpEnv, httpSet, "out", shortEnv, shortSet)

	var buf *string

	flag.Var(&conf.HTTPAddr, "a", "адрес http-сервера")

	flag.Var(&conf.ShortAddr, "b", "базовый адрес сокращенного URL")

	buf = flag.String("f", "./tmp/short-url-db.json", "полное имя файла, куда сохраняются данные в формате JSON")

	url := domain.URL{}

	urls := make([]domain.JSONDatabaseStr, 1)

	r := chi.NewRouter()

	fmt.Println(len(os.Args))

	if !httpSet || !shortSet || !jsonFileSet {
		flag.Parse()
	}

	fmt.Println(len(os.Args))

	conf.JSONFile = *buf

	if httpSet {
		conf.HTTPAddr = config.AddrWithCheck{Addr: httpEnv, WasSet: true}
	}

	if shortSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: shortEnv, WasSet: true}
	}

	if jsonFileSet {
		conf.JSONFile = jsonFileEnv
	}

	if !conf.HTTPAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: "localhost:8080", WasSet: true}
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.HTTPAddr.WasSet {
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HTTPAddr
	}

	fmt.Println(conf.JSONFile)

	db := domain.NewDB("json", conf.JSONFile)

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logger.Sync()

	sugar := *logger.Sugar()

	postContext := domain.NewContext(&urls, conf.ShortAddr.Addr, time.Now().Unix(), db, "", false)
	getContext := domain.NewContext(&urls, "", 0, db, "", false)

	r.Post("/", domain.GzipHandle(url.GenerateShortURLHandler(*postContext), &sugar))
	r.Get("/{short}", domain.GzipHandle(url.GetOriginalURLHandler(*getContext), &sugar))
	r.Post("/api/shorten", domain.GzipHandle(url.GenerateShortURLFromJSONHandler(*postContext), &sugar))

	fmt.Println(conf)
	err = http.ListenAndServe(conf.HTTPAddr.Addr, r)
	if err != nil {
		fmt.Println(err)
		return
	}
}
