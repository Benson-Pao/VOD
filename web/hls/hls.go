package hls

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	vodnet "github.com/Benson-Pao/VOD/net"

	"github.com/Benson-Pao/VOD/config"
	"github.com/Benson-Pao/VOD/core/vod"
	"github.com/Benson-Pao/VOD/db"
	vodhttp "github.com/Benson-Pao/VOD/protocol/http"
)

var crossdomainxml = []byte(`<?xml version="1.0" ?>
<cross-domain-policy>
	<allow-access-from domain="*" />
	<allow-http-request-headers-from domain="*" headers="*"/>
</cross-domain-policy>`)

type Server struct {
	vods   map[string]*vod.VodInfo
	Config *config.ConfigInfo
	Mux    sync.RWMutex
	Quit   chan bool
	sql    *db.MSSQL
	IP     *vodnet.IPInfo
}

var hls *Server

func NewServer(config *config.ConfigInfo) (*Server, error) {

	s := &Server{
		vods:   make(map[string]*vod.VodInfo),
		Config: config,
		sql:    db.NewDAL(config),
		IP:     &vodnet.IPInfo{},
	}

	hls = s

	if s.Config.Local.Temp.IsRebootAllRemove {
		s.Mux.Lock()
		if _, err := os.Stat(s.Config.Local.Temp.Path); !os.IsNotExist(err) {
			err := os.RemoveAll(s.Config.Local.Temp.Path)
			if err != nil {
				log.Println("Remove TempPath Error:", err)
				s.Mux.Unlock()
				return nil, err
			}
		}
		s.Mux.Unlock()
	}

	if s.Config.Local.Temp.IsAutoRemove {
		go func() {
			time.Sleep(6 * time.Hour)
			for {
				time.Sleep(10 * time.Second)

				if _, err := os.Stat(s.Config.Local.Temp.Path); !os.IsNotExist(err) {
					dirList, ReadDirerr := ioutil.ReadDir(s.Config.Local.Temp.Path)
					if ReadDirerr != nil {
						log.Println("read directory error")
						continue
					}

					for _, fi := range dirList {
						if fi.IsDir() {
							key := fi.Name()
							tempDirectory := s.Config.Local.Temp.Path + "/" + key
							if _, ok := s.vods[key]; ok {

								info := s.vods[key]
								info.Mux.Lock()
								if info.IsTimeOut() {
									if _, err := os.Stat(tempDirectory); !os.IsNotExist(err) {
										err := os.RemoveAll(tempDirectory)
										if err != nil {
											log.Println("Remove TempPath Error:", err)
										}
									}
									delete(s.vods, key)
								}
								info.Mux.Unlock()
							} else {
								if _, err := os.Stat(tempDirectory); !os.IsNotExist(err) {
									err := os.RemoveAll(tempDirectory)
									if err != nil {
										log.Println("Remove TempPath Error:", err)
									}
								}
							}
						}
					}
				}

			}
		}()
	}

	go s.Serve()
	log.Println("Hls Start Port ", config.Local.Service.Hls.Port+"/"+
		config.Local.Service.Hls.RoutingName+"/"+"{Application}/{FileName} ")

	return s, nil
}

func (s *Server) GetConfig() *config.ConfigInfo {
	return s.Config
}

func (s *Server) GetVods() map[string]*vod.VodInfo {
	return s.vods
}

func (hls *Server) Serve() {
	if hls.Config.Local.Service.Hls.IsEnable {
		http.HandleFunc("/"+hls.Config.Local.Service.Hls.RoutingName+"/", hls.handle)
		s := &http.Server{
			Addr:           hls.Config.Local.Service.Hls.Port,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			ConnState:      ConnStateListener,
		}
		s.ListenAndServe()
	}
	//http.ListenAndServe(hls.Config.Local.Service.Hls.Port, nil)
}

func ConnStateListener(c net.Conn, cs http.ConnState) {
	if hls.IP.LocalAddr == "" {
		hls.IP.LocalAddr = c.LocalAddr().String()
		hls.IP.RemoteAttr = c.RemoteAddr().String()
	}
}

