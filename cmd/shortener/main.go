package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/go-chi/chi/v5"
	mdlwr "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/acme/autocert"

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

func router(pathToRepo string, pg *state.Postgres, wg *sync.WaitGroup) chi.Router {
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
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatchAdapter(wg)))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs))
	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(shortURLsChan, &once, wg)))
	r.Mount("/debug", mdlwr.Profiler())

	return r
}

func WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return middleware.GzipHandle(middleware.Authorize(middleware.WithLogging(h)))
}

func main() {
	util.PrintVariable(buildVersion, "version")
	util.PrintVariable(buildDate, "date")
	util.PrintVariable(buildCommit, "commit")

	var conf config.Config

	httpEnv, httpSet := os.LookupEnv("SERVER_ADDRESS")
	shortEnv, shortSet := os.LookupEnv("BASE_URL")
	jsonFileEnv, jsonFileSet := os.LookupEnv("FILE_STORAGE_PATH")
	dsnEnv, dsnSet := os.LookupEnv("DATABASE_DSN")
	secureEnv, secureSet := os.LookupEnv("ENABLE_HTTPS")
	configEnv, configSet := os.LookupEnv("CONFIG")

	fmt.Println("serv", httpEnv, httpSet, "out", shortEnv, shortSet)

	var buf *string
	var httpsRequired *string
	var confFilePath *string

	flag.Var(&conf.HTTPAddr, "a", "http server address")

	flag.Var(&conf.ShortAddr, "b", "base address of the shortened URL")

	dsnBuf := flag.String("d", "", "string to connect to database")

	buf = flag.String("f", "./tmp/short-url-db.json", "full name of file where to store URL data in JSON format")

	httpsRequired = flag.String("s", "", "turn https on")

	confFilePath = flag.String("c", "", "config file path")

	if !httpSet || !shortSet || !jsonFileSet || !dsnSet || !secureSet || !configSet {
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

	if secureSet {
		httpsRequired = &secureEnv
	}

	if configSet {
		confFilePath = &configEnv
	}

	var rawConfig struct {
		JSONFile     string `json:"file_storage_path,omitempty"`
		DSN          string `json:"database_dsn,omitempty"`
		HTTPAddr     string `json:"server_address,omitempty"`
		ShortAddr    string `json:"base_url,omitempty"`
		HTTPSEnabled string `json:"enable_https,omitempty"`
	}

	if *confFilePath != "" {
		file, err := os.Open(*confFilePath)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		var content string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			content += scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		err = json.Unmarshal([]byte(content), &rawConfig)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			return
		}

		if conf.HTTPAddr.Addr == "" {
			set := true
			if rawConfig.HTTPAddr == "" {
				set = false
			}

			fmt.Println("=====", set)
			conf.HTTPAddr = config.AddrWithCheck{Addr: rawConfig.HTTPAddr, WasSet: set}
		}

		if conf.ShortAddr.Addr == "" {
			set := true
			if rawConfig.ShortAddr == "" {
				set = false
			}

			conf.ShortAddr = config.AddrWithCheck{Addr: rawConfig.ShortAddr, WasSet: set}
		}

		if conf.JSONFile == "" {
			conf.JSONFile = rawConfig.JSONFile
		}

		if conf.DSN == "" {
			conf.DSN = rawConfig.DSN
		}

		if *httpsRequired == "" {
			*httpsRequired = rawConfig.HTTPSEnabled
		}
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

	defAddr := "://localhost:"
	if *httpsRequired != "" {
		defAddr = "https" + defAddr + "443/"
	} else {
		defAddr = "http" + defAddr + "8080/"
	}

	if !conf.HTTPAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: defAddr, WasSet: true}
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

	defer func() {
		err = util.GetLogger().Sync()
		if err != nil {
			return
		}
	}()

	util.GetLogger().Infoln("dsn", conf.DSN)

	state.InitShortAddress(conf.ShortAddr.Addr)

	var wg sync.WaitGroup

	r := router(conf.JSONFile, pg, &wg)

	var m *autocert.Manager

	if *httpsRequired != "" {
		m = &autocert.Manager{
			Cache:  autocert.DirCache(".cache"),
			Prompt: autocert.AcceptTOS,
		}

		go func() {
			h := m.HTTPHandler(nil)
			fmt.Println(http.ListenAndServe(":80", h))
		}()
	}

	util.GetLogger().Infoln(conf)

	addrToServe := strings.TrimPrefix(conf.HTTPAddr.Addr, "http://")
	addrToServe = strings.TrimPrefix(addrToServe, "https://")
	addrToServe = strings.TrimSuffix(addrToServe, "/")

	server := http.Server{
		Addr:    addrToServe,
		Handler: r,
	}

	c := make(chan os.Signal, 1)
	ret := make(chan struct{}, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		<-c
		ret <- struct{}{}
	}()

	go func() {
		if *httpsRequired != "" {
			err = server.ListenAndServeTLS("cert/localhost.crt", "cert/localhost.key")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil {
			util.GetLogger().Error(err)
			ret <- struct{}{}
		}
	}()

	<-ret

	start := time.Now()

	timeoutInterval := 5 * time.Second

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	util.GetLogger().Infoln("дошел до shutdown")
	if err := server.Shutdown(shutdownCtx); err != nil {
		util.GetLogger().Infoln("shutdown:", err)
		return
	} else {
		cancel()
	}

	util.GetLogger().Infoln("прошел shutdown")

	wg.Wait()
	<-shutdownCtx.Done()
	util.GetLogger().Infoln("shutdownCtx done:", shutdownCtx.Err().Error())

	util.GetLogger().Infoln(time.Since(start))
}
