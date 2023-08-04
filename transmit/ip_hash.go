package transmit

import (
	"hash/crc32"

	"simple_proxygateway/config"
)

type ipHashTransmit struct {
}

func init() {
	register(config.LoadBalanceModeIpHash, &ipHashTransmit{})
}

func (ipHashTransmit ipHashTransmit) getUrlString(urlSlice []config.ServiceUrlStruct, ip string) string {
	sliceLen := len(urlSlice)
	ipHash := crc32.ChecksumIEEE([]byte(ip))
	return urlSlice[int(ipHash)%sliceLen].Url
}
