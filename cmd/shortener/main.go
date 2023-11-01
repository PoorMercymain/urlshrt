package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PoorMercymain/urlshrt/internal/interceptor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/PoorMercymain/urlshrt/pkg/api"

	_ "net/http/pprof"

	_ "google.golang.org/grpc/encoding/gzip"

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

func router(us *service.URL, ur *repository.URL, jwtKey string, CIDR string, shortURLsChan *domain.MutexChanString, wg *sync.WaitGroup, once *sync.Once) chi.Router {
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
	m, _ := state.GetCurrentURLsPtr()
	util.GetLogger().Infoln(m.Urls)

	r := chi.NewRouter()

	r.Post("/", WrapHandler(uh.CreateShortened, jwtKey))
	r.Get("/{short}", WrapHandler(uh.ReadOriginal, jwtKey))
	r.Post("/api/shorten", WrapHandler(uh.CreateShortenedFromJSON, jwtKey))
	r.Get("/ping", WrapHandler(uh.PingPg, jwtKey))
	r.Post("/api/shorten/batch", WrapHandler(uh.CreateShortenedFromBatchAdapter(wg), jwtKey))
	r.Get("/api/user/urls", WrapHandler(uh.ReadUserURLs, jwtKey))
	r.Delete("/api/user/urls", WrapHandler(uh.DeleteUserURLsAdapter(shortURLsChan, once, wg), jwtKey))
	r.Get("/api/internal/stats", middleware.CheckCIDR(WrapHandler(uh.ReadAmountOfURLsAndUsers, jwtKey), CIDR))
	r.Mount("/debug", mdlwr.Profiler())

	return r
}

func WrapHandler(h http.HandlerFunc, jwtKey string) http.HandlerFunc {
	return middleware.GzipHandle(middleware.Authorize(middleware.WithLogging(h), jwtKey))
}

func defineFlags(conf *config.Config) {
	flag.Var(&conf.HTTPAddr, "a", "http server address")

	flag.Var(&conf.ShortAddr, "b", "base address of the shortened URL")

	flag.StringVar(&conf.DSN, "d", "", "string to connect to database")

	flag.StringVar(&conf.JSONFile, "f", "", "full name of file where to store URL data in JSON format")

	flag.BoolVar(&conf.HTTPSEnabled, "s", false, "turns https on if not set to false")

	flag.StringVar(&conf.ConfigFilePath, "c", "", "config file path")

	flag.StringVar(&conf.TrustedSubnet, "t", "", "trusted subnet from which access for stats endpoint is not denied")

	flag.StringVar(&conf.JWTKey, "j", "", "key to generate JWTs and get info from them")

	flag.StringVar(&conf.GRPCAddr, "g", "", "gRPC server address")

	flag.BoolVar(&conf.GRPCSecureEnabled, "e", false, "turning secure gRPC on")

	flag.StringVar(&conf.GRPCFileStorage, "fg", "", "full name of file where to store URL data in JSON format to be used by gRPC server")

	flag.StringVar(&conf.GRPCDatabaseDSN, "dg", "", "string to connect to database of gRPC server")

	flag.StringVar(&conf.GRPCTrustedSubnet, "tg", "", "trusted subnet from which access for stats endpoint on gRPC server is not denied")

	flag.StringVar(&conf.GRPCJWTKey, "jg", "", "gRPC server key to generate JWTs and get info from them")
}

