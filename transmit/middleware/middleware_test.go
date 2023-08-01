package middleware

import (
	"sync"
	"sync/atomic"
	"testing"

	"simple_proxygateway/config"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	proxyConfig *config.Client
)

func TestMain(m *testing.M) {
	proxyConfig = &config.Client{}
	config.LoadConf(proxyConfig, "../../config.yaml")
	m.Run()
}

func TestSetConfig(t *testing.T) {
	Convey("set config", t, func() {
		Limiter.SetConfig(*proxyConfig)
		So(len(Limiter.limiterSlice), ShouldEqual, 2)
	})
}

func TestIpTable(t *testing.T) {
	Convey("set config", t, func() {
		proxyConfig.IpTable = append(proxyConfig.IpTable, "127.0.0.1")
		Limiter.SetConfig(*proxyConfig)
		Convey("handle remoteAddr", func() {
			result := Limiter.Handle("127.0.0.1:80")
			So(result, ShouldBeFalse)
		})
	})
}

func TestRestrictor(t *testing.T) {
	Convey("set config", t, func() {
		proxyConfig.Restrictor.Rate = 1
		proxyConfig.Restrictor.MaxToken = 10
		Limiter.SetConfig(*proxyConfig)
		Convey("handle remoteAddr", func() {
			var successCount int64
			var wg sync.WaitGroup
			successCount, times := 0, 20
			for i := 0; i < times; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					result := Limiter.Handle("127.0.0.1:80")
					if result {
						atomic.AddInt64(&successCount, 1)
					}
				}()
			}
			wg.Wait()
			So(successCount, ShouldBeLessThan, times)
		})
	})
}