func (s *Server) checkApplication(ApplicationName string) (*config.ApplicationInfo, error) {
	for _, v := range s.Config.Application {
		if v.Name == ApplicationName {
			return &v, nil
		}
	}
	return nil, errors.New("ApplicationName Not Exist")
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	clientIP := vodhttp.GetClientIP(r)
	log.Println(clientIP+" =>", s.IP.LocalAddr+r.URL.String())

	if path.Base(r.URL.Path) == "crossdomain.xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(crossdomainxml)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	reqPath := strings.TrimLeft(r.URL.Path, "/")

	ps := strings.Split(reqPath, "/")

	if len(ps) != 3 {
		log.Println("checkApplication reqPath error ", reqPath)
		http.NotFound(w, r)
		return
	}

	app, err := s.checkApplication(ps[1])
	if err != nil {
		log.Println("checkApplication error :", err)
		return
	}

	routingName := ps[0]
	fileInfo := strings.Split(ps[2], ".")
	name := strings.Split(fileInfo[0], "_")
	key := name[0]
	appKey := app.Name + "_" + key
	filename := ps[2]
	tempDirectory := s.Config.Local.Temp.Path + "/" + appKey
	tempFile := tempDirectory + "/" + filename

	switch path.Ext(r.URL.Path) {
	case ".ts":
		if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
			w.Header().Set("Content-Type", "video/mp2ts")
			ts, err := os.Open(tempFile)
			defer ts.Close()
			if err != nil {
				log.Println("Open File error: ", err)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(filename + " Open error!"))
				return
			}

			bufReader := bufio.NewReader(ts)
			buf := make([]byte, 1024)

			for {
				readNum, err := bufReader.Read(buf)
				if err != nil && err != io.EOF {
					panic(err)
				}
				if 0 == readNum {
					break
				}
			}
			http.ServeContent(w, r, filename, time.Now(), ts)
		} else {
			log.Println(filename+" File Not Found: ", err)
			http.NotFound(w, r)
		}
		break
	case ".m3u8":
		if s.Config.Local.HeaderToken != "" {
			if s.Config.Local.HeaderToken != r.Header.Get("HeaderToken") {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("hls valid Error"))
				log.Println("hls valid Error")
				return
			}
		}

		if s.Config.API.IsEnable {
			msg, err := s.CheckAPI(r, app, key)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(msg))
				return
			}
		}

		m3u8Name := key + ".m3u8"
		m3u8File := tempDirectory + "/" + m3u8Name
		sourceFile := app.Path + "/" + key + "/" + key + app.Ext

		if _, err := os.Stat(m3u8File); os.IsNotExist(err) {
			if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
				log.Println("Source File Not Found :", err)
				s.Config.SetLog("Video Not Found", "Client IP->"+clientIP+",Request->"+s.IP.LocalAddr+r.URL.String()+"=>"+fmt.Sprint(err))

				_, ok := s.vods[appKey]
				if ok {
					delete(s.vods, appKey)
				}
				http.NotFound(w, r)
				return
			}
		}

		s.Mux.Lock()

		if _, err := os.Stat(tempDirectory); os.IsNotExist(err) {
			err := os.MkdirAll(tempDirectory, os.ModePerm)
			if err != nil {
				log.Printf("mkdir failed![%v]\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 - mkdir failed!"))
				s.Mux.Unlock()
				return
			}
		}

		_, ok := s.vods[appKey]
		if !ok {
			s.vods[appKey] = vod.NewVod(appKey, s.Config, tempDirectory)
		} else {
			s.vods[appKey].LastTime = time.Now()
		}

		s.Mux.Unlock()

		s.vods[appKey].Mux.Lock()
		if !s.vods[appKey].IsReady {

			vod.SaveM3u8(s.Config, w, key, sourceFile, routingName, app, tempDirectory, m3u8File)

		}
		m3u8, err := os.Open(m3u8File)
		defer m3u8.Close()
		if err != nil {
			s.vods[appKey].IsReady = false
			log.Println("Open File error: ", err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("m3u8 Not Found!"))
			s.vods[appKey].Mux.Unlock()
			return
		} else {
			s.vods[appKey].IsReady = true
		}

		s.vods[appKey].Mux.Unlock()

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Content-Disposition", "attachment; filename="+m3u8Name)

		http.ServeContent(w, r, filename, time.Now(), m3u8)
		break
	}
}

func (s *Server) CheckAPI(r *http.Request, app *config.ApplicationInfo, key string) (string, error) {
	if s.Config.API.IsEnable {
		// vars := r.URL.Query()
		// data := vars.Get(s.Config.API.ParmKey)

		headers := map[string]string{}
		if s.Config.Local.HostToken != "" {
			headers["HostToken"] = s.Config.Local.HostToken
		}
		ClientIP := vodhttp.GetClientIP(r)
		parms := r.URL.RawQuery + "&ClientIP=" + ClientIP + "&Application=" + app.Name + "&FileName=" + key + app.Ext
		r, err := vodhttp.Get(s.Config.API.URL, parms, headers)
		if err != nil {
			log.Println(err)
			return "Valid API Error", err
		}

		msg := &vodhttp.RespInfo{}

		err = json.Unmarshal([]byte(r), msg)
		if err != nil {
			log.Println(err)
			return "JSON Covert Error", err
		}
		if !msg.Result {
			err = errors.New("API Valid Error")
			log.Println(err)
			return "Valid Error", err
		}
	}
	return "", nil
}

func (s *Server) FindVideo(FileName string) []vod.VideoInfo {
	name := strings.Split(FileName, ".")

	videos := make([]vod.VideoInfo, 0)
	if len(name) == 2 {
		ext := strings.SplitN(name[1], "/", 2)
		FileName = name[0] + "." + ext[0]
		var FullName string
		for _, v := range s.Config.Application {
			FullName = v.Path + "/" + name[0] + "/" + FileName
			if _, err := os.Stat(FullName); !os.IsNotExist(err) {
				videos = append(videos, vod.VideoInfo{
					FileName:    FileName,
					Directory:   v.Path + "/" + name[0],
					FullName:    FullName,
					Application: v,
				})
			}

		}
	}
	return videos
}
