package transmit

import (
	"math/rand"
	"time"

	"simple_proxygateway/config"
)

type weightTransmit struct {
}

func init() {
	register(config.LoadBalanceModeWeight, &weightTransmit{})
}

func (weightTransmit weightTransmit) getUrlString(urlSlice []config.ServiceUrlStruct, ip string) string {
	rand.Seed(time.Now().UnixNano())
	maxLen := 0
	for _, urlStruct := range urlSlice {
		maxLen += urlStruct.Weight
	}
	randIndex := rand.Intn(maxLen)
	index := 0
	for _, urlStruct := range urlSlice {
		index += urlStruct.Weight
		if index > randIndex {
			return urlStruct.Url
		}

	}
	return urlSlice[0].Url
}
