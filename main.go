package main

import (
	"log"
	"net/http"

	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
	"simple_proxygateway/transmit"
)

func initRouter(handler http.Handler) {
	http.Handle("/", handler)
}

func main() {
	proxyConfig := &config.Client{}
	config.LoadConf(proxyConfig, "config.yaml")
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	proxy := transmit.NewProxyHandler(ServiceDiscover, proxyConfig.LoadBalanceMode)
	initRouter(proxy)
	if err := http.ListenAndServe(proxyConfig.Port, nil); err != nil {
		log.Fatal(err)
	}
}