func main() {
	const (
		defaultFileStorage = "./tmp/short-url-db.json"
		defaultJWTKey      = "ultrasecretkey" // user should set a value to jwt key through config file/flag/env variable, if he won't then this unsafe value will be used
		HTTPPrefix         = "http://"
		HTTPSPrefix        = "https://"
		slash              = "/"
	)

	// default names of env variables
	var (
		serverAddressEnvName   = "SERVER_ADDRESS"
		baseURLEnvName         = "BASE_URL"
		fileStoragePathEnvName = "FILE_STORAGE_PATH"
		databaseDSNEnvName     = "DATABASE_DSN"
		enableHTTPSEnvName     = "ENABLE_HTTPS"
		configFileEnvName      = "CONFIG"
		trustedSubnetEnvName   = "TRUSTED_SUBNET"
		jwtKeyEnvName          = "JWT_KEY"

		// other options (not mentioned in this block) are shared with http/https server
		grpcAddressEnvName       = "GRPC_ADDRESS"
		enableGRPCSecureEnvName  = "ENABLE_SECURE_GRPC"
		grpcFileStorageEnvName   = "GRPC_FILE_STORAGE_PATH"
		grpcDatabaseDSNEnvName   = "GRPC_DSN"
		grpcTrustedSubnetEnvName = "GRPC_TRUSTED_SUBNET"
		grpcJWTKeyEnvName        = "GRPC_JWT_KEY"

		// these vars can be configured through config file (certificate and key paths too)
		defaultHTTPS01ChallengeServer = ":80"
		cacheDirPath                  = ".cache"
		defGRPCAddr                   = "localhost:4431"
		defAddr                       string

		// certificate and key paths (you should get them before starting the service,
		// but if you use provided makefile, it shall get them for you)
		keyPath  = "cert/localhost.key"
		certPath = "cert/localhost.crt"
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
		JSONFileEnvName            string `json:"file_storage_path_env,omitempty"`
		DSNEnvName                 string `json:"database_dsn_env,omitempty"`
		HTTPAddrEnvName            string `json:"server_address_env,omitempty"`
		ShortAddrEnvName           string `json:"base_url_env,omitempty"`
		HTTPSEnabledEnvName        string `json:"enable_https_env,omitempty"`
		ConfigEnvName              string `json:"config_env,omitempty"`
		TrustedSubnetEnvName       string `json:"trusted_subnet_env,omitempty"`
		JWTKeyEnvName              string `json:"jwt_key_env,omitempty"`
		GRPCAddressEnvName         string `json:"grpc_address_env,omitempty"`
		GRPCSecureEnabledEnvName   string `json:"grpc_secure_env,omitempty"`
		GRPCFileStoragePathEnvName string `json:"grpc_file_storage_path_env_name,omitempty"`
		GRPCDatabaseDSNEnvName     string `json:"grpc_database_dsn_env_name,omitempty"`
		GRPCTrustedSubnetEnvName   string `json:"grpc_trusted_subnet_env_name,omitempty"`
		GRPCJWTKeyEnvName          string `json:"grpc_jwt_key_env_name,omitempty"`
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

		if configWithNames.TrustedSubnetEnvName != "" {
			trustedSubnetEnvName = configWithNames.TrustedSubnetEnvName
		}

		if configWithNames.JWTKeyEnvName != "" {
			jwtKeyEnvName = configWithNames.JWTKeyEnvName
		}

		if configWithNames.GRPCAddressEnvName != "" {
			grpcAddressEnvName = configWithNames.GRPCAddressEnvName
		}

		if configWithNames.GRPCSecureEnabledEnvName != "" {
			enableGRPCSecureEnvName = configWithNames.GRPCSecureEnabledEnvName
		}

		if configWithNames.GRPCFileStoragePathEnvName != "" {
			grpcFileStorageEnvName = configWithNames.GRPCFileStoragePathEnvName
		}

		if configWithNames.GRPCDatabaseDSNEnvName != "" {
			grpcDatabaseDSNEnvName = configWithNames.GRPCDatabaseDSNEnvName
		}

		if configWithNames.GRPCTrustedSubnetEnvName != "" {
			grpcTrustedSubnetEnvName = configWithNames.GRPCTrustedSubnetEnvName
		}

		if configWithNames.GRPCJWTKeyEnvName != "" {
			grpcJWTKeyEnvName = configWithNames.GRPCJWTKeyEnvName
		}
	}

	// getting values of environment variables
	httpEnv, httpSet := os.LookupEnv(serverAddressEnvName)
	shortEnv, shortSet := os.LookupEnv(baseURLEnvName)
	jsonFileEnv, jsonFileSet := os.LookupEnv(fileStoragePathEnvName)
	dsnEnv, dsnSet := os.LookupEnv(databaseDSNEnvName)
	secureEnv, secureSet := os.LookupEnv(enableHTTPSEnvName)
	configEnv, configSet := os.LookupEnv(configFileEnvName)
	trustedSubnetEnv, trustedSubnetSet := os.LookupEnv(trustedSubnetEnvName)
	jwtKeyEnv, jwtKeySet := os.LookupEnv(jwtKeyEnvName)
	grpcEnv, grpcSet := os.LookupEnv(grpcAddressEnvName)
	grpcSecureEnv, grpcSecureSet := os.LookupEnv(enableGRPCSecureEnvName)
	grpcFileStorageEnv, grpcFileStorageSet := os.LookupEnv(grpcFileStorageEnvName)
	grpcDatabaseDSNEnv, grpcDatabaseDSNSet := os.LookupEnv(grpcDatabaseDSNEnvName)
	grpcTrustedSubnetEnv, grpcTrustedSubnetSet := os.LookupEnv(grpcTrustedSubnetEnvName)
	grpcJWTKeyEnv, grpcJWTKeySet := os.LookupEnv(grpcJWTKeyEnvName)

	var boolSecureEnv, boolSecureGRPCEnv bool
	if secureSet {
		// parsing value because os.LookupEnv returns a string, not a bool
		boolSecureEnv, err = strconv.ParseBool(secureEnv)
		if err != nil {
			util.GetLogger().Infoln(err)
			return
		}
	}

	if grpcSecureSet {
		boolSecureGRPCEnv, err = strconv.ParseBool(grpcSecureEnv)
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

	if trustedSubnetSet {
		conf.TrustedSubnet = trustedSubnetEnv
	}

	if jwtKeySet {
		conf.JWTKey = jwtKeyEnv
	}

	if grpcSet {
		conf.GRPCAddr = grpcEnv
	}

	if grpcSecureSet {
		conf.GRPCSecureEnabled = boolSecureGRPCEnv
	}

	if grpcFileStorageSet {
		conf.GRPCFileStorage = grpcFileStorageEnv
	}

	if grpcDatabaseDSNSet {
		conf.GRPCDatabaseDSN = grpcDatabaseDSNEnv
	}

	if grpcTrustedSubnetSet {
		conf.GRPCTrustedSubnet = grpcTrustedSubnetEnv
	}

	if grpcJWTKeySet {
		conf.GRPCJWTKey = grpcJWTKeyEnv
	}

	// required names of settings in a config file are not the same as in config struct, so we need another one which is rawConfig
	var rawConfig struct {
		JSONFile          string `json:"file_storage_path,omitempty"`
		DSN               string `json:"database_dsn,omitempty"`
		HTTPAddr          string `json:"server_address,omitempty"`
		ShortAddr         string `json:"base_url,omitempty"`
		HTTPSEnabled      bool   `json:"enable_https,omitempty"`
		TrustedSubnet     string `json:"trusted_subnet,omitempty"`
		JWTKey            string `json:"jwt_key,omitempty"`
		GRPCAddr          string `json:"grpc_address,omitempty"`
		GRPCSecureEnabled bool   `json:"enable_secure_grpc,omitempty"`
		GRPCFileStorage   string `json:"grpc_file_storage,omitempty"`
		GRPCDatabaseDSN   string `json:"grpc_dsn,omitempty"`
		GRPCTrustedSubnet string `json:"grpc_trusted_subnet,omitempty"`
		GRPCJWTKey        string `json:"grpc_jwt_key,omitempty"`

		DefaultHTTPS01ChallengeAddress string `json:"default_https_01_challenge_address"`
		CacheDirPath                   string `json:"cache_dir"`
		DefaultGRPCAddress             string `json:"default_grpc_address"`
		DefaultAddress                 string `json:"default_address"`
		CertificateKeyPath             string `json:"cert_key_path"`
		CertificatePath                string `json:"cert_path"`
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

		// may be now the check can be simplified
		if (conf.JSONFile == defaultFileStorage || conf.JSONFile == "") && rawConfig.JSONFile != "" {
			conf.JSONFile = rawConfig.JSONFile
		}

		if conf.DSN == "" {
			conf.DSN = rawConfig.DSN
		}

		if !conf.HTTPSEnabled {
			conf.HTTPSEnabled = rawConfig.HTTPSEnabled
		}

		if conf.TrustedSubnet == "" {
			conf.TrustedSubnet = rawConfig.TrustedSubnet
		}

		if conf.JWTKey == "" {
			conf.JWTKey = rawConfig.JWTKey
		}

		if conf.GRPCAddr == "" {
			conf.GRPCAddr = rawConfig.GRPCAddr
		}

		if !conf.GRPCSecureEnabled {
			conf.GRPCSecureEnabled = rawConfig.GRPCSecureEnabled
		}

		if rawConfig.DefaultHTTPS01ChallengeAddress != "" {
			defaultHTTPS01ChallengeServer = rawConfig.DefaultHTTPS01ChallengeAddress
		}

		if rawConfig.CacheDirPath != "" {
			cacheDirPath = rawConfig.CacheDirPath
		}

		if rawConfig.DefaultGRPCAddress != "" {
			defGRPCAddr = rawConfig.DefaultGRPCAddress
		}

		if rawConfig.DefaultAddress != "" {
			defAddr = rawConfig.DefaultAddress
		}

		if rawConfig.CertificateKeyPath != "" {
			keyPath = rawConfig.CertificateKeyPath
		}

		if rawConfig.CertificatePath != "" {
			certPath = rawConfig.CertificatePath
		}

		if conf.GRPCFileStorage == "" {
			conf.GRPCFileStorage = rawConfig.GRPCFileStorage
		}

		if conf.GRPCDatabaseDSN == "" {
			conf.GRPCDatabaseDSN = rawConfig.GRPCDatabaseDSN
		}

		if conf.GRPCTrustedSubnet == "" {
			conf.GRPCTrustedSubnet = rawConfig.GRPCTrustedSubnet
		}

		if conf.GRPCJWTKey == "" {
			conf.GRPCJWTKey = rawConfig.GRPCJWTKey
		}
	}

	if conf.JWTKey == "" {
		conf.JWTKey = defaultJWTKey
	}

	if conf.JSONFile == "" {
		conf.JSONFile = defaultFileStorage
	}

	if conf.GRPCFileStorage == "" {
		conf.GRPCFileStorage = conf.JSONFile
	}

	if conf.GRPCDatabaseDSN == "" {
		conf.GRPCDatabaseDSN = conf.DSN
	}

	if conf.GRPCTrustedSubnet == "" {
		conf.GRPCTrustedSubnet = conf.TrustedSubnet
	}

	if conf.GRPCJWTKey == "" {
		conf.GRPCJWTKey = conf.JWTKey
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
	if defAddr == "" {
		defAddr = "://localhost:"
		if conf.HTTPSEnabled {
			defAddr = fmt.Sprintf("https%s443/", defAddr)
		} else {
			defAddr = fmt.Sprintf("http%s8080/", defAddr)
		}
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

	if conf.GRPCAddr == "" {
		conf.GRPCAddr = defGRPCAddr
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
	var once sync.Once

	ur := repository.NewURL(conf.JSONFile, pg)
	us := service.NewURL(ur)

	var urGRPC *repository.URL
	var usGRPC *service.URL
	pgGRPC := &state.Postgres{}
	if conf.JSONFile == conf.GRPCFileStorage && conf.DSN == conf.GRPCDatabaseDSN {
		usGRPC = us
	} else {
		if conf.GRPCDatabaseDSN != "" {
			pgGRPC, err = state.NewPG(conf.GRPCDatabaseDSN)
			if err != nil {
				util.GetLogger().Infoln(err)
			}
			util.GetLogger().Debugln(pgGRPC)
			var pgPtr *sql.DB
			pgPtr, err = pgGRPC.GetPgPtr()
			if err != nil {
				util.GetLogger().Infoln(err)
			}
			defer pgPtr.Close()
		}

		urGRPC = repository.NewURL(conf.GRPCFileStorage, pgGRPC)
		usGRPC = service.NewURL(urGRPC)
	}

	shortURLsChan := domain.NewMutexChanString(make(chan domain.URLWithID, 10))
	r := router(us, ur, conf.JWTKey, conf.TrustedSubnet, shortURLsChan, &wg, &once)

	var m *autocert.Manager

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

	listenerGRPC, err := net.Listen("tcp", conf.GRPCAddr)
	if err != nil {
		util.GetLogger().Infoln("failed to listen:", err)
		return
	}

	var grpcServer *grpc.Server
	if conf.GRPCSecureEnabled {
		creds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
		if err != nil {
			log.Fatalf("Failed to setup tls: %v", err)
		}
		grpcServer = grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(interceptor.Log,
			interceptor.Authorize(conf.JWTKey), interceptor.CheckCIDR(conf.TrustedSubnet), interceptor.ValidateRequest))
	} else {
		grpcServer = grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.Log, interceptor.Authorize(conf.JWTKey),
			interceptor.CheckCIDR(conf.TrustedSubnet), interceptor.ValidateRequest))
	}

	urlshrtServer := &handler.Server{Wg: &wg, Once: &once, Srv: usGRPC, ShortURLsChan: shortURLsChan}
	api.RegisterUrlshrtV1Server(grpcServer, urlshrtServer)

	// channel to intercept signals for graceful shutdown
	c := make(chan os.Signal, 1)
	ret := make(chan struct{}, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		<-c
		ret <- struct{}{}
	}()

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

	go func() {
		ma, _ := state.GetCurrentURLsPtr()
		util.GetLogger().Infoln(ma.Urls)
		err = grpcServer.Serve(listenerGRPC)
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

	grpcServer.GracefulStop()
	// waiting for shutdown to finish
	<-shutdownCtx.Done()
	util.GetLogger().Debugln("shutdownCtx done:", shutdownCtx.Err().Error())

	util.GetLogger().Debugln(time.Since(start))
}
