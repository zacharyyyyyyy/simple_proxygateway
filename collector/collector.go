package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/logger"
)

type (
	builder interface {
		new(client config.Collector) writer
	}
	writer interface {
		run(ctx context.Context, dataChan <-chan interface{})
	}
	EsMsg struct {
		ServiceName  string
		TransmitTime int
		ResultTime   int
		Host         string
		StatusCode   int
	}
)

var (
	dataChan                          = make(chan interface{}, 200)
	builderMap                        = make(map[string]builder)
	collectorCtx, collectorCancelFunc = context.WithCancel(context.Background())
	closeChan                         = make(chan struct{}, 1)
	running                           bool
)

func NewCollector(config config.Client) {
	if config.Collector.Switch == "" {
		log.Fatal("switch do not set")
	}
	if builderHandler, ok := builderMap[config.Collector.Switch]; ok {
		handler := builderHandler.new(config.Collector)
		go func() {
			running = true
			handler.run(collectorCtx, dataChan)
			closeChan <- struct{}{}
		}()
		return
	}
	log.Fatal("collector not exists")
}

func Write(data EsMsg) {
	if running {
		dataChan <- data
	}
}

func Stop() {
	collectorCancelFunc()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		logger.Runtime.Error("stop collector timeout")
	case <-closeChan:
	}
	fmt.Println("collector stop")
}

func register(name string, builder builder) {
	if _, ok := builderMap[name]; !ok {
		builderMap[name] = builder
	}
}
