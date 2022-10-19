package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"
	authserver "github.com/SButnyakov/music-backend-authorization/internal/app/authserver"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "configs/authserver.toml", "path to config file")
}

func main() {
	flag.Parse()

	config := authserver.NewConfig()
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatal(err)
	}

	if err := authserver.Start(config); err != nil {
		log.Fatal(err)
	}
}
