package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rusart/muxprom"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/version"
	"github.com/threefoldtech/zos/tools/explorer/config"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	"github.com/threefoldtech/zos/tools/explorer/pkg/directory"
	"github.com/threefoldtech/zos/tools/explorer/pkg/escrow"
	escrowdb "github.com/threefoldtech/zos/tools/explorer/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/explorer/pkg/phonebook"
	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
	"github.com/threefoldtech/zos/tools/explorer/pkg/workloads"
)

// Pkg is a shorthand type for func
type Pkg func(*mux.Router, *mongo.Database) error

func main() {
	app.Initialize()

	var (
		listen       string
		dbConf       string
		dbName       string
		seed         string
		network      string
		asset        string
		ver          bool
		flushEscrows bool
	)

	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")
	flag.StringVar(&dbConf, "mongo", "mongodb://localhost:27017", "connection string to mongo database")
	flag.StringVar(&dbName, "name", "explorer", "database name")
	flag.StringVar(&config.Config.Seed, "seed", "", "wallet seed")
	flag.StringVar(&config.Config.Network, "network", "", "tfchain network")
	flag.StringVar(&config.Config.Asset, "asset", "", "which asset to use")
	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.BoolVar(&flushEscrows, "flush-escrows", false, "flush all escrows in the database, including currently active ones, and their associated addressses")

	flag.Parse()

	if ver {
		version.ShowAndExit(false)
	}

	if err := config.Valid(); err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	dropEscrow := false

	if flushEscrows {
		dropEscrow = userInputYesNo("Are you sure you want to drop all escrows and related addresses?") &&
			userInputYesNo("Are you REALLY sure you want to drop all escrows and related addresses?")
	}

	if flushEscrows && !dropEscrow {
		log.Fatal().Msg("user indicated he does not want to remove existing escrow information - please restart the explorer without the \"flush-escrows\" flag")
	}

	ctx := context.Background()
	client, err := connectDB(ctx, dbConf)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to database")
	}

	s, err := createServer(listen, dbName, client, network, seed, asset, dropEscrow)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to create HTTP server")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go s.ListenAndServe()

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v\n", err)
	}
}

func connectDB(ctx context.Context, connectionURI string) (*mongo.Client, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		return nil, err
	}

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

func createServer(listen, dbName string, client *mongo.Client, network, seed string, asset string, dropEscrowData bool) (*http.Server, error) {
	db, err := mw.NewDatabaseMiddleware(dbName, client)
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	prom := muxprom.New(
		muxprom.Router(router),
		muxprom.Namespace("explorer"),
	)
	prom.Instrument()

	router.Use(db.Middleware)
	router.Path("/metrics").Handler(promhttp.Handler()).Name("metrics")

	if dropEscrowData {
		log.Warn().Msg("dropping escrow and address collection")
		if err := db.Database().Collection(escrowdb.AddressCollection).Drop(context.Background()); err != nil {
			log.Fatal().Err(err).Msg("failed to drop address collection")
		}
		if err := db.Database().Collection(escrowdb.EscrowCollection).Drop(context.Background()); err != nil {
			log.Fatal().Err(err).Msg("failed to drop escrow collection")
		}
		log.Info().Msg("escrow and address collection dropped successfully. restart the explorer without \"flush-escrows\" flag")
		os.Exit(0)
	}

	var e escrow.Escrow
	if seed != "" && asset != "" {
		log.Info().Msgf("escrow disabled on %s %s", config.Config.Network, config.Config.Asset)
		if err := escrowdb.Setup(context.Background(), db.Database()); err != nil {
			log.Fatal().Err(err).Msg("failed to create escrow database indexes")
		}

		wallet, err := stellar.New(config.Config.Seed, config.Config.Network, config.Config.Asset)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create stellar wallet")
		}

		e = escrow.NewStellar(wallet, db.Database())

	} else {
		log.Info().Msg("escrow disabled")
		e = escrow.NewFree(db.Database())
	}

	go e.Run(context.Background())

	pkgs := []Pkg{
		phonebook.Setup,
		directory.Setup,
	}

	for _, pkg := range pkgs {
		if err := pkg(router, db.Database()); err != nil {
			log.Error().Err(err).Msg("failed to register package")
		}
	}

	if err = workloads.Setup(router, db.Database(), e); err != nil {
		log.Error().Err(err).Msg("failed to register package")
	}

	log.Printf("start on %s\n", listen)
	r := handlers.LoggingHandler(os.Stderr, router)
	r = handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)

	return &http.Server{
		Addr:    listen,
		Handler: r,
	}, nil
}

func userInputYesNo(question string) bool {
	var reply string
	fmt.Printf("%s (yes/no): ", question)
	_, err := fmt.Scan(&reply)
	if err != nil {
		// if we can't read from the cmd line something is really really wrong
		log.Fatal().Err(err).Msg("could not read user reply from command line")
	}

	reply = strings.TrimSpace(reply)
	reply = strings.ToLower(reply)

	return reply == "y" || reply == "yes"
}
