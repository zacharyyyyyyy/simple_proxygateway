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

	"github.com/patrickmn/go-cache"
)

type transmitHandler interface {
	getUrlString(req *http.Request) string
}

var (
	transmitHandlerMap = make(map[string]transmitHandler)
	localCache         *cache.Cache
)

func init() {
	localCache = cache.New(3*time.Second, 10*time.Second)
}

func NewProxyHandler(serviceDiscover etcd.ServiceDiscover, loadBalanceMode string) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			reg := regexp.MustCompile(`\/`)
			pathPieceSlice := reg.Split(req.URL.Path, -1)
			serviceName := pathPieceSlice[1]
			ip, _, _ := net.SplitHostPort(req.RemoteAddr)
			transmitHost := getTransmitHostByCache(ip, serviceName, loadBalanceMode)
			rawUrl := combineUrl(req.URL.Scheme, transmitHost, req.URL.Path, req.URL.RawQuery)
			u, _ := url.Parse(rawUrl)
			req.URL = u
			req.Host = u.Host // 必须显示修改Host，否则转发可能失败
		},
		ModifyResponse: func(resp *http.Response) error {
			log.Println("resp status:", resp.Status)
			log.Println("resp headers:")
			for hk, hv := range resp.Header {
				log.Println(hk, ":", strings.Join(hv, ","))
			}
			return nil
		},
		ErrorLog: log.New(logger.Runtime, "ReverseProxy:", log.LstdFlags|log.Lshortfile),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				logger.Runtime.Error(err.Error())
				w.WriteHeader(http.StatusBadGateway)
				_, _ = fmt.Fprintf(w, err.Error())
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

func combineUrl(scheme string, transmitHost string, originPath string, rawQuery string) string {
	if scheme != "" {
		scheme = scheme + "://"
	}
	reg := regexp.MustCompile(`\/`)
	pathPieceSlice := reg.Split(originPath, -1)
	pathPieceSlice = pathPieceSlice[1:]
	pathMix := strings.Join(pathPieceSlice[2:], "/")
	if len(pathMix) > 0 {
		pathMix = "/" + pathMix
	}
	if rawQuery != "" {
		rawQuery = "?" + rawQuery
	}
	return scheme + transmitHost + pathMix + rawQuery
}

func getTransmitHostByCache(ip string, serviceName string, loadBalanceMode string) string {
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	if transmitHost, ok := localCache.Get(ip + "_" + serviceName); ok {
		return transmitHost.(string)
	}
	return getTransmitHost(ip, serviceName, loadBalanceMode)
}

func getTransmitHost(ip string, serviceName string, loadBalanceMode string) string {
	//todo
	return ""
}
