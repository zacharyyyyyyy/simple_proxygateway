package main

import (
	"log"
	"net/http"
	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
	"simple_proxygateway/transmit"
)

func main() {
	proxyConfig := &config.Client{}
	config.LoadConf(proxyConfig, "config.yaml")
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	proxy := transmit.NewProxyHandler(ServiceDiscover)
	http.Handle("/", proxy)
	if err := http.ListenAndServe(proxyConfig.Port, nil); err != nil {
		log.Fatal(err)
	}
}
