package main

import (
	"flag"

	"time"

	gfy "stillgrove.com/gofeedyourself/pkg/feedservice"
	config "stillgrove.com/gofeedyourself/pkg/feedservice/config"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"

	log "github.com/sirupsen/logrus"
)

const (
	ModeDefault = "dev"
	ModeUsage   = "permitted options: all (products and images), and ftp-only"
	HostUsage   = "override host from config, e.g. to localhost:8080 for development"
)

var (
	modeFlag string
	// HostFlag allows to ovveride the domain of the WooCommerce Database to be updated
	HostFlag string
	// BuildTime will be populated by the linker to tell builds appart after they were shipped
	BuildTime string
)

func init() {
	flag.StringVar(&modeFlag, "mode", ModeDefault, ModeUsage)
	flag.StringVar(&HostFlag, "host", "", HostUsage)
}

func main() {
	var (
		err error
		cfg *config.File
	)

	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.WithFields(
		log.Fields{
			"Image Built on": BuildTime,
			"Started at":     time.Now().UTC(),
		},
	).Println("Application Started")

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"

	cfg, err = config.New(configPath)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if len(HostFlag) > 0 {
		if helpers.IsOnline(HostFlag) {
			log.WithField(HostFlag, "GET successful").Println("Custom Host flag set")
			cfg.SetHost(HostFlag)
		}
		log.WithField(HostFlag, "Couldn't GET").Println("Custom Host flag rejected")
	}

	switch mode := modeFlag; mode {
	case "ftp-only":
		p, err := gfy.New(cfg, "woocommerce", true)
		if err != nil {
			log.Fatalf("%v", err)
		}
		p.PurgeImages()
	case "all":
		p, err := gfy.New(cfg, "woocommerce", true)
		if err != nil {
			log.Fatalf("%v", err)
		}
		p.PurgeProducts()
	default:
		log.WithField("Message", ModeUsage).Fatalln("No mode specified")
	}
}
