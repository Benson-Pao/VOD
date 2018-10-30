package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Benson-Pao/VOD/core"
)

type LogInfo struct {
	Path     string
	IsEnable bool
}

func (c *ConfigInfo) SetLog(key string, Message string) error {
	now := time.Now()
	today := fmt.Sprintf("%2d-%2d-%2d", now.Year(), now.Month(), now.Day())
	filepath := c.Local.Log.Path + "/" + key + today + ".log"
	datetime := fmt.Sprintf("%2d-%2d-%2d %2d:%2d:%2d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	c.mux.Lock()
	if _, err := os.Stat(c.Local.Log.Path); os.IsNotExist(err) {
		err := os.MkdirAll(c.Local.Log.Path, os.ModePerm)
		if err != nil {
			log.Printf("mkdir failed![%v]\n", err)
			c.mux.Unlock()
			return err
		}
		if _, err := os.Stat(filepath); os.IsNotExist(err) {

		}
	}
	c.mux.Unlock()

	_, err := core.WriteLine(filepath, "["+datetime+"]:"+Message)
	return err
}
