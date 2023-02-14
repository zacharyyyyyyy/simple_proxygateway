package main

import (
	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
)

func main() {
	proxyConfig := &config.Client{}
	config.LoadConf(proxyConfig, "config.yaml")
	etcd.NewEtcd(*proxyConfig)

}
