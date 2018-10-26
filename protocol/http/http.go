package http

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func Get(url string, parms string, Header map[string]string) (string, error) {
	if parms != "" {
		parms = "?" + parms
	}
	req, err := http.NewRequest("GET", url+parms, nil)
	if err != nil {
		return "", err
	}
	if Header != nil {
		for Key := range Header {
			req.Header.Add(Key, Header[Key])
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func Post(url string, parms string, Header map[string]string) (string, error) {

	req, err := http.NewRequest("POST", url, strings.NewReader(parms))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if Header != nil {
		for Key := range Header {
			if req.Header.Get(Key) == "" {
				req.Header.Add(Key, Header[Key])
			} else {
				req.Header.Set(Key, Header[Key])
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetClientIP(r *http.Request) string {
	var list []string
	if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
		list = strings.Split(ips, ",")
	}
	if list != nil && len(list) > 0 && list[0] != "" {
		rip, _, err := net.SplitHostPort(list[0])
		if err != nil {
			rip = list[0]
		}
		return rip
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr

}

type RespInfo struct {
	Result  bool
	Message string
	Code    string
}
