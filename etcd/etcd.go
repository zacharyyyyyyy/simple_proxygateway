package etcd

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/logger"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceDiscover interface {
	DiscoverService(serviceConfig config.Client)
	Stop()
	Get(serviceName string) (string, error)
}

type LocalCache struct {
	serviceMap sync.Map
	lock       sync.RWMutex
	stop       chan struct{}
}

var etcdHandler *clientv3.Client

var (
	ServiceNotFoundErr = errors.New("service not found")
	etcdInitError      = errors.New("etcd initialization failure")
)

func NewEtcd(serviceConfig config.Client) ServiceDiscover {
	var err error
	stopChan := make(chan struct{}, 1)
	localCache := &LocalCache{stop: stopChan}
	etcdConfig := serviceConfig.Etcd
	etcdHandler, err = clientv3.New(clientv3.Config{
		Username:             etcdConfig.UserName,
		Password:             etcdConfig.Password,
		Endpoints:            etcdConfig.Endpoints,
		DialTimeout:          time.Duration(etcdConfig.DialTimeout) * time.Second,
		DialKeepAliveTimeout: time.Duration(etcdConfig.DialKeepAliveTime) * time.Second,
	})
	if err != nil {
		logger.Runtime.Error(err.Error())
		log.Fatal(etcdInitError)
	}
	localCache.DiscoverService(serviceConfig)
	go localCache.watch()
	return localCache
}

func (etcdLocalCache *LocalCache) DiscoverService(serviceConfig config.Client) {
	var wg sync.WaitGroup
	for _, service := range serviceConfig.ReverseHost {
		wg.Add(1)
		go func(serviceName string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			res, err := etcdHandler.Get(ctx, serviceName)
			if err != nil {
				logger.Runtime.Error("discover service err:" + err.Error())
			} else {
				etcdLocalCache.lock.RLock()
				defer etcdLocalCache.lock.RUnlock()
				etcdLocalCache.serviceMap.Store(service.ServiceName, string(res.Kvs[0].Value))
			}
			cancel()
		}(service.ServiceName)
	}
	wg.Wait()
}

func (etcdLocalCache *LocalCache) Stop() {
	close(etcdLocalCache.stop)
}

func (etcdLocalCache *LocalCache) Get(serviceName string) (string, error) {
	etcdLocalCache.lock.RLock()
	defer etcdLocalCache.lock.RUnlock()
	if host, ok := etcdLocalCache.serviceMap.Load(serviceName); ok {
		return host.(string), nil
	}
	return "", ServiceNotFoundErr
}

//监听服务变化
func (etcdLocalCache *LocalCache) watch() {
	etcdLocalCache.serviceMap.Range(func(key, value interface{}) bool {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			watchChan := etcdHandler.Watch(ctx, key.(string))
			cancel()
			for {
				select {
				case watchRes := <-watchChan:
					for _, ev := range watchRes.Events {
						etcdLocalCache.lock.Lock()
						if ev.Type == mvccpb.PUT {
							etcdLocalCache.serviceMap.Store(ev.Kv.Key, ev.Kv.Value)
						} else {
							etcdLocalCache.serviceMap.Delete(ev.Kv.Key)
						}
						etcdLocalCache.lock.Unlock()
					}

				case <-etcdLocalCache.stop:
					break
				}
			}
		}()
		return true
	})

}
