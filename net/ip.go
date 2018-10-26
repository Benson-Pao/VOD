package net

import (
	"log"
	"net"
)

type IPInfo struct {
	LocalAddr  string
	RemoteAttr string
}

func GetLocalIPs() ([]string, error) {
	interfaceAddr, err := net.InterfaceAddrs()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	ips := make([]string, 0)
	for _, address := range interfaceAddr {
		ipNet, isValidIpNet := address.(*net.IPNet)
		if isValidIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	return ips, nil
}

func GetMacAddrs() ([]string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	macs := make([]string, 0)
	for _, netInterface := range netInterfaces {
		macAddr := netInterface.HardwareAddr.String()
		if len(macAddr) == 0 {
			continue
		}

		macs = append(macs, macAddr)
	}
	return macs, nil
}
