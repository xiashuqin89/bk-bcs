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

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path"

	"github.com/Tencent/bk-bcs/bcs-common/common/tcp/listener"
	"github.com/TencentBlueKing/bkmonitor-kits/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/route"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/docs" // docs xxx
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/api/metrics"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/api/pod"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/api/telemetry"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/config"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/rest"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/rest/middleware"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/rest/tracing"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/storegw"
)

// APIServer :
type APIServer struct {
	ctx      context.Context
	engine   *gin.Engine
	srv      *http.Server
	addrIPv6 string
}

// NewAPIServer :
func NewAPIServer(ctx context.Context, addr string, addrIPv6 string) (*APIServer, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	srv := &http.Server{Addr: addr, Handler: engine}

	s := &APIServer{
		ctx:      ctx,
		engine:   engine,
		srv:      srv,
		addrIPv6: addrIPv6,
	}
	s.newRoutes(engine)

	return s, nil
}

// Run :
func (a *APIServer) Run() error {
	dualStackListener := listener.NewDualStackListener()
	if err := dualStackListener.AddListenerWithAddr(a.srv.Addr); err != nil {
		return err
	}

	if a.addrIPv6 != "" {
		if err := dualStackListener.AddListenerWithAddr(a.addrIPv6); err != nil {
			return err
		}
		logger.Infof("api serve dualStackListener with ipv6: %s", a.addrIPv6)
	}

	return a.srv.Serve(dualStackListener)
}

// Close :
func (a *APIServer) Close() error {
	return a.srv.Shutdown(a.ctx)
}

// newRoutes xxx
// @Title     BCS-Monitor OpenAPI
// @BasePath  /bcsapi/v4/monitor/api/projects/:projectId/clusters/:clusterId
func (a *APIServer) newRoutes(engine *gin.Engine) {
	// 添加 X-Request-Id 头部
	requestIdMiddleware := requestid.New(
		requestid.WithGenerator(func() string {
			return tracing.RequestIdGenerator()
		}),
	)

	engine.Use(requestIdMiddleware, cors.Default())

	// openapi 文档
	// 访问 swagger/index.html, swagger/doc.json
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	engine.GET("/-/healthy", HealthyHandler)
	engine.GET("/-/ready", ReadyHandler)

	// 注册 HTTP 请求
	registerRoutes(engine.Group(""))
	registerMetricsRoutes(engine.Group(""))

	if config.G.Web.RoutePrefix != "" {
		registerRoutes(engine.Group(config.G.Web.RoutePrefix))
		registerMetricsRoutes(engine.Group(config.G.Web.RoutePrefix))
	}
	registerRoutes(engine.Group(path.Join(config.G.Web.RoutePrefix, config.APIServicePrefix)))
	registerMetricsRoutes(engine.Group(path.Join(config.G.Web.RoutePrefix, config.APIServicePrefix)))
}

func registerRoutes(engine *gin.RouterGroup) {
	// 日志相关接口
	engine.Use(middleware.AuthRequired())

	route := engine.Group("/projects/:projectId/clusters/:clusterId")
	{
		route.GET("/namespaces/:namespace/pods/:pod/containers", rest.RestHandlerFunc(pod.GetPodContainers))
		route.GET("/namespaces/:namespace/pods/:pod/logs", rest.RestHandlerFunc(pod.GetPodLog))
		route.GET("/namespaces/:namespace/pods/:pod/logs/download", rest.StreamHandler(pod.DownloadPodLog))

		// sse 实时日志流
		route.GET("/namespaces/:namespace/pods/:pod/logs/stream", rest.StreamHandler(pod.PodLogStream))

		// 蓝鲸监控采集器
		route.GET("/telemetry/bkmonitor_agent/", rest.STDRestHandlerFunc(telemetry.IsBKMonitorAgent))
	}
}

// registerMetricsRoutes metrics 相关接口
func registerMetricsRoutes(engine *gin.RouterGroup) {

	engine.Use(middleware.AuthRequired())

	// 命名规范
	// usage 代表 百分比
	// used 代表已使用
	// overview, info 数值量

	route := engine.Group("/metrics/projects/:projectCode/clusters/:clusterId")
	{
		route.GET("/overview", rest.RestHandlerFunc(metrics.GetClusterOverview))
		route.GET("/cpu_usage", rest.RestHandlerFunc(metrics.ClusterCPUUsage))
		route.GET("/memory_usage", rest.RestHandlerFunc(metrics.ClusterMemoryUsage))
		route.GET("/disk_usage", rest.RestHandlerFunc(metrics.ClusterDiskUsage))
		route.GET("/nodes/:ip/info", rest.RestHandlerFunc(metrics.GetNodeInfo))
		route.GET("/nodes/:ip/overview", rest.RestHandlerFunc(metrics.GetNodeOverview))
		route.GET("/nodes/:ip/cpu_usage", rest.RestHandlerFunc(metrics.GetNodeCPUUsage))
		route.GET("/nodes/:ip/memory_usage", rest.RestHandlerFunc(metrics.GetNodeMemoryUsage))
		route.GET("/nodes/:ip/network_receive", rest.RestHandlerFunc(metrics.GetNodeNetworkReceiveUsage))
		route.GET("/nodes/:ip/network_transmit", rest.RestHandlerFunc(metrics.GetNodeNetworkTransmitUsage))
		route.GET("/nodes/:ip/diskio_usage", rest.RestHandlerFunc(metrics.GetNodeDiskioUsage))
		route.POST("/namespaces/:namespace/pods/cpu_usage", rest.RestHandlerFunc(
			metrics.PodCPUUsage)) // 多个Pod场景, 可能有几十，上百Pod场景, 需要使用 Post 传递参数
		route.POST("/namespaces/:namespace/pods/memory_used", rest.RestHandlerFunc(metrics.PodMemoryUsed))
		route.POST("/namespaces/:namespace/pods/network_receive", rest.RestHandlerFunc(metrics.PodNetworkReceive))
		route.POST("/namespaces/:namespace/pods/network_transmit", rest.RestHandlerFunc(metrics.PodNetworkTransmit))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/cpu_usage",
			rest.RestHandlerFunc(metrics.ContainerCPUUsage))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/memory_used",
			rest.RestHandlerFunc(metrics.ContainerMemoryUsed))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/cpu_limit",
			rest.RestHandlerFunc(metrics.ContainerCPULimit))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/memory_limit",
			rest.RestHandlerFunc(metrics.ContainerMemoryLimit))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/disk_read_total",
			rest.RestHandlerFunc(metrics.ContainerDiskReadTotal))
		route.GET("/namespaces/:namespace/pods/:pod/containers/:container/disk_write_total",
			rest.RestHandlerFunc(metrics.ContainerDiskWriteTotal))
	}
}

// RegisterStoreGWRoutes 注册storegw http-sd
func RegisterStoreGWRoutes(gw *storegw.StoreGW) *route.Router {
	router := route.New()
	router.Get("/api/discovery/targetgroups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(gw.TargetGroups())
	})

	return router
}

// HealthyHandler 健康检查
func HealthyHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("OK"))
}

// ReadyHandler 健康检查
func ReadyHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("OK"))
}
