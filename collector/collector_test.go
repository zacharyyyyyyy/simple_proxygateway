package collector

import (
	"fmt"
	"github.com/olivere/elastic/v7"
	"log"
	"simple_proxygateway/config"
	"simple_proxygateway/logger"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	proxyConfig *config.Client
	esClient    *elastic.Client
)

func TestMain(m *testing.M) {
	proxyConfig = &config.Client{}
	config.LoadConf(proxyConfig, "../config.yaml")
	esConfig := proxyConfig.Collector.Es
	errorLog := log.New(logger.Runtime, "", log.LstdFlags)
	client, err := elastic.NewClient(
		elastic.SetErrorLog(errorLog),
		elastic.SetURL(fmt.Sprintf("%s:%s", esConfig.Host, esConfig.Port)),
		elastic.SetSniff(false),
		elastic.SetBasicAuth(esConfig.Username, esConfig.Password), // 账号密码
	)
	if err != nil {
		panic(err)
	}
	esClient = client
	m.Run()
}

func TestRegister(t *testing.T) {
	Convey("check builderMap", t, func() {
		fmt.Println(len(builderMap))
		So(len(builderMap), ShouldEqual, 1)
	})
}

func TestEs(t *testing.T) {
	Convey("new collector", t, func() {
		NewCollector(*proxyConfig)
		Convey("write data", func() {
			Write("test")
			Convey("wait for flush", func() {
				time.Sleep(6 * time.Second)
				Convey("check es data", func() {

					//result, err := esClient.Get().Index(proxyConfig.Collector.Es.Index).Type("_doc").Do(context.Background())

				})
			})
		})
	})
}
