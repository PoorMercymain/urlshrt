package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/handler"
	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/repository"
	"github.com/PoorMercymain/urlshrt/internal/service"
	"github.com/PoorMercymain/urlshrt/internal/state"
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

	//url := domain.URL{}

	//urls := make([]domain.URLStringJSON, 1)

	r := chi.NewRouter()

	//fmt.Println(len(os.Args))

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

	//db := domain.NewDB("json", conf.JSONFile)

	err := middleware.InitLogger()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}

	defer middleware.GetLogger().Sync()

	/*urls, errDB := db.GetUrls()
	if errDB != nil {
		domain.GetLogger().Infoln(errDB)
		urls = make([]domain.URLStringJSON, 1)
	}*/

	//mut := new(sync.Mutex)

	ur := repository.NewURL(conf.JSONFile)
	us := service.NewURL(ur)
	uh := handler.NewURL(us)

	urls, err := ur.ReadAll(context.Background())
	if err != nil {
		middleware.GetLogger().Infoln("init", err)
		urls = make([]state.URLStringJSON, 1)
	}

	state.InitCurrentURLs(&urls)

	state.InitShortAddress(conf.ShortAddr.Addr)

	//data := domain.NewData(&urls, conf.ShortAddr.Addr, time.Now().Unix(), db, "", false, mut)
	//getData := domain.NewData(&urls, "", 0, db, "", false)

	r.Post("/", middleware.GzipHandle(http.HandlerFunc(uh.CreateShortened)))
	r.Get("/{short}", middleware.GzipHandle(http.HandlerFunc(uh.ReadOriginal)))
	r.Post("/api/shorten", middleware.GzipHandle(http.HandlerFunc(uh.CreateShortenedFromJSON)))

	middleware.GetLogger().Infoln(conf)
	addrToServe := strings.TrimPrefix(conf.HTTPAddr.Addr, "http://")
	addrToServe = strings.TrimSuffix(addrToServe, "/")
	err = http.ListenAndServe(addrToServe, r)
	if err != nil {
		middleware.GetLogger().Error(err)
		return
	}
}
