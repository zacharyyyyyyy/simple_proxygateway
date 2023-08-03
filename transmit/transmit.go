package transmit

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"simple_proxygateway/collector"
	"simple_proxygateway/config"
	"simple_proxygateway/transmit/middleware"
	"strconv"
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
	localCacheDefaultExpiration = 3
	localCacheCleanUpTime       = 10
	defaultUrl                  string
)

func init() {
	localCache = cache.New(time.Duration(localCacheDefaultExpiration)*time.Second, time.Duration(localCacheCleanUpTime)*time.Second)
}

func NewProxyHandler(serviceDiscover etcd.ServiceDiscover, loadBalanceMode string, proxyConfig config.Client) http.Handler {
	defaultUrl = proxyConfig.DefaultUrl
	middleware.Limiter.SetConfig(proxyConfig)
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			middlewareResult := middleware.Limiter.Handle(req.RemoteAddr)
			var rawUrl, serviceName string
			if middlewareResult {
				rawUrl, serviceName = getRawUrlAndServiceName(req.URL, req.RemoteAddr, loadBalanceMode, serviceDiscover)
			} else {
				rawUrl, serviceName = "", ""
			}
			u, _ := url.Parse(rawUrl)
			req.URL = u
			req.Host = u.Host // 必须显示修改Host，否则转发可能失败
			req.Header.Add("Service", serviceName)
			req.Header.Add("Transmit-Time", strconv.FormatInt(time.Now().Unix(), 10))
		},
		ModifyResponse: func(resp *http.Response) error {
			infoLog := fmt.Sprintf("source host:%s,path:%s,code:%d", resp.Request.URL.Host, resp.Request.URL.Path, resp.StatusCode)
			logger.Runtime.Info(infoLog)
			go func() {
				//转发记录采集
				transmitTime, _ := strconv.Atoi(resp.Request.Header.Get("Transmit-Time"))
				collector.Write(collector.EsMsg{
					ServiceName:  resp.Request.Header.Get("Service"),
					TransmitTime: transmitTime,
					ResultTime:   int(time.Now().Unix()),
					Host:         resp.Request.Host,
					StatusCode:   resp.StatusCode,
				})
			}()
			return nil
		},
		ErrorLog: log.New(logger.Runtime, "ReverseProxy:", log.LstdFlags|log.Lshortfile),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				logger.Runtime.Error(err.Error())
				w.Header().Set("Content-Type", "application/json")
				//Host 为空时，默认为限流或ip黑名单等限制
				errStruct := new(struct {
					Msg  string
					Data interface{}
					Code int
				})
				if r.URL.Host == "" {
					w.WriteHeader(http.StatusInternalServerError)
					errStruct.Msg = "error!temporarily unavailable for service"
					errStruct.Data = ""
					errStruct.Code = http.StatusInternalServerError
				} else {
					w.WriteHeader(http.StatusNotFound)
					errStruct.Msg = "error!service not found!"
					errStruct.Data = ""
					errStruct.Code = http.StatusNotFound
				}
				go func() {
					//转发记录采集
					transmitTime, _ := strconv.Atoi(r.Header.Get("Transmit-Time"))
					collector.Write(collector.EsMsg{
						ServiceName:  r.Header.Get("Service"),
						TransmitTime: transmitTime,
						ResultTime:   int(time.Now().Unix()),
						Host:         r.Host,
						StatusCode:   errStruct.Code,
					})
				}()
				errJson, _ := jsoniter.Marshal(errStruct)
				w.Write(errJson)
			}
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(proxyConfig.HttpTransport.DialTimeOut) * time.Second,   //连接超时
				KeepAlive: time.Duration(proxyConfig.HttpTransport.DialKeepAlive) * time.Second, //长连接超时时间
			}).DialContext,
			MaxIdleConns:          proxyConfig.HttpTransport.MaxIdleConns, //最大空闲连接
			MaxIdleConnsPerHost:   proxyConfig.HttpTransport.MaxIdleConnsPerHost,
			MaxConnsPerHost:       proxyConfig.HttpTransport.MaxConnsPerHost,                                    //每个host最大连接数
			IdleConnTimeout:       time.Duration(proxyConfig.HttpTransport.IdleConnTimeout) * time.Second,       //空闲超时时间
			TLSHandshakeTimeout:   time.Duration(proxyConfig.HttpTransport.TLSHandshakeTimeout) * time.Second,   //tls握手超时时间
			ExpectContinueTimeout: time.Duration(proxyConfig.HttpTransport.ExpectContinueTimeout) * time.Second, //100-continue 超时时间
		},
	}
	return proxy
}

func register(modeName string, transmitHandler transmitHandler) {
	transmitHandlerMap[modeName] = transmitHandler
}

func getRawUrlAndServiceName(reqUrl *url.URL, originRemoteAddr string, loadBalanceMode string, serviceDiscover etcd.ServiceDiscover) (string, string) {
	reg := regexp.MustCompile(`\/`)
	pathPieceSlice := reg.Split(reqUrl.Path, -1)
	serviceName := pathPieceSlice[1]
	ip, _, _ := net.SplitHostPort(originRemoteAddr)
	transmitHost := getTransmitHostByCache(ip, serviceName)
	if transmitHost == "" {
		transmitHost = getTransmitHost(ip, serviceName, loadBalanceMode, serviceDiscover)
	}
	rawUrl := combineUrl(reqUrl.Scheme, transmitHost, pathPieceSlice[2:], reqUrl.RawQuery)
	return rawUrl, serviceName
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
	err = errors.New("service data not exists")
	logger.Runtime.Error(fmt.Errorf("transmit error : %w", err).Error())
	return defaultUrl
}
