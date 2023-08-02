package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"simple_proxygateway/collector"
	"syscall"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
	"simple_proxygateway/transmit"

	"github.com/arl/statsviz"
	"log"
)

func initRouter(handler http.Handler) {
	http.Handle("/", handler)
	mux := http.DefaultServeMux
	if err := statsviz.Register(mux, statsviz.Root("/go/statsviz")); err != nil {
		log.Fatal(err)
	}
}

func main() {
	proxyConfig := &config.Client{}
	config.LoadConf(proxyConfig, "config.yaml")
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	proxy := transmit.NewProxyHandler(ServiceDiscover, proxyConfig.LoadBalanceMode, *proxyConfig)
	initRouter(proxy)
	if proxyConfig.OpenCollector {
		collector.NewCollector(*proxyConfig)
	}
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
		if proxyConfig.OpenCollector {
			collector.Stop()
		}
		_ = server.Shutdown(ctx)
	}
	fmt.Println("server stop!")
}
