package transmit

import (
	"context"
	"log"
	"testing"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/etcd"

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
	config.LoadConf(proxyConfig, "../config.yaml")
	etcdHandler, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	m.Run()
}

func TestCombineUrl(t *testing.T) {
	Convey("combineUrl check", t, func() {
		testHost := "http://www.qq.com/te/st?key=1&value=2"
		combineUrlString := combineUrl("http", "www.qq.com", []string{"te", "st"}, "key=1&value=2")
		So(testHost, ShouldEqual, combineUrlString)
	})
}

func TestTransmitHost(t *testing.T) {
	ServiceDiscover := etcd.NewEtcd(*proxyConfig)
	Convey("register handler & add etcd Data", t, func() {
		register(config.LoadBalanceModeRandom, &randomTransmit{})
		register(config.LoadBalanceModeIpHash, &ipHashTransmit{})
		register(config.LoadBalanceModeRoundRobin, &roundRobinTransmit{})
		register(config.LoadBalanceModeWeight, &weightTransmit{})
		respL, err := etcdHandler.Grant(context.TODO(), 30)
		if err != nil {
			t.Fatal(err)
		}
		urlSilce := []config.ServiceUrlStruct{
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
		Convey("check random mode", func() {
			transmitUrl := getTransmitHost("127.0.0.1", proxyConfig.ReverseHost[0].ServiceName, config.LoadBalanceModeRandom, ServiceDiscover)
			So(transmitUrl, ShouldBeIn, []string{"127.0.0.1", "127.0.0.4"})
		})
		Convey("check ip hash mode", func() {
			transmitUrl := getTransmitHost("127.0.0.1", proxyConfig.ReverseHost[0].ServiceName, config.LoadBalanceModeIpHash, ServiceDiscover)
			So(transmitUrl, ShouldBeIn, []string{"127.0.0.1", "127.0.0.4"})
		})
		Convey("check weight mode", func() {
			transmitUrl := getTransmitHost("127.0.0.1", proxyConfig.ReverseHost[0].ServiceName, config.LoadBalanceModeWeight, ServiceDiscover)
			So(transmitUrl, ShouldBeIn, []string{"127.0.0.1", "127.0.0.4"})
		})
		Convey("check round robin mode", func() {
			transmitUrl := getTransmitHost("127.0.0.1", proxyConfig.ReverseHost[0].ServiceName, config.LoadBalanceModeRoundRobin, ServiceDiscover)
			So(transmitUrl, ShouldBeIn, []string{"127.0.0.1", "127.0.0.4"})
		})
	})
}
