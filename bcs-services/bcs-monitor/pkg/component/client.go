/*
 * Tencent is pleased to support the open source community by making Blueking Container Service available.
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package component

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	resty "github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/thanos-io/thanos/pkg/store"
	"k8s.io/klog/v2"

	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/rest/tracing"
)

const (
	timeout = time.Second * 30
)

var (
	maskKeys = map[string]struct{}{
		"bk_app_secret": {},
	}
	clientOnce   sync.Once
	globalClient *resty.Client
)

// restyReqToCurl curl 格式的请求日志
func restyReqToCurl(r *resty.Request) string {
	headers := ""
	for key, values := range r.Header {
		for _, value := range values {
			headers += fmt.Sprintf(" -H %q", fmt.Sprintf("%s: %s", key, value))
		}
	}

	// 过滤掉敏感信息
	rawURL := *r.RawRequest.URL
	queryValue := rawURL.Query()
	for key := range queryValue {
		if _, ok := maskKeys[key]; ok {
			queryValue.Set(key, "<masked>")
		}
	}
	rawURL.RawQuery = queryValue.Encode()

	reqMsg := fmt.Sprintf("curl -X %s '%s'%s", r.Method, rawURL.String(), headers)
	if r.Body != nil {
		switch body := r.Body.(type) {
		case []byte:
			reqMsg += fmt.Sprintf(" -d %q", body)
		case string:
			reqMsg += fmt.Sprintf(" -d %q", body)
		case io.Reader:
			reqMsg += fmt.Sprintf(" -d %q (io.Reader)", body)
		default:
			prtBodyBytes, err := json.Marshal(body)
			if err != nil {
				reqMsg += fmt.Sprintf(" -d %q (MarshalErr %s)", body, err)
			} else {
				reqMsg += fmt.Sprintf(" -d '%s'", prtBodyBytes)
			}
		}
	}
	if r.FormData.Encode() != "" {
		encodeStr := r.FormData.Encode()
		reqMsg += fmt.Sprintf(" -d %q", encodeStr)
		rawStr, _ := url.QueryUnescape(encodeStr)
		reqMsg += fmt.Sprintf(" -raw `%s`", rawStr)
	}

	return reqMsg
}

// restyResponseToCurl 返回日志
func restyResponseToCurl(resp *resty.Response) string {
	// 最大打印 1024 个字符
	body := string(resp.Body())
	if len(body) > 1024 {
		body = fmt.Sprintf("%s...(Total %s)", body[:1024], humanize.Bytes(uint64(len(body))))
	}

	respMsg := fmt.Sprintf("[%s] %s %s", resp.Status(), resp.Time(), body)
	return respMsg
}

func restyErrHook(r *resty.Request, err error) {
	klog.Infof("[%s] REQ: %s", store.RequestIDValue(r.RawRequest.Context()), restyReqToCurl(r))
	klog.Infof("[%s] RESP: [err] %s", store.RequestIDValue(r.RawRequest.Context()), err)
}

func restyAfterResponseHook(c *resty.Client, r *resty.Response) error {
	klog.Infof("[%s] REQ: %s", store.RequestIDValue(r.Request.Context()), restyReqToCurl(r.Request))
	klog.Infof("[%s] RESP: %s", store.RequestIDValue(r.Request.Context()), restyResponseToCurl(r))
	return nil
}

func restyBeforeRequestHook(c *resty.Client, r *http.Request) error {
	tracing.SetRequestIDValue(r, store.RequestIDValue(r.Context()))
	return nil
}

// GetClient : 新建Client, 设置公共参数，每次新建，cookies不复用
func GetClient() *resty.Client {
	if globalClient == nil {
		clientOnce.Do(func() {
			globalClient = resty.New().SetTimeout(timeout)
			globalClient = globalClient.SetDebug(false) // 更多详情, 可以开启为 true
			globalClient.SetDebugBodyLimit(1024)
			globalClient.OnAfterResponse(restyAfterResponseHook)
			globalClient.SetPreRequestHook(restyBeforeRequestHook)
			globalClient.OnError(restyErrHook)
			globalClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
		})
	}
	return globalClient
}

// BKResult 蓝鲸返回规范的结构体
type BKResult struct {
	Code    interface{} `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// UnmarshalBKResult 反序列化为蓝鲸返回规范
func UnmarshalBKResult(resp *resty.Response, data interface{}) error {
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("http code %d != 200", resp.StatusCode())
	}

	// 部分接口，如 usermanager 返回的content-type不是json, 需要手动Unmarshal
	bkResult := &BKResult{Data: data}
	if err := json.Unmarshal(resp.Body(), bkResult); err != nil {
		return err
	}

	if err := bkResult.ValidateCode(); err != nil {
		return err
	}

	return nil
}

// ValidateCode 返回结果是否OK
func (r *BKResult) ValidateCode() error {
	var resultCode int

	switch code := r.Code.(type) {
	case int:
		resultCode = code
	case float64:
		resultCode = int(code)
	case string:
		c, err := strconv.Atoi(code)
		if err != nil {
			return err
		}
		resultCode = c
	default:
		return errors.Errorf("conversion to int from %T not supported", code)
	}

	if resultCode != 0 {
		return errors.Errorf("resp code %d != 0, %s", resultCode, r.Message)
	}
	return nil
}
