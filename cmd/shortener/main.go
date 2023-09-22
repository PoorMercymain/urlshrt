package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"

	_ "net/http/pprof"

	"github.com/go-chi/chi/v5"
	mdlwr "github.com/go-chi/chi/v5/middleware"

	"github.com/PoorMercymain/urlshrt/internal/config"
	"github.com/PoorMercymain/urlshrt/internal/domain"
	"github.com/PoorMercymain/urlshrt/internal/handler"
	"github.com/PoorMercymain/urlshrt/internal/middleware"
	"github.com/PoorMercymain/urlshrt/internal/repository"
	"github.com/PoorMercymain/urlshrt/internal/service"
	"github.com/PoorMercymain/urlshrt/internal/state"
	"github.com/PoorMercymain/urlshrt/pkg/util"
)

var (
	buildVersion, buildDate, buildCommit string
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

	shortURLsChan := domain.NewMutexChanString(make(chan domain.URLWithID, 10))
	var once sync.Once

	r.Post("/", WrapHandler(uh.CreateShortened))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON))
	r.Get("/ping", WrapHandler(uh.PingPg))
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatch))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))
	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(shortURLsChan, &once)))
	r.Mount("/debug", mdlwr.Profiler())

	return r
}

func WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return middleware.GzipHandle(middleware.Authorize(middleware.WithLogging(h)))
}

func printGlobalVariable(variable string, shortDescription string) {
	if variable != "" {
		fmt.Println("Build", shortDescription+":", variable)
	} else {
		fmt.Println("Build", shortDescription+": N/A")
	}
}

func main() {
	printGlobalVariable(buildVersion, "version")
	printGlobalVariable(buildDate, "date")
	printGlobalVariable(buildCommit, "commit")

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
		var pgPtr *sql.DB
		pgPtr, err = pg.GetPgPtr()
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

	c := make(chan os.Signal, 1)
	ret := make(chan struct{}, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		ret <- struct{}{}
	}()

	go func() {
		err = http.ListenAndServe(addrToServe, r)
		if err != nil {
			util.GetLogger().Error(err)
			ret <- struct{}{}
		}
	}()

	<-ret
}
