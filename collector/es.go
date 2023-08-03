package collector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"simple_proxygateway/config"
	"simple_proxygateway/logger"

	"github.com/olivere/elastic/v7"
	"golang.org/x/sync/semaphore"
)

type elasticSearch struct {
	client       *elastic.Client
	bulkRequest  *elastic.BulkService
	dataSlice    []interface{}
	bulkMaxCount int
	index        string
}

const esOutputer = "es"

var (
	elasticHandler              = &elasticSearch{}
	goroutineLimit        int64 = 100
	goroutineWeight       int64 = 1
	sema                        = semaphore.NewWeighted(goroutineLimit)
	tickerTimeout               = 5
	goroutineNotEnoughErr       = errors.New("es goroutine not enough")
)

func init() {
	register(esOutputer, elasticHandler)
}

func (es elasticSearch) new(proxyConfig config.Collector) writer {
	esConfig := proxyConfig.Es
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
	elasticHandler.client = client
	elasticHandler.dataSlice = make([]interface{}, 0, 20)
	elasticHandler.bulkRequest = client.Bulk()
	elasticHandler.bulkMaxCount = esConfig.BulkMaxCount
	elasticHandler.index = esConfig.Index
	return elasticHandler
}

func (es elasticSearch) run(ctx context.Context, dataChan <-chan interface{}) {
	defer es.client.Stop()
	ticker := time.NewTimer(time.Duration(tickerTimeout) * time.Second)
	for {
		select {
		case data := <-dataChan:
			elasticHandler.dataSlice = append(elasticHandler.dataSlice, data)
			if ctx.Err() != nil {
				es.bulkCreate(elasticHandler.dataSlice)
				logger.Runtime.Info("elasticsearch client close!")
				return
			}
			if len(elasticHandler.dataSlice) >= es.bulkMaxCount {
				if ok := sema.TryAcquire(goroutineWeight); ok != true {
					logger.Runtime.Error(goroutineNotEnoughErr.Error())
					continue
				}
				bulkData := elasticHandler.dataSlice[:es.bulkMaxCount]
				elasticHandler.dataSlice = elasticHandler.dataSlice[es.bulkMaxCount:]
				go func(bulkData []interface{}) {
					defer sema.Release(goroutineWeight)
					es.bulkCreate(bulkData)
				}(bulkData)
			}
		case <-ticker.C:
			//清空剩余data
			es.bulkCreate(elasticHandler.dataSlice)
		case <-ctx.Done():
			//清空剩余data
			es.bulkCreate(elasticHandler.dataSlice)
			logger.Runtime.Info("elasticsearch client close!")
			return
		}
	}
}

func (es elasticSearch) bulkCreate(bulkData []interface{}) {
	if len(bulkData) > 0 {
		for _, data := range bulkData {
			req := elastic.NewBulkIndexRequest().Index(es.index).Type("_doc").Doc(data)
			es.bulkRequest.Add(req)
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := es.bulkRequest.Do(ctx)
		if err != nil {
			logger.Runtime.Error(err.Error())
		}
	}
}
