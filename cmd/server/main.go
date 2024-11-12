package main

import (
	"context"
	"flag"
	"nearby-friends/cache"
	"nearby-friends/db"
	"nearby-friends/server"
	//"nearby-friends/types"
	"net/http"

	"go.uber.org/zap"
)

var debug bool

func main() {
	serverInfo := server.Info{}

	flag.BoolVar(&debug, "debug", true, "Enable debug logging")

	flag.StringVar(&serverInfo.CACertPath, "caCert", "", "Path to the CA Cert file")
	flag.StringVar(&serverInfo.CAKeyPath, "caKey", "", "Path to the CA Key file")
	flag.StringVar(&serverInfo.Host, "srvhost", "", "Server host")
	flag.StringVar(&serverInfo.Port, "srvport", "8080", "Server port")

	dbInfo := db.ConnInfo{}
	flag.StringVar(&dbInfo.Hostname, "dbhost", "mysql", "Database host")
	flag.StringVar(&dbInfo.Username, "dbuser", "root", "Database username")
	flag.StringVar(&dbInfo.Password, "dbpassword", "admin", "Database password")
	flag.StringVar(&dbInfo.DBName, "dbname", "user", "Database name")

	cacheInfo := cache.ConnInfo{}
	flag.StringVar(&cacheInfo.Host, "cachehost", "redis", "Cache host")
	flag.StringVar(&cacheInfo.Port, "cacheport", "6379", "Cache port")
	flag.StringVar(&cacheInfo.Username, "cacheuser", "", "Cache username")
	flag.StringVar(&cacheInfo.Password, "cachepassword", "", "Cache password")
	flag.IntVar(&cacheInfo.DB, "cacheDB", 0, "Cache Database")

	pubSubInfo := cache.ConnInfo{}
	flag.StringVar(&pubSubInfo.Host, "pubsubhost", "redis", "PubSub host")
	flag.StringVar(&pubSubInfo.Port, "pubsubport", "6379", "PubSub port")
	flag.StringVar(&pubSubInfo.Username, "pubsubuser", "", "PubSub username")
	flag.StringVar(&pubSubInfo.Password, "pubsubpassword", "", "PubSub password")
	flag.IntVar(&pubSubInfo.DB, "pubsubDB", 0, "Pubsub Database")

	flag.Parse()

	background := context.Background()

	c := zap.NewProductionConfig()
	if debug {
		c.Level.SetLevel(zap.DebugLevel)
	}
	log, _ := c.Build()
	defer log.Sync() // flushes buffer, if any

	slog := log.Sugar()
	db, err := db.NewDBHandler(db.MySQL, dbInfo, log)
	if err != nil {
		slog.Fatalf("error creating new DB handler: %v", err)
	}

	userCache, err := cache.NewCacheHandler(background, cache.RedisCache, cacheInfo, log)
	if err != nil {
		slog.Fatalf("error creating new cache handler: %v", err)
	}

	userPubSub, err := cache.NewPubSubHandler(background, cache.RedisPubSub, pubSubInfo, log)
	if err != nil {
		slog.Fatalf("error creating new pubsub handler: %v", err)
	}

	slog.Infof("Server to run on %v", serverInfo.Addr())
	handler := server.NewRequestHandler(background, db, userCache, userPubSub, log)
	if serverInfo.CACertPath != "" && serverInfo.CAKeyPath != "" {
		if err := http.ListenAndServeTLS(serverInfo.Addr(), serverInfo.CACertPath, serverInfo.CAKeyPath, handler.WithMiddleware()); err != nil {
			slog.Fatal(err)
		}
	} else {
		if err := http.ListenAndServe(serverInfo.Addr(), handler.WithMiddleware()); err != nil {
			slog.Fatal(err)
		}
	}
}
