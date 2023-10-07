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
	"strconv"
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

func defineFlags(conf *config.Config) {
	flag.Var(&conf.HTTPAddr, "a", "http server address")

	flag.Var(&conf.ShortAddr, "b", "base address of the shortened URL")

	flag.StringVar(&conf.DSN, "d", "", "string to connect to database")

	flag.StringVar(&conf.JSONFile, "f", "./tmp/short-url-db.json", "full name of file where to store URL data in JSON format")

	flag.BoolVar(&conf.HTTPSEnabled, "s", false, "turns https on if not set to false")

	flag.StringVar(&conf.ConfigFilePath, "c", "", "config file path")
}

func main() {
	const (
		defaultFileStorage = "./tmp/short-url-db.json"
		HTTPPrefix         = "http://"
		HTTPSPrefix        = "https://"
		slash              = "/"
	)

	var (
		serverAddressEnvName   = "SERVER_ADDRESS"
		baseURLEnvName         = "BASE_URL"
		fileStoragePathEnvName = "FILE_STORAGE_PATH"
		databaseDSNEnvName     = "DATABASE_DSN"
		enableHTTPSEnvName     = "ENABLE_HTTPS"
		configFileEnvName      = "CONFIG"
	)

	util.PrintVariable(buildVersion, "version")
	util.PrintVariable(buildDate, "date")
	util.PrintVariable(buildCommit, "commit")

	err := util.InitLogger()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}

	var conf config.Config

	var configWithNamesPath string

	configWithNamesEnv, configWithNamesSet := os.LookupEnv(configFileEnvName)
	if !configWithNamesSet {
		defineFlags(&conf)
		flag.Parse()
		configWithNamesPath = conf.ConfigFilePath
	} else {
		configWithNamesPath = configWithNamesEnv
	}

	// struct to redefine default env variables names
	var configWithNames struct {
		JSONFileEnvName     string `json:"file_storage_path_env,omitempty"`
		DSNEnvName          string `json:"database_dsn_env,omitempty"`
		HTTPAddrEnvName     string `json:"server_address_env,omitempty"`
		ShortAddrEnvName    string `json:"base_url_env,omitempty"`
		HTTPSEnabledEnvName string `json:"enable_https_env,omitempty"`
		ConfigEnvName       string `json:"config_env,omitempty"`
	}

	if configWithNamesPath != "" {
		file, err := os.Open(configWithNamesPath)
		if err != nil {
			util.GetLogger().Infoln("Error opening file:", err)
			return
		}

		var content strings.Builder
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			content.Write(scanner.Bytes())
		}

		if err := scanner.Err(); err != nil {
			util.GetLogger().Infoln("Error reading file:", err)
			return
		}

		err = json.Unmarshal([]byte(content.String()), &configWithNames)
		if err != nil {
			util.GetLogger().Infoln("Error unmarshalling JSON:", err)
			return
		}
		file.Close()

		if configWithNames.ConfigEnvName != "" {
			configFileEnvName = configWithNames.ConfigEnvName
		}

		if configWithNames.DSNEnvName != "" {
			databaseDSNEnvName = configWithNames.DSNEnvName
		}

		if configWithNames.HTTPAddrEnvName != "" {
			serverAddressEnvName = configWithNames.HTTPAddrEnvName
		}

		if configWithNames.JSONFileEnvName != "" {
			fileStoragePathEnvName = configWithNames.JSONFileEnvName
		}

		if configWithNames.ShortAddrEnvName != "" {
			baseURLEnvName = configWithNames.ShortAddrEnvName
		}

		if configWithNames.HTTPSEnabledEnvName != "" {
			enableHTTPSEnvName = configWithNames.HTTPSEnabledEnvName
		}
	}

	// getting values of environment variables
	httpEnv, httpSet := os.LookupEnv(serverAddressEnvName)
	shortEnv, shortSet := os.LookupEnv(baseURLEnvName)
	jsonFileEnv, jsonFileSet := os.LookupEnv(fileStoragePathEnvName)
	dsnEnv, dsnSet := os.LookupEnv(databaseDSNEnvName)
	secureEnv, secureSet := os.LookupEnv(enableHTTPSEnvName)
	configEnv, configSet := os.LookupEnv(configFileEnvName)

	var boolSecureEnv bool
	if secureSet {
		// parsing value because os.LookupEnv returns a string, not a bool
		boolSecureEnv, err = strconv.ParseBool(secureEnv)
		if err != nil {
			util.GetLogger().Infoln(err)
			return
		}
	}

	util.GetLogger().Debugln("serv", httpEnv, httpSet, "out", shortEnv, shortSet)

	// if a value was set by environment variable, we have to redefine values in config because it was set by flags before
	if httpSet {
		conf.HTTPAddr = config.AddrWithCheck{Addr: httpEnv, WasSet: true}
		util.GetLogger().Infoln(conf.HTTPAddr)
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
		conf.HTTPSEnabled = boolSecureEnv
	}

	if configSet {
		conf.ConfigFilePath = configEnv
	}

	// required names of settings in a config file are not the same as in config struct, so we need another one which is rawConfig
	var rawConfig struct {
		JSONFile     string `json:"file_storage_path,omitempty"`
		DSN          string `json:"database_dsn,omitempty"`
		HTTPAddr     string `json:"server_address,omitempty"`
		ShortAddr    string `json:"base_url,omitempty"`
		HTTPSEnabled bool   `json:"enable_https,omitempty"`
	}

	if conf.ConfigFilePath != "" {
		file, err := os.Open(conf.ConfigFilePath)
		if err != nil {
			util.GetLogger().Infoln("Error opening file:", err)
			return
		}
		defer file.Close()

		var content strings.Builder
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			content.Write(scanner.Bytes())
		}

		if err := scanner.Err(); err != nil {
			util.GetLogger().Infoln("Error reading file:", err)
			return
		}

		err = json.Unmarshal([]byte(content.String()), &rawConfig)
		if err != nil {
			util.GetLogger().Infoln("Error unmarshalling JSON:", err)
			return
		}

		// if a variable is empty or has a default value (which means it was not set by both flags and environment variables)
		// then we need to use values from config file
		if conf.HTTPAddr.Addr == "" {
			set := true
			if rawConfig.HTTPAddr == "" {
				set = false
			}

			util.GetLogger().Debugln("=====", set)
			conf.HTTPAddr = config.AddrWithCheck{Addr: rawConfig.HTTPAddr, WasSet: set}
		}

		if conf.ShortAddr.Addr == "" {
			set := true
			if rawConfig.ShortAddr == "" {
				set = false
			}

			conf.ShortAddr = config.AddrWithCheck{Addr: rawConfig.ShortAddr, WasSet: set}
		}

		if (conf.JSONFile == defaultFileStorage || conf.JSONFile == "") && rawConfig.JSONFile != "" {
			conf.JSONFile = rawConfig.JSONFile
		}

		if conf.DSN == "" {
			conf.DSN = rawConfig.DSN
		}

		if !conf.HTTPSEnabled {
			conf.HTTPSEnabled = rawConfig.HTTPSEnabled
		}
	}

	// creating a postgres struct
	pg := &state.Postgres{}

	if conf.DSN != "" {
		pg, err = state.NewPG(conf.DSN)
		if err != nil {
			util.GetLogger().Infoln(err)
		}
		util.GetLogger().Debugln(pg)
		var pgPtr *sql.DB
		pgPtr, err = pg.GetPgPtr()
		if err != nil {
			util.GetLogger().Infoln(err)
		}
		defer pgPtr.Close()
	}

	// if address were not specified, we may need to use a default address which is different for HTTP and HTTPS
	defAddr := "://localhost:"
	if conf.HTTPSEnabled {
		defAddr = fmt.Sprintf("https%s443/", defAddr)
	} else {
		defAddr = fmt.Sprintf("http%s8080/", defAddr)
	}

	// if both addresses were not set, we just use the default one
	// if only one address were set, its value is used for another address too
	if !conf.HTTPAddr.WasSet && !conf.ShortAddr.WasSet {
		conf.ShortAddr = config.AddrWithCheck{Addr: defAddr, WasSet: true}
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.HTTPAddr.WasSet {
		conf.HTTPAddr = conf.ShortAddr
	} else if !conf.ShortAddr.WasSet {
		conf.ShortAddr = conf.HTTPAddr
	}

	util.GetLogger().Debugln(conf.JSONFile)

	defer func() {
		err = util.GetLogger().Sync()
		if err != nil {
			return
		}
	}()

	util.GetLogger().Debugln("dsn", conf.DSN)

	// initializing base address of short URLs
	state.InitShortAddress(conf.ShortAddr.Addr)

	// WaitGroup is required for handlers which can create goroutines working in
	// background without using network, so we could wait for them to shut down gracefully
	var wg sync.WaitGroup

	r := router(conf.JSONFile, pg, &wg)

	var m *autocert.Manager

	const cacheDirPath = ".cache"
	const defaultHTTPS01ChallengeServer = ":80"

	// for HTTPs certificates are required, so we are setting up autocert manager and a handler for HTTPS 01 challenge
	if conf.HTTPSEnabled {
		m = &autocert.Manager{
			Cache:  autocert.DirCache(cacheDirPath),
			Prompt: autocert.AcceptTOS,
		}

		go func() {
			h := m.HTTPHandler(nil)
			fmt.Println(http.ListenAndServe(defaultHTTPS01ChallengeServer, h))
		}()
	}

	util.GetLogger().Debugln(conf)

	addrToServe := strings.TrimPrefix(conf.HTTPAddr.Addr, HTTPPrefix)
	addrToServe = strings.TrimPrefix(addrToServe, HTTPSPrefix)
	addrToServe = strings.TrimSuffix(addrToServe, slash)

	util.GetLogger().Infoln(addrToServe)
	server := http.Server{
		Addr:    addrToServe,
		Handler: r,
	}

	// channel to intercept signals for graceful shutdown
	c := make(chan os.Signal, 1)
	ret := make(chan struct{}, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		<-c
		ret <- struct{}{}
	}()

	// certificate and key paths (you should get them before starting the service,
	// but if you use provided makefile, it shall get them for you)
	const certPath = "cert/localhost.crt"
	const keyPath = "cert/localhost.key"
	go func() {
		if conf.HTTPSEnabled {
			err = server.ListenAndServeTLS(certPath, keyPath)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil {
			util.GetLogger().Error(err)
			ret <- struct{}{}
		}
	}()

	// waiting for a signal for shutting down or an error to occur
	<-ret

	start := time.Now()

	const timeoutInterval = 5 * time.Second

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	util.GetLogger().Debugln("дошел до shutdown")

	// shutting down gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		util.GetLogger().Infoln("shutdown:", err)
		return
	} else {
		cancel()
	}

	util.GetLogger().Debugln("прошел shutdown")

	// waiting for goroutines which are not using network to finish their jobs
	wg.Wait()

	// waiting for shutdown to finish
	<-shutdownCtx.Done()
	util.GetLogger().Debugln("shutdownCtx done:", shutdownCtx.Err().Error())

	util.GetLogger().Debugln(time.Since(start))
}
