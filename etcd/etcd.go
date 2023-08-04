package etcd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
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
	Exit()
	discoverAllServices(serviceConfig config.Client)
}

type (
	ServiceMapStruct struct {
		ServiceUrlSlice []config.ServiceUrlStruct
	}
	LocalCache struct {
		stop          chan struct{}
		closeComplete chan struct{}
		localCache    *cache.Cache
	}
)

var (
	etcdHandler              *clientv3.Client
	localCacheExpirationTime time.Duration
)

var (
	ServiceNotFoundErr = errors.New("service not found")
	etcdInitError      = errors.New("etcd initialization failure")
)

func NewEtcd(serviceConfig config.Client) ServiceDiscover {
	var err error
	etcdConfig := serviceConfig.Etcd
	localCacheExpirationTime = time.Duration(etcdConfig.LocalCacheDefaultExpiration) * time.Second
	localCache := cache.New(localCacheExpirationTime, time.Duration(etcdConfig.LocalCacheCleanUpTime)*time.Second)
	localCacheStruct := &LocalCache{stop: make(chan struct{}, 1), localCache: localCache, closeComplete: make(chan struct{}, 1)}
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
	go localCacheStruct.watch(serviceConfig.ReverseHost, localCacheExpirationTime)
	runtime.SetFinalizer(localCacheStruct, (*LocalCache).Exit)
	return localCacheStruct
}

func (etcdLocalCache *LocalCache) Get(serviceName string) (ServiceMapStruct, error) {
	if hostObj, ok := etcdLocalCache.localCache.Get(serviceName); ok {
		return hostObj.(ServiceMapStruct), nil
	}
	//缓存不存在则更新缓存
	etcdLocalCache.discoverService(serviceName, localCacheExpirationTime)
	if hostObj, ok := etcdLocalCache.localCache.Get(serviceName); ok {
		return hostObj.(ServiceMapStruct), nil
	}
	return ServiceMapStruct{}, ServiceNotFoundErr
}

func (etcdLocalCache *LocalCache) Delete(serviceName string) {
	etcdLocalCache.removeService(serviceName)
	etcdLocalCache.localCache.Delete(serviceName)
}

func (etcdLocalCache *LocalCache) Exit() {
	close(etcdLocalCache.stop)
	closeTimer := time.NewTimer(10 * time.Second)
	select {
	case <-etcdLocalCache.closeComplete:
	case <-closeTimer.C:
		logger.Runtime.Error("etcd watcher stop timeout!")
	}
	etcdLocalCache.localCache.Flush()
	etcdHandler.Close()
	fmt.Println("etcd stop!")
}

func (etcdLocalCache *LocalCache) discoverAllServices(serviceConfig config.Client) {
	var wg sync.WaitGroup
	for _, service := range serviceConfig.ReverseHost {
		wg.Add(1)
		service := service
		go func(serviceName string) {
			defer wg.Done()
			etcdLocalCache.discoverService(service.ServiceName, time.Duration(serviceConfig.Etcd.LocalCacheDefaultExpiration)*time.Second)
		}(service.ServiceName)
	}
	wg.Wait()
}

func (etcdLocalCache *LocalCache) discoverService(serviceName string, timeout time.Duration) {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	res, err := etcdHandler.Get(ctx, serviceName)
	if err != nil {
		logger.Runtime.Error("discover service err:" + err.Error())
		return
	}
	serviceUrlSlice := make([]config.ServiceUrlStruct, 0)
	for _, val := range res.Kvs {
		_ = jsoniter.Unmarshal(val.Value, &serviceUrlSlice)
	}
	if len(serviceUrlSlice) > 0 {
		serviceMapStruct := ServiceMapStruct{
			ServiceUrlSlice: serviceUrlSlice,
		}
		etcdLocalCache.localCache.Set(serviceName, serviceMapStruct, timeout)
	}
}

func (etcdLocalCache *LocalCache) removeService(serviceName string) {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	_, err := etcdHandler.Delete(ctx, serviceName)
	if err != nil {
		logger.Runtime.Error("delete service err:" + err.Error())
	}
}

// 监听服务变化
func (etcdLocalCache *LocalCache) watch(reverseHost []config.ReverseHost, timeout time.Duration) {
	var wg sync.WaitGroup
	for _, host := range reverseHost {
		host := host
		wg.Add(1)
		go func() {
			defer wg.Done()
			watchChan := etcdHandler.Watch(context.TODO(), host.ServiceName)
		LOOP:
			for {
				select {
				case watchRes := <-watchChan:
					etcdEventHandle(etcdLocalCache.localCache, watchRes.Events, timeout)
				case <-etcdLocalCache.stop:
					break LOOP
				}
			}
		}()
	}
	wg.Wait()
	etcdLocalCache.closeComplete <- struct{}{}
}

func etcdEventHandle(cache *cache.Cache, events []*clientv3.Event, timeout time.Duration) {
	for _, ev := range events {
		if ev.Type == mvccpb.PUT {
			serviceUrlSlice := make([]config.ServiceUrlStruct, 0)
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
