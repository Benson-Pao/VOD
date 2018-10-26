package vod

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/Benson-Pao/VOD/config"
)

type VodInfo struct {
	Key            string
	Mux            sync.RWMutex
	IsReady        bool
	FirstTime      time.Time
	LastTime       time.Time
	Config         *config.ConfigInfo
	VideoDirectory string
}

type VideoInfo struct {
	FileName    string
	Directory   string
	FullName    string
	Application config.ApplicationInfo
}

func NewVod(Key string, config *config.ConfigInfo, VideoDirectory string) *VodInfo {
	now := time.Now()
	ret := &VodInfo{
		Key:            Key,
		FirstTime:      now,
		LastTime:       now,
		Config:         config,
		VideoDirectory: VideoDirectory,
	}
	return ret
}

func (i *VodInfo) SetLastTime() {
	i.LastTime = time.Now()
}

func (i *VodInfo) GetDurationSeconds() float64 {
	duration := time.Now().Sub(i.LastTime)
	return duration.Seconds()
}

func (i *VodInfo) IsTimeOut() bool {
	return i.GetDurationSeconds() > i.Config.Local.Temp.TimeOutSec
}

func saveM3u8(ffmpegpath string, key string, VideoPath string, RoutingName string, app *config.ApplicationInfo, tempDirectory string, m3u8Path string) *exec.Cmd {
	//ffmpeg -i d:/vod/5077076/5077076.mp4 -c copy -hls_time 40 -hls_list_size 0 -y d:/vod/temp/output.m3u8
	//ffmpeg -i d:/vod/5078089/5078089.mp4 -c copy -hls_time 60 -hls_list_size 0 -hls_base_url /vod/5078089/ -hls_segment_filename d:/vod/temp/5078089/5078089-%3d.ts -y d:/vod/temp/5078089/5078089.m3u8
	cmd := exec.Command(ffmpegpath,
		"-i",
		VideoPath,
		"-c",
		"copy",
		"-hls_time",
		app.Fragment,
		"-hls_list_size",
		"0",
		"-hls_base_url",
		"/"+RoutingName+"/"+app.Name+"/",
		"-hls_segment_filename",
		tempDirectory+"/"+key+"_%3d.ts",
		"-y",
		m3u8Path)
	return cmd
}

func SaveM3u8(config *config.ConfigInfo, w http.ResponseWriter, key string, VideoPath string, RoutingName string, app *config.ApplicationInfo, tempDirectory string, m3u8Path string) error {
	path, err := config.GetFFmpeg()
	if err != nil {
		log.Println(err)
		return err
	}
	cmd := saveM3u8(path, key, VideoPath, RoutingName, app, tempDirectory, m3u8Path)
	err = ServeCommand(cmd, w)
	if err != nil {
		log.Println("Error serving screenshot: ", err)
		return err
	}
	return nil
}

func ServeCommand(cmd *exec.Cmd, w io.Writer) error {
	stdout, err := cmd.StdoutPipe()
	defer stdout.Close()
	if err != nil {
		log.Printf("Error opening stdout of command: %v", err)
		return err
	}

	err = cmd.Start()
	if err != nil {
		log.Printf("Error starting command: %v", err)
		return err
	}

	time.Sleep(10 * time.Millisecond)

	_, err = io.Copy(w, stdout)
	if err != nil {
		log.Printf("Error copying data to client: %v", err)
		cmd.Process.Signal(syscall.SIGKILL)
		cmd.Process.Wait()
		return err
	}
	cmd.Wait()
	return nil

}
