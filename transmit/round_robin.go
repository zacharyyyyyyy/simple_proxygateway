package transmit

import (
	"simple_proxygateway/config"
	"sync/atomic"
)

type roundRobinTransmit struct {
}

func init() {
	register(config.LoadBalanceModeRoundRobin, &roundRobinTransmit{})
}

var currentIndex int32 = 0

func (roundRobinTransmit) getUrlString(urlSlice []serviceUrlStruct, ip string) string {
	sliceLen := len(urlSlice)
	if currentIndex > int32(sliceLen) {
		atomic.StoreInt32(&currentIndex, 0)
	}
	index := atomic.AddInt32(&currentIndex, 1) % int32(sliceLen)
	return urlSlice[index].url

}
