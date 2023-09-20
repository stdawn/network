/*
@Time : 2021/1/2 10:58
@Author : LiuKun
@File : base
@Software: GoLand
@Description:
*/

package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type RequestMethod string

const (
	RequestMethodGet     RequestMethod = "GET"
	RequestMethodPost    RequestMethod = "POST"
	RequestMethodPut     RequestMethod = "PUT"
	RequestMethodDelete  RequestMethod = "DELETE"
	RequestMethodHead    RequestMethod = "HEAD"
	RequestMethodOptions RequestMethod = "OPTIONS"
	RequestMethodTrace   RequestMethod = "TRACE"
	RequestMethodConnect RequestMethod = "CONNECT"
)

func (r RequestMethod) String() string {
	return string(r)
}

// Request 带3次重试的网络请求
func Request(method RequestMethod, url, params string, header interface{}) (string, error) {
	response, _, err := RequestWithCookie(method, url, params, header, nil)
	return response, err
}

// RequestWithCookie 带3次重试的网络请求,添加Cookie
func RequestWithCookie(method RequestMethod, url, params string, header interface{}, cookie []*http.Cookie) (string, []*http.Cookie, error) {
	return retryRequest(3, method, url, params, header, cookie)
}

// SynRequest 同步网络请求
func SynRequest(method RequestMethod, url, params string, header interface{}, cookie []*http.Cookie) (string, []*http.Cookie, error) {
	var req *http.Request
	var err error

	switch method {
	case RequestMethodGet:
		fallthrough
	case RequestMethodHead:
		reqUrl := url
		if len(params) > 0 {
			if strings.Contains(url, "?") {
				reqUrl = reqUrl + "&" + params
			} else {
				reqUrl = reqUrl + "?" + params
			}
		}
		req, err = http.NewRequest(string(method), reqUrl, nil)
	case RequestMethodPost:
		fallthrough
	case RequestMethodPut:
		fallthrough
	case RequestMethodDelete:
		fallthrough
	case RequestMethodOptions:
		fallthrough
	case RequestMethodTrace:
		fallthrough
	case RequestMethodConnect:
		req, err = http.NewRequest(string(method), url, strings.NewReader(params))
	default:
		return "", nil, errors.New(string(method) + "请求方法无效")

	}

	if err != nil {
		return "", nil, err
	}

	//增加header选项
	err = addRequestHeader(req, header)
	if err != nil {
		return "", nil, err

	}
	//防止Response出现乱码
	req.Header.Set("Accept-Encoding", "")
	//同一个Host不重用TCP，减少当大量并发请求时的EOF出错率
	req.Close = true

	for _, c := range cookie {
		req.AddCookie(c)
	}

	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Do(req)

	if err != nil {
		return "", nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	return string(body), resp.Cookies(), nil
}

// 失败网络请求重试
func retryRequest(retryCount int, method RequestMethod, url, params string, header interface{}, cookie []*http.Cookie) (string, []*http.Cookie, error) {

	response, cs, err := SynRequest(method, url, params, header, cookie)
	if err == nil {
		return response, cs, nil
	}
	if retryCount == 0 {
		return response, cs, err
	}
	return retryRequest(retryCount-1, method, url, params, header, cookie)
}

func addRequestHeader(req *http.Request, header interface{}) error {

	var mapResult map[string]string

	switch header.(type) {
	case string:
		headerStr := header.(string)
		if len(headerStr) < 1 {
			return nil
		}
		err := json.Unmarshal([]byte(headerStr), &mapResult)
		if err != nil {
			return errors.New(fmt.Sprintf("JsonToMap err: %s", err.Error()))
		}
	case map[string]string:
		mapResult = header.(map[string]string)
	default:
		return errors.New("header is not string or map[string]string")
	}

	for k, v := range mapResult {

		if len(k) < 1 {
			continue
		}
		headerKey := strings.TrimSpace(k)
		headerValue := strings.TrimSpace(v)

		if headerKey == "Content-Length" {
			continue
		}
		req.Header.Set(headerKey, headerValue)
	}

	return nil
}
