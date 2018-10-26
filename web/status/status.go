package status

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Benson-Pao/VOD/config"
	"github.com/Benson-Pao/VOD/core/vod"
	"github.com/Benson-Pao/VOD/web/hls"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

type Server struct {
	vods    map[string]*vod.VodInfo
	Config  *config.ConfigInfo
	hls     *hls.Server
	HlsConn NetInfo
	CPU     int
}

type Windows struct{}

type NetInfo struct {
	ESTABLISHED int32
	TIME_WAIT   int32
	FIN_WAIT_2  int32
	Total       int32
}

var server *Server

func NewServer(hls *hls.Server) *Server {
	log.Println("Status Start Port", hls.Config.Local.Service.Status.Port)
	s := &Server{
		vods:    hls.GetVods(),
		Config:  hls.GetConfig(),
		hls:     hls,
		HlsConn: NetInfo{},
	}
	server = s
	s.Serve()
	return s
}

func (s *Server) Serve() {
	if s.Config.Local.Service.Status.IsEnable {
		http.HandleFunc("/"+s.Config.Local.Service.Status.RoutingName+"/", s.handle)
		http.ListenAndServe(s.Config.Local.Service.Status.Port, nil)
	}
}

func ConnStateListener(c net.Conn, cs http.ConnState) {
	//if server.LocalAddr == "" {
	// server.LocalAddr = c.LocalAddr().String()
	// server.RemoteAttr = c.RemoteAddr().String()
	//}
}

func (s *Server) ReadState() {

	switch strings.ToLower(s.Config.Local.OS) {
	case "windows":
		win := &Windows{}
		win.ReadState(s)
		break

	}

}

func (s *Server) ReadHlsConn() {
	if s.hls.IP.LocalAddr != "" {
		var os interface{}
		switch strings.ToLower(s.Config.Local.OS) {
		case "windows":
			os = &Windows{}
			os.(*Windows).ReadHlsConn(s)
			break
		}
	}

}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	//log.Println(r.URL)
	if s.Config.Local.HostToken != "" {
		if s.Config.Local.HostToken != r.Header.Get("HostToken") {
			http.NotFound(w, r)
			log.Println("HostToken valid Error")
			return
		}
	}
	names := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")

	if len(names) > 1 {
		switch names[1] {
		case "hls":
			s.ReadHlsConn()
			bytes, _ := json.Marshal(s.HlsConn)
			w.Write(bytes)
			break
		case "sysinfo":
			s.ReadState()
			bytes, _ := json.Marshal(s)
			w.Write(bytes)
			break
		case "video":
			if len(names) == 3 {
				data := s.hls.FindVideo(names[2])
				bytes, _ := json.Marshal(data)
				w.Write(bytes)
			} else {
				http.NotFound(w, r)
			}

			break
		case "reloadcfg":
			s.Config.ReLoad()
			bytes, _ := json.Marshal(s.Config)
			w.Write(bytes)
			break
		default:
			http.NotFound(w, r)
		}

	} else {
		http.NotFound(w, r)
	}

}

func (win *Windows) ReadState(s *Server) error {
	cpu, err := win.ReadCPU()
	if err != nil {
		log.Println(err)
		return nil
	}
	s.CPU = cpu
	win.ReadHlsConn(s)
	//win.hls = s.hlsConn
	return nil
}

func (win *Windows) ReadCPU() (int, error) {
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	reader := bufio.NewReader(stdout)
	for i := 0; i <= 1; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		if i == 1 {
			cpurate, _ := strconv.Atoi(strings.TrimSpace(line))
			return cpurate, nil
		}
	}
	cmd.Wait()
	return 0, nil
}

func (win *Windows) ReadHlsConn(s *Server) error {

	cmd := exec.Command("netstat", "-n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		log.Println(err)
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	utf8Reader := transform.NewReader(bytes.NewReader(buf.Bytes()), traditionalchinese.Big5.NewDecoder())
	reader := bufio.NewReader(utf8Reader)
	s.HlsConn.Total = 0
	s.HlsConn.ESTABLISHED = 0
	s.HlsConn.FIN_WAIT_2 = 0
	s.HlsConn.TIME_WAIT = 0

	for {
		line, readerr := reader.ReadString('\n')
		if readerr != nil || io.EOF == readerr {
			break
		}
		//log.Println(s.hls.IP.LocalAddr + " " + line)
		if strings.Contains(line, s.hls.IP.LocalAddr) {
			if strings.Contains(line, "ESTABLISHED") {
				s.HlsConn.ESTABLISHED++
			} else if strings.Contains(line, "TIME_WAIT") {
				s.HlsConn.TIME_WAIT++
			} else if strings.Contains(line, "FIN_WAIT_2") {
				s.HlsConn.FIN_WAIT_2++
			}
		}
	}
	s.HlsConn.Total = s.HlsConn.ESTABLISHED +
		s.HlsConn.TIME_WAIT +
		s.HlsConn.FIN_WAIT_2
	cmd.Wait()
	return nil

}
