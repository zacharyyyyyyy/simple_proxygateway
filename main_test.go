package main

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"reflect"
	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
	"testing"
	"time"
)

var (
	proxyConfig *config.Client
	etcdHandler *clientv3.Client
)

func TestMain(m *testing.M) {
	var err error
	proxyConfig = &config.Client{}
	config.LoadConf(proxyConfig, "config.yaml")
	etcdHandler, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	m.Run()
}

func TestEtcd(t *testing.T) {
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	Convey("add etcd data", t, func() {
		respL, err := etcdHandler.Grant(context.TODO(), 30)
		if err != nil {
			fmt.Println(err)
			return
		}
		urlSilce := []etcd.ServiceUrlStruct{
			{
				Url:    "127.0.0.1",
				Weight: 0,
			},
			{
				Url:    "127.0.0.4",
				Weight: 1,
			},
		}
		jsonStr, _ := jsoniter.Marshal(urlSilce)
		_, err = etcdHandler.Put(context.TODO(), proxyConfig.ReverseHost[0].ServiceName, string(jsonStr), clientv3.WithLease(respL.ID))
		if err != nil {
			log.Fatal(err)
		}
		Convey("check local data", func() {
			localData, err := ServiceDiscover.Get(proxyConfig.ReverseHost[0].ServiceName)
			if err != nil {
				t.Fatal(err)
			}
			localUrlSlice := localData.ServiceUrlSlice
			shouldFunc := func(actual interface{}, expected ...interface{}) string {
				if !reflect.DeepEqual(actual, expected[0]) {
					return fmt.Sprintf("excepted:%v, got:%v", expected[0], actual)
				}
				return ""
			}
			So(urlSilce, shouldFunc, localUrlSlice)
		})
	})

	Convey("edit etcd data", t, func() {
		respL, err := etcdHandler.Grant(context.TODO(), 30)
		if err != nil {
			fmt.Println(err)
			return
		}
		urlSilce := []etcd.ServiceUrlStruct{
			{
				Url:    "127.0.0.1",
				Weight: 0,
			},
			{
				Url:    "127.0.0.5",
				Weight: 1,
			},
		}
		jsonStr, _ := jsoniter.Marshal(urlSilce)
		_, err = etcdHandler.Put(context.TODO(), proxyConfig.ReverseHost[0].ServiceName, string(jsonStr), clientv3.WithLease(respL.ID))
		if err != nil {
			log.Fatal(err)
		}
		Convey("check local data", func() {
			localData, err := ServiceDiscover.Get(proxyConfig.ReverseHost[0].ServiceName)
			if err != nil {
				t.Fatal(err)
			}
			localUrlSlice := localData.ServiceUrlSlice
			shouldFunc := func(actual interface{}, expected ...interface{}) string {
				if !reflect.DeepEqual(actual, expected[0]) {
					return fmt.Sprintf("excepted:%v, got:%v", expected[0], actual)
				}
				return ""
			}
			So(urlSilce, shouldFunc, localUrlSlice)
		})
	})

}
