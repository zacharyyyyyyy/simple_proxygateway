package transmit

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"simple_proxygateway/etcd"
	"simple_proxygateway/logger"

	jsoniter "github.com/json-iterator/go"
	"github.com/patrickmn/go-cache"
)

type transmitHandler interface {
	getUrlString(urlSlice []serviceUrlStruct, ip string) string
}

type serviceUrlStruct struct {
	url    string
	weight int
}

var (
	transmitHandlerMap          = make(map[string]transmitHandler)
	localCache                  *cache.Cache
	defaultHost                 = "127.0.0.1"
	localCacheDefaultExpiration = 3
	localCacheCleanUpTime       = 10
)

func init() {
	localCache = cache.New(time.Duration(localCacheDefaultExpiration)*time.Second, time.Duration(localCacheCleanUpTime)*time.Second)
}

func NewProxyHandler(serviceDiscover etcd.ServiceDiscover, loadBalanceMode string) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			rawUrl := getRawUrl(req.URL, req.RemoteAddr, loadBalanceMode, serviceDiscover)
			u, _ := url.Parse(rawUrl)
			req.URL = u
			req.Host = u.Host // 必须显示修改Host，否则转发可能失败
		},
		ModifyResponse: func(resp *http.Response) error {
			infoLog := fmt.Sprintf("source host:%s,path:%s,code:%d", resp.Request.URL.Host, resp.Request.URL.Path, resp.StatusCode)
			logger.Runtime.Info(infoLog)
			return nil
		},
		ErrorLog: log.New(logger.Runtime, "ReverseProxy:", log.LstdFlags|log.Lshortfile),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				logger.Runtime.Error(err.Error())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				errStruct := new(struct {
					Msg  string
					Data interface{}
					Code int
				})
				errStruct.Msg = "error!,service not found!"
				errStruct.Data = ""
				errStruct.Code = http.StatusNotFound
				errJson, _ := jsoniter.Marshal(errStruct)
				w.Write(errJson)
			}
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second, //连接超时
				KeepAlive: 60 * time.Second, //长连接超时时间
			}).DialContext,
			MaxIdleConns:          100,              //最大空闲连接
			IdleConnTimeout:       90 * time.Second, //空闲超时时间
			TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
			ExpectContinueTimeout: 1 * time.Second,  //100-continue 超时时间
		},
	}
	return proxy
}

func register(modeName string, transmitHandler transmitHandler) {
	transmitHandlerMap[modeName] = transmitHandler
}

func getRawUrl(reqUrl *url.URL, originRemoteAddr string, loadBalanceMode string, serviceDiscover etcd.ServiceDiscover) string {
	reg := regexp.MustCompile(`\/`)
	pathPieceSlice := reg.Split(reqUrl.Path, -1)
	serviceName := pathPieceSlice[1]
	ip, _, _ := net.SplitHostPort(originRemoteAddr)
	transmitHost := getTransmitHostByCache(ip, serviceName)
	if transmitHost == "" {
		transmitHost = getTransmitHost(ip, serviceName, loadBalanceMode, serviceDiscover)
	}
	rawUrl := combineUrl(reqUrl.Scheme, transmitHost, pathPieceSlice[2:], reqUrl.RawQuery)
	return rawUrl
}

func combineUrl(scheme string, transmitHost string, originPathSlice []string, rawQuery string) string {
	if scheme != "" {
		scheme = scheme + "://"
	}
	pathMix := strings.Join(originPathSlice, "/")
	if len(pathMix) > 0 {
		pathMix = "/" + pathMix
	}
	if rawQuery != "" {
		rawQuery = "?" + rawQuery
	}
	return scheme + transmitHost + pathMix + rawQuery
}

func getTransmitHostByCache(ip string, serviceName string) string {
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	if transmitHost, ok := localCache.Get(ip + "_" + serviceName); ok {
		return transmitHost.(string)
	}
	return ""
}

func getTransmitHost(ip string, serviceName string, loadBalanceMode string, serviceDiscover etcd.ServiceDiscover) string {
	var err error
	if transmitHandler, ok := transmitHandlerMap[loadBalanceMode]; ok {
		serviceSlice, err := serviceDiscover.Get(serviceName)
		if err == nil {
			urlSlice := make([]serviceUrlStruct, 0)
			for _, urlStruct := range serviceSlice.ServiceUrlSlice {
				urlSlice = append(urlSlice, serviceUrlStruct{url: urlStruct.Url, weight: urlStruct.Weight})
			}
			hostResult := transmitHandler.getUrlString(urlSlice, ip)
			localCache.Set(ip+"_"+serviceName, hostResult, time.Duration(localCacheDefaultExpiration)*time.Second)
			return hostResult
		}
	}
	logger.Runtime.Error(fmt.Errorf("transmit error : %w", err).Error())
	return defaultHost
}
