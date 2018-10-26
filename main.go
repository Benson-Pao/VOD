package main

import (
	"log"

	"github.com/Benson-Pao/VOD/config"
	"github.com/Benson-Pao/VOD/web/hls"
	"github.com/Benson-Pao/VOD/web/status"
)

var (
	version = "master"
)

var configInfo *config.ConfigInfo

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("vod panic: ", r)
		}
	}()

	config, err := config.GetConfig()
	if err == nil {
		configInfo = config
		hls, err := hls.NewServer(configInfo)
		if err != nil {
			log.Println(err)
			return
		}

		status.NewServer(hls)

		<-hls.Quit

	}
}
