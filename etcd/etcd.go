package etcd

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/logger"

	jsoniter "github.com/json-iterator/go"
	"github.com/patrickmn/go-cache"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceDiscover interface {
	Get(serviceName string) (ServiceMapStruct, error)
	Stop()
	discoverAllServices(serviceConfig config.Client)
}

type (
	ServiceMapStruct struct {
		ServiceUrlSlice []ServiceUrlStruct
	}
	ServiceUrlStruct struct {
		Url    string
		Weight int
	}
	LocalCache struct {
		stop       chan struct{}
		localCache *cache.Cache
	}
)

var etcdHandler *clientv3.Client

var (
	ServiceNotFoundErr = errors.New("service not found")
	etcdInitError      = errors.New("etcd initialization failure")
)

func NewEtcd(serviceConfig config.Client) ServiceDiscover {
	var err error
	stopChan := make(chan struct{}, 1)
	etcdConfig := serviceConfig.Etcd
	localCache := cache.New(time.Duration(etcdConfig.LocalCacheDefaultExpiration)*time.Second, time.Duration(etcdConfig.LocalCacheCleanUpTime)*time.Second)
	localCacheStruct := &LocalCache{stop: stopChan, localCache: localCache}
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
	localCacheStruct.discoverAllServices(serviceConfig)
	go localCacheStruct.watch(serviceConfig.ReverseHost, time.Duration(etcdConfig.LocalCacheDefaultExpiration)*time.Second)
	return localCacheStruct
}

func (etcdLocalCache *LocalCache) Stop() {
	close(etcdLocalCache.stop)
	etcdLocalCache.localCache.Flush()
}

func (etcdLocalCache *LocalCache) Get(serviceName string) (ServiceMapStruct, error) {

	if hostObj, ok := etcdLocalCache.localCache.Get(serviceName); ok {
		return hostObj.(ServiceMapStruct), nil
	}
	return ServiceMapStruct{}, ServiceNotFoundErr
}

func (etcdLocalCache *LocalCache) discoverAllServices(serviceConfig config.Client) {
	var wg sync.WaitGroup
	for _, service := range serviceConfig.ReverseHost {
		wg.Add(1)
		go func(serviceName string) {
			defer wg.Done()
			etcdLocalCache.discoverService(service.ServiceName, time.Duration(serviceConfig.Etcd.LocalCacheDefaultExpiration)*time.Second)
		}(service.ServiceName)
	}
	wg.Wait()
}

func (etcdLocalCache *LocalCache) discoverService(serviceName string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	res, err := etcdHandler.Get(ctx, serviceName)
	if err != nil {
		logger.Runtime.Error("discover service err:" + err.Error())
		return
	}
	serviceUrlSlice := make([]ServiceUrlStruct, 0)
	for _, val := range res.Kvs {
		_ = jsoniter.Unmarshal(val.Value, &serviceUrlSlice)
	}
	serviceMapStruct := ServiceMapStruct{
		ServiceUrlSlice: serviceUrlSlice,
	}
	etcdLocalCache.localCache.Set(serviceName, serviceMapStruct, timeout)
	cancel()
}

//??????????????????
func (etcdLocalCache *LocalCache) watch(reverseHost []config.ReverseHost, timeout time.Duration) {
	for _, host := range reverseHost {
		go func() {
			watchChan := etcdHandler.Watch(context.TODO(), host.ServiceName)
			for {
				select {
				case watchRes := <-watchChan:
					etcdEventHandle(etcdLocalCache.localCache, watchRes.Events, timeout)
				case <-etcdLocalCache.stop:
					break
				}
			}
		}()
	}
}

func etcdEventHandle(cache *cache.Cache, events []*clientv3.Event, timeout time.Duration) {
	for _, ev := range events {
		if ev.Type == mvccpb.PUT {
			serviceUrlSlice := make([]ServiceUrlStruct, 0)
			_ = jsoniter.Unmarshal(ev.Kv.Value, &serviceUrlSlice)
			serviceMapStruct := ServiceMapStruct{
				ServiceUrlSlice: serviceUrlSlice,
			}
			cache.Set(string(ev.Kv.Key), serviceMapStruct, timeout)
		} else {
			cache.Delete(string(ev.Kv.Key))
		}
	}
}
