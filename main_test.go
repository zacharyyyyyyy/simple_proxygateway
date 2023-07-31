package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/etcd"
	"simple_proxygateway/transmit"

	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	clientv3 "go.etcd.io/etcd/client/v3"
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
			t.Fatal(err)
		}
		urlSilce := []etcd.ServiceUrlStruct{
			{
				Url:    "127.0.0.1:80",
				Weight: 0,
			},
			{
				Url:    "127.0.0.4:80",
				Weight: 1,
			},
		}
		jsonStr, _ := jsoniter.Marshal(urlSilce)
		_, err = etcdHandler.Put(context.TODO(), proxyConfig.ReverseHost[0].ServiceName, string(jsonStr), clientv3.WithLease(respL.ID))
		if err != nil {
			t.Fatal(err)
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
			t.Fatal(err)
		}
		urlSilce := []etcd.ServiceUrlStruct{
			{
				Url:    "127.0.0.1:80",
				Weight: 0,
			},
			{
				Url:    "127.0.0.5:80",
				Weight: 1,
			},
		}
		jsonStr, _ := jsoniter.Marshal(urlSilce)
		_, err = etcdHandler.Put(context.TODO(), proxyConfig.ReverseHost[0].ServiceName, string(jsonStr), clientv3.WithLease(respL.ID))
		if err != nil {
			t.Fatal(err)
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

func TestTransmit(t *testing.T) {
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	proxy := transmit.NewProxyHandler(ServiceDiscover, proxyConfig.LoadBalanceMode, *proxyConfig)
	initRouter(proxy)
	testsStruct := []struct {
		testName string
		method   string
		target   string
		code     int
	}{
		{
			testName: "transmit test",
			method:   "POST",
			target:   "/te/st?key=1&key1=2",
			code:     404,
		},
	}
	for _, testsRow := range testsStruct {
		Convey(testsRow.testName, t, func() {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(testsRow.method, testsRow.target, strings.NewReader(""))
			req.Header.Set("Content-type", "application/x-www-form-urlencoded")
			http.DefaultServeMux.ServeHTTP(w, req)
			result := make(map[string]interface{}, 0)
			err := jsoniter.Unmarshal(w.Body.Bytes(), &result)
			if err != nil {
				t.Fatal(err)
			}
			So(w.Code, ShouldEqual, testsRow.code)

		})
	}
}
