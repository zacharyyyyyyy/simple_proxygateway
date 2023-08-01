package middleware

import (
	"simple_proxygateway/config"
)

type ipTable struct {
	blacklist map[string]struct{}
}

var ipTableHandler ipTable

func buildIpTableHandler(proxyConfig config.Client) limiterHandler {
	ipTableHandler = ipTable{
		blacklist: make(map[string]struct{}, 0),
	}
	for _, ip := range proxyConfig.IpTable {
		ipTableHandler.blacklist[ip] = struct{}{}
	}
	return ipTableHandler
}

func (handler ipTable) limiterHandleFunc(ip string) bool {
	if _, ok := handler.blacklist[ip]; ok {
		return false
	}
	return true
}
