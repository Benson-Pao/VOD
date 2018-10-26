package config

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"strings"
	"sync"
)

var (
	cfg = flag.String("Config", "vod.cfg", "vod Config File")
)

type URLInfo struct {
	URL      string
	IsEnable bool
}

type PortInfo struct {
	Port        string
	IsEnable    bool
	RoutingName string
}

type ServiceInfo struct {
	Hls    PortInfo
	Status PortInfo
}

type TempInfo struct {
	Path              string
	TimeOutSec        float64
	IsRebootAllRemove bool
	IsAutoRemove      bool
}

type LocalInfo struct {
	HostID      string
	HostToken   string
	HeaderToken string
	Domain      string
	OS          string
	Service     ServiceInfo
	Temp        TempInfo
	Log         LogInfo
}

type ApplicationInfo struct {
	Name     string
	Path     string
	Fragment string
	Ext      string
}

type SQLServerInfo struct {
	IsEnable bool
	Server   string
	Port     string
	User     string
	Password string
	DataBase string
}

type ConfigInfo struct {
	Local       LocalInfo
	API         URLInfo
	Application []ApplicationInfo
	SQL         SQLServerInfo
	mux         sync.RWMutex
}

func LoadConfig(filename string) (*ConfigInfo, error) {
	var config ConfigInfo
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("ReadLoadConfigFile %s error:%v", filename, err)
		return nil, err
	}

	log.Printf("LoadConfig: \r\n%s", string(data))

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("json.Unmarshal error:%v", err)
		return nil, err
	}
	return &config, nil
}

func GetConfig() (*ConfigInfo, error) {
	log.Printf("Server Config Loading....")
	return LoadConfig(*cfg)
}

func (c *ConfigInfo) GetFFmpeg() (string, error) {

	var path string
	switch strings.ToLower(c.Local.OS) {
	case "windows":
		path = "./ffmpeg/win64/ffmpeg"
		return path, nil
	default:
		return "", errors.New("OS Unavailable")
	}
}

func (c *ConfigInfo) ReLoad() {
	info, err := LoadConfig(*cfg)
	if err == nil {
		c = info
	}
}
