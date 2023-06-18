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
	"github.com/PoorMercymain/urlshrt/pkg/util"
	"github.com/go-chi/chi/v5"
)

func router(pathToRepo string, pg *state.Postgres) chi.Router {
	ur := repository.NewURL(pathToRepo, pg)
	us := service.NewURL(ur)
	uh := handler.NewURL(us)

	urls, err := ur.ReadAll(context.Background())
	if err != nil {
		util.GetLogger().Infoln("init", err)
		urls = make([]state.URLStringJSON, 1)
	}

	urlsMap := make(map[string]state.URLStringJSON)
	for _, u := range urls {
		urlsMap[u.OriginalURL] = u
	}

	state.InitCurrentURLs(&urlsMap)

	r := chi.NewRouter()

	r.Post("/", WrapHandler(uh.CreateShortened))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))
	r.Get("/ping", WrapHandler(uh.PingPg))
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatch))

	return r
}

func WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return middleware.GzipHandle(middleware.WithLogging(h))
}

func main() {
	var conf config.Config

	httpEnv, httpSet := os.LookupEnv("SERVER_ADDRESS")
	shortEnv, shortSet := os.LookupEnv("BASE_URL")
	jsonFileEnv, jsonFileSet := os.LookupEnv("FILE_STORAGE_PATH")
	dsnEnv, dsnSet := os.LookupEnv("DATABASE_DSN")

	fmt.Println("serv", httpEnv, httpSet, "out", shortEnv, shortSet)

	var buf *string

	flag.Var(&conf.HTTPAddr, "a", "http server address")

	flag.Var(&conf.ShortAddr, "b", "base address of the shortened URL")

	dsnBuf := flag.String("d", "", "string to connect to database")

	buf = flag.String("f", "./tmp/short-url-db.json", "full name of file where to store URL data in JSON format")

	if !httpSet || !shortSet || !jsonFileSet || !dsnSet {
		flag.Parse()
	}

	fmt.Println(len(os.Args))

	conf.JSONFile = *buf
	conf.DSN = *dsnBuf

	if httpSet {
		conf.HTTPAddr = config.AddrWithCheck{Addr: httpEnv, WasSet: true}
	}

	if shortSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: shortEnv, WasSet: true}
	}

	if jsonFileSet {
		conf.JSONFile = jsonFileEnv
	}

	if dsnSet {
		conf.DSN = dsnEnv
	}

	pg := &state.Postgres{}
	var err error
	
	if conf.DSN != "" {
		pg, err = state.NewPG(conf.DSN)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(pg)
		pgPtr, err := pg.GetPgPtr()
		if err != nil {
			fmt.Println(err)
		}
		defer pgPtr.Close()
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

	err = util.InitLogger()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}

	defer util.GetLogger().Sync()

	util.GetLogger().Infoln("dsn", conf.DSN)

	state.InitShortAddress(conf.ShortAddr.Addr)

	r := router(conf.JSONFile, pg)

	util.GetLogger().Infoln(conf)
	addrToServe := strings.TrimPrefix(conf.HTTPAddr.Addr, "http://")
	addrToServe = strings.TrimSuffix(addrToServe, "/")
	err = http.ListenAndServe(addrToServe, r)
	if err != nil {
		util.GetLogger().Error(err)
		return
	}
}
