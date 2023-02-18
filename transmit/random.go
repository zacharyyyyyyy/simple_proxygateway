package transmit

import (
	"math/rand"
	"time"

	"simple_proxygateway/config"
)

type randomTransmit struct {
}

func init() {
	register(config.LoadBalanceModeRandom, &randomTransmit{})
}

func (randomTransmit randomTransmit) getUrlString(urlSlice []serviceUrlStruct, ip string) string {
	rand.Seed(time.Now().UnixNano())
	sliceLen := len(urlSlice)
	randIndex := rand.Intn(sliceLen)
	return urlSlice[randIndex].url
}
