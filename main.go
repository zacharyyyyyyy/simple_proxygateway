package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	proxy := transmit.NewProxyHandler(ServiceDiscover, proxyConfig.LoadBalanceMode, *proxyConfig)
	initRouter(proxy)
	server := http.Server{Addr: proxyConfig.Port, Handler: nil}
	go func() {
		fmt.Println("server running!")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()
	signs := make(chan os.Signal, 1)
	signal.Notify(signs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-signs:
		fmt.Println("server stopping!")
		ctx, _ := context.WithTimeout(context.Background(), time.Duration(proxyConfig.TimeOut)*time.Second)
		ServiceDiscover.Exit()
		_ = server.Shutdown(ctx)
	}
	fmt.Println("server stop!")
}
