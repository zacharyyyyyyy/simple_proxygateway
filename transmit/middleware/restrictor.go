package middleware

import (
	"context"
	"time"

	"simple_proxygateway/config"

	"golang.org/x/time/rate"
)

type restrictor struct {
	open        bool
	rateLimiter *rate.Limiter
	waitTime    int
}

var restrictorHandler restrictor

func buildRestrictorHandler(proxyConfig config.Client) limiterHandler {
	restrictorHandler = restrictor{
		open:        proxyConfig.Restrictor.Open,
		waitTime:    proxyConfig.Restrictor.WaitTime,
		rateLimiter: rate.NewLimiter(rate.Limit(proxyConfig.Restrictor.Rate), proxyConfig.Restrictor.MaxToken),
	}
	return restrictorHandler
}

func (restrictor restrictor) limiterHandleFunc(ip string) bool {
	if !restrictor.open {
		return true
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(restrictor.waitTime)*time.Second)
	err := restrictor.rateLimiter.Wait(ctx)
	if err != nil {
		return false
	}
	return true
}
