package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type ReverseHost struct {
	ServiceName     string `yaml:"service_name"`
	LoadBalanceMode string `yaml:"load_balance_mode"`
}

type Etcd struct {
	Endpoints                   []string `yaml:"endpoints"`
	UserName                    string   `yaml:"username"`
	Password                    string   `yaml:"password"`
	DialTimeout                 int      `yaml:"dial_timeout"`
	DialKeepAliveTimeout        int      `yaml:"dial_keepalive_timeout"`
	DialKeepAliveTime           int      `yaml:"dial_keepalive_time"`
	LocalCacheDefaultExpiration int      `yaml:"local_cache_default_expiration"` //本地缓存默认过期时间
	LocalCacheCleanUpTime       int      `yaml:"local_cache_clean_up_time"`      //本地缓存过期清理时间
}

type Client struct {
	ReverseHost []ReverseHost `yaml:"reverse_host"`
	Etcd        Etcd          `yaml:"etcd"`
	TimeOut     int           `yaml:"timeout"`
	Port        string        `yaml:"port"`
}

const (
	LoadBalanceModeRandom     = "random"
	LoadBalanceModeIpHash     = "ip_hash"
	LoadBalanceModeWeight     = "weight"
	LoadBalanceModeRoundRobin = "round_robin"
)

func LoadConf(config *Client, configFileName string) {
	var f *os.File
	f, err := os.Open(configFileName)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.NewDecoder(f).Decode(config)
	if err != nil {
		log.Fatal(err)
	}
}
