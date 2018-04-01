// package main provides a command line interface for starting the eventdb REST API.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/oauth2"

	firebase "firebase.google.com/go"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	oauthFB "golang.org/x/oauth2/facebook"

	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/facebook"
	"github.com/findrandomevents/eventdb/log"
	"github.com/findrandomevents/eventdb/pg"
	"github.com/findrandomevents/eventdb/prom"
	"github.com/findrandomevents/eventdb/rest"
	"github.com/findrandomevents/eventdb/service"
)

func main() {
	var (
		adminUIDs         = flag.String("admin-uids", os.Getenv("ADMIN_UIDS"), "comma-separated list of firebase uids that have admin privileges")
		corsOrigins       = flag.String("cors-origins", "", "comma-seaprated list of request origins where CORS requests are allowed")
		dbURL             = flag.String("db", os.Getenv("DB"), "a database connection URL for the PostgreSQL database")
		environment       = flag.String("environment", os.Getenv("ENV"), "development or production, controls log verbosity")
		firebaseProjectID = flag.String("project-id", "the-third-party", "The firebase project-id used for auth")
		oauthID           = flag.String("oauth-id", os.Getenv("OAUTH_ID"), "ID token used to authenticate with Facebook OAuth")
		oauthSecret       = flag.String("oauth-secret", os.Getenv("OAUTH_SECRET"), "Secret token used to authenticate with Facebook OAuth")
		port              = flag.Int("port", 8080, "the port where the REST API listens for connections")
	)
	flag.Parse()

	ctx := context.Background()

	var logger *zap.Logger
	var err error
	if *environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic(err)
	}

	if *oauthID == "" {
		logger.Fatal("missing oauth-id")
	}
	if *oauthSecret == "" {
		logger.Fatal("missing oauth-secret")
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		logger.Fatal("open postgres failed", zap.Error(err))
	}
	db.SetMaxOpenConns(5)

	eventStore := &pg.EventStore{DB: db}
	if err = eventStore.Init(ctx); err != nil {
		logger.Fatal("init event store failed", zap.Error(err))
	}

	userStore := &pg.UserStore{DB: db}
	if err = userStore.Init(ctx); err != nil {
		logger.Fatal("init user store failed", zap.Error(err))
	}

	destStore := &pg.DestStore{DB: db}
	if err = destStore.Init(ctx); err != nil {
		logger.Fatal("init dest store failed", zap.Error(err))
	}

	oauthConf := &oauth2.Config{
		ClientID:     *oauthID,
		ClientSecret: *oauthSecret,
		Endpoint:     oauthFB.Endpoint,
	}
	fbClientFactory := func(oauthToken string) service.FacebookClient {
		http := oauthConf.Client(ctx, &oauth2.Token{AccessToken: oauthToken})
		return &facebook.Client{HTTP: http}
	}

	firebaseApp, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: *firebaseProjectID,
	})
	if err != nil {
		logger.Fatal("init firebase failed", zap.Error(err))
	}
	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		logger.Fatal("init firebase failed", zap.Error(err))
	}
	jwtProvider := &auth.FirebaseProvider{
		AuthClient: authClient,
		AdminUIDs:  strings.Split(*adminUIDs, ","),
	}

	service := &service.Service{
		DestStore:  destStore,
		EventStore: eventStore,
		UserStore:  userStore,

		FacebookClient: fbClientFactory,

		Auth: jwtProvider,
	}

	var handler http.Handler
	handler = rest.New(service)
	handler = log.WrapHandler(handler, logger)
	handler = handlers.CORS(
		handlers.AllowedHeaders([]string{"Authorization"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "OPTIONS", "HEAD"}),
		handlers.AllowedOrigins(strings.Split(*corsOrigins, ",")),
	)(handler)
	http.Handle("/", handler)

	http.Handle("/metrics", prom.Handler())

	addr := fmt.Sprint(":", *port)
	logger.Info("listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatal("http server failed", zap.Error(err))
	}
}
