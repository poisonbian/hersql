package main

import (
	"flag"
	"fmt"
	"os"

	"hersql/config"
	"hersql/server"
)

var confFilename string

func init() {
	flag.StringVar(&confFilename, "conf", "conf.yml", "Please enter a configuration file name")
}

func main() {
	flag.Parse()
	conf, err := config.Parse(confFilename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server, err := server.NewServer(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server.Start()
}
