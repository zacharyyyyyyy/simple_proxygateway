package collector

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/logger"

	"github.com/olivere/elastic/v7"
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
		fmt.Println(builderMap)
		So(len(builderMap), ShouldEqual, 1)
	})
}

func TestEs(t *testing.T) {
	Convey("new collector", t, func() {
		NewCollector(*proxyConfig)
		Convey("clear es", func() {
			esClient.DeleteIndex(proxyConfig.Collector.Es.Index).Do(context.Background())
			Convey("write data", func() {
				type w struct {
					Name string
					Age  int
				}
				b := w{
					Name: "te",
					Age:  10,
				}
				dataChan <- b
				Convey("wait for flush", func() {
					time.Sleep(7 * time.Second)
					Convey("check es data", func() {
						result, err := esClient.Search(proxyConfig.Collector.Es.Index).Do(context.Background())
						if err != nil {
							log.Fatal(err)
						}
						var data w
						fmt.Println(result.Hits.TotalHits.Value)
						fmt.Println(result.Status)
						So(result.Hits.TotalHits.Value, ShouldEqual, 1)
						for _, val := range result.Each(reflect.TypeOf(data)) {
							So(val.(w).Name, ShouldEqual, "te")
						}
						esClient.DeleteIndex(proxyConfig.Collector.Es.Index).Do(context.Background())
					})
				})
			})
		})
	})
}
