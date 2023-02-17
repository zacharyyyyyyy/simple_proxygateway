package transmit

import (
	"math/rand"
	"simple_proxygateway/config"
	"time"
)

type weightTransmit struct {
}

func init() {
	register(config.LoadBalanceModeWeight, &weightTransmit{})
}

func (weightTransmit weightTransmit) getUrlString(urlSlice []serviceUrlStruct, ip string) string {
	rand.Seed(time.Now().UnixNano())
	maxLen := 0
	for _, urlStruct := range urlSlice {
		maxLen += urlStruct.weight
	}
	randIndex := rand.Intn(maxLen)
	index := 0
	for _, urlStruct := range urlSlice {
		index += urlStruct.weight
		if index > randIndex {
			return urlStruct.url
		}

	}
	return urlSlice[0].url
}
