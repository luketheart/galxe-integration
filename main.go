package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/artela-network/galxe-integration/api"
	"github.com/artela-network/galxe-integration/config"
	"github.com/artela-network/galxe-integration/fetcher"
	"github.com/artela-network/galxe-integration/logging"
	_ "github.com/artela-network/galxe-integration/logging"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug mode")
	serviceConf := flag.String("config", "./config.json", "monitor config json file path")
	flag.Parse()

	if !*debug {
		logging.SetupLogRotate()
	} else {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode enabled")
	}

	log.Info("starting the service...")

	ctx, cancel := context.WithCancel(context.Background())

	conf := loadConfig(*serviceConf)

	chainFetcher, err := fetcher.NewFetcher(ctx, conf.Fetcher)
	if err != nil {
		log.Fatalf("failed to create fetcher: %v", err)
	}
	chainFetcher.Start()

	apiServer := api.NewServer(ctx, conf.APIServer)
	apiServer.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGINT)

	<-c

	apiServer.Stop()

	cancel()

	log.Info("service exited")
}

func loadConfig(configFile string) *config.Config {
	// load config file
	if strings.HasPrefix(configFile, ".") {
		cwd, _ := os.Getwd()
		configFile = path.Join(cwd, configFile)
	}

	log.Infof("loading config file: %s ...", configFile)
	f, err := os.Open(configFile)
	if err != nil {
		log.Panicf("failed to load config: %v", err)
	}
	defer f.Close()

	rawConf, err := io.ReadAll(f)
	if err != nil {
		log.Panicf("failed to read config: %v", err)
	}

	conf := &config.Config{}
	if err := json.Unmarshal(rawConf, conf); err != nil {
		log.Panicf("failed to parse config: %v", err)
	}

	return conf
}
