package middleware

import (
	"net"
	"simple_proxygateway/config"
)

type limiterHandler interface {
	limiterHandleFunc(ip string) bool
}

type buildHandlerFunc func(proxyConfig config.Client) limiterHandler

type LimiterStruct struct {
	limiterSlice []limiterHandler
	proxyConfig  config.Client
}

var Limiter *LimiterStruct

func init() {
	newLimiter()
}

func newLimiter() {
	lStruct := &LimiterStruct{
		limiterSlice: make([]limiterHandler, 0),
	}
	Limiter = lStruct
}

func (Limiter *LimiterStruct) SetConfig(proxyConfig config.Client) {
	Limiter.proxyConfig = proxyConfig
	Limiter.use(buildIpTableHandler, buildRestrictorHandler)
}

func (Limiter *LimiterStruct) Handle(remoteAddr string) bool {
	var result bool
	ip, _, _ := net.SplitHostPort(remoteAddr)
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	for _, handler := range Limiter.limiterSlice {
		if result = handler.limiterHandleFunc(ip); !result {
			return false
		}
	}
	return true
}

func (Limiter *LimiterStruct) use(handles ...buildHandlerFunc) {
	for _, handle := range handles {
		Limiter.limiterSlice = append(Limiter.limiterSlice, handle(Limiter.proxyConfig))
	}
}
