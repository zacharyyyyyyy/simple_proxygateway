package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type (
	ReverseHost struct {
		ServiceName string `yaml:"service_name"`
	}
	Etcd struct {
		Endpoints                   []string `yaml:"endpoints"`
		UserName                    string   `yaml:"username"`
		Password                    string   `yaml:"password"`
		DialTimeout                 int      `yaml:"dial_timeout"`
		DialKeepAliveTimeout        int      `yaml:"dial_keepalive_timeout"`
		DialKeepAliveTime           int      `yaml:"dial_keepalive_time"`
		LocalCacheDefaultExpiration int      `yaml:"local_cache_default_expiration"` //本地缓存默认过期时间
		LocalCacheCleanUpTime       int      `yaml:"local_cache_clean_up_time"`      //本地缓存过期清理时间
	}
	HttpTransport struct {
		DialTimeOut           int `yaml:"dial_time_out"`
		DialKeepAlive         int `yaml:"dial_keep_alive"`
		MaxIdleConns          int `yaml:"max_idle_conns"`
		MaxIdleConnsPerHost   int `yaml:"max_idle_conns_per_host"`
		MaxConnsPerHost       int `yaml:"max_conns_per_host"`
		IdleConnTimeout       int `yaml:"idle_conn_timeout"`
		TLSHandshakeTimeout   int `yaml:"tls_handshake_timeout"`
		ExpectContinueTimeout int `yaml:"expect_continue_timeout"`
	}
	Restrictor struct {
		Open     bool `yaml:"open"`
		Rate     int  `yaml:"rate"`
		MaxToken int  `yaml:"max_token"`
		WaitTime int  `yaml:"wait_time"`
	}
	ElasticSearch struct {
		Username     string `yaml:"username"`
		Password     string `yaml:"password"`
		Host         string `yaml:"host"`
		Port         string `yaml:"port"`
		Index        string `yaml:"index"`
		BulkMaxCount int    `yaml:"bulk_max_count"`
	}
	Collector struct {
		Switch string        `yaml:"switch"`
		Es     ElasticSearch `yaml:"es"`
	}
	Client struct {
		ReverseHost     []ReverseHost `yaml:"reverse_host"`
		Etcd            Etcd          `yaml:"etcd"`
		TimeOut         int           `yaml:"timeout"`
		Port            string        `yaml:"port"`
		LoadBalanceMode string        `yaml:"load_balance_mode"`
		DefaultUrl      string        `yaml:"default_url"`
		HttpTransport   HttpTransport `yaml:"http_transport"`
		IpTable         []string      `yaml:"ip_table"`
		Restrictor      Restrictor    `yaml:"restrictor"`
		OpenCollector   bool          `yaml:"open_collector"`
		Collector       Collector     `yaml:"collector"`
	}
)

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
