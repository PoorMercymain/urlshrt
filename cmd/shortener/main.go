package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/go-chi/chi/v5"
)

func main() {
	var conf config.Config

	httpEnv, httpSet := os.LookupEnv("SERVER_ADDRESS")
	shortEnv, shortSet := os.LookupEnv("BASE_URL")
	jsonFileEnv, jsonFileSet := os.LookupEnv("FILE_STORAGE_PATH")

	fmt.Println("serv", httpEnv, httpSet, "out", shortEnv, shortSet)

	var buf *string

	flag.Var(&conf.HTTPAddr, "a", "http server address")

	flag.Var(&conf.ShortAddr, "b", "base address of the shortened URL")

	buf = flag.String("f", "./tmp/short-url-db.json", "full name of file where to store URL data in JSON format")

	url := domain.URL{}

	urls := make([]domain.URLStringJSON, 1)

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
		conf.ShortAddr = config.AddrWithCheck{Addr: "http://localhost:8080/", WasSet: true}
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.HTTPAddr.WasSet {
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HTTPAddr
	}

	fmt.Println(conf.JSONFile)

	db := domain.NewDB("json", conf.JSONFile)

	err := domain.InitLogger()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}

	defer domain.GetLogger().Sync()

	urls, errDB := db.GetUrls()
	if errDB != nil {
		domain.GetLogger().Infoln(errDB)
		urls = make([]domain.URLStringJSON, 1)
	}

	mut := new(sync.Mutex)

	data := domain.NewData(&urls, conf.ShortAddr.Addr, time.Now().Unix(), db, "", false, mut)
	//getData := domain.NewData(&urls, "", 0, db, "", false)

	r.Post("/", domain.GzipHandle(url.GenerateShortURLHandler(data)))
	r.Get("/{short}", domain.GzipHandle(url.GetOriginalURLHandler(data)))
	r.Post("/api/shorten", domain.GzipHandle(url.GenerateShortURLFromJSONHandler(data)))

	domain.GetLogger().Infoln(conf)
	addrToServe := strings.TrimPrefix(conf.HTTPAddr.Addr, "http://")
	addrToServe = strings.TrimSuffix(addrToServe, "/")
	err = http.ListenAndServe(addrToServe, r)
	if err != nil {
		domain.GetLogger().Error(err)
		return
	}
}
