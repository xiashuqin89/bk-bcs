/*
 * Tencent is pleased to support the open source community by making Blueking Container Service available.
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package app xxx
package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Tencent/bk-bcs/bcs-common/common/blog"
	"github.com/Tencent/bk-bcs/bcs-common/common/encrypt"
	"github.com/Tencent/bk-bcs/bcs-common/common/http/ipv6server"
	"github.com/Tencent/bk-bcs/bcs-common/common/ssl"
	"github.com/Tencent/bk-bcs/bcs-common/common/static"
	"github.com/Tencent/bk-bcs/bcs-common/common/types"
	"github.com/Tencent/bk-bcs/bcs-common/common/util"
	"github.com/Tencent/bk-bcs/bcs-common/common/version"
	"github.com/Tencent/bk-bcs/bcs-common/pkg/auth/iam"
	"github.com/Tencent/bk-bcs/bcs-common/pkg/odm/drivers/mongo"
	middleauth "github.com/Tencent/bk-bcs/bcs-services/pkg/bcs-auth/middleware"
	"github.com/gorilla/mux"
	ggRuntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	microRgt "github.com/micro/go-micro/v2/registry"
	microEtcd "github.com/micro/go-micro/v2/registry/etcd"
	microSvc "github.com/micro/go-micro/v2/service"
	microGrpc "github.com/micro/go-micro/v2/service/grpc"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	gCred "google.golang.org/grpc/credentials"

	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/auth"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/common"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/component/project"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/discovery"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/handler"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/operation"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/options"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/release"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/release/bcs"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/repo"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/repo/bkrepo"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/store"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/utils/envx"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/utils/runtimex"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/wrapper"
	helmmanager "github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/proto/bcs-helm-manager"
)

// HelmManager describe the helm-service manager instance
type HelmManager struct {
	opt *options.HelmManagerOptions

	microSvc  microSvc.Service
	microRgt  microRgt.Registry
	discovery *discovery.ModuleDiscovery

	// http service
	httpServer *ipv6server.IPv6Server

	// metric service
	metricServer *ipv6server.IPv6Server

	// tls config for helm manager service and client side
	tlsConfig       *tls.Config
	clientTLSConfig *tls.Config

	// mongo
	mongoOptions   *mongo.Options
	model          store.HelmManagerModel
	platform       repo.Platform
	releaseHandler release.Handler

	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	stopCh        chan struct{}
}

// NewHelmManager create a new helm manager
func NewHelmManager(opt *options.HelmManagerOptions) *HelmManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &HelmManager{
		opt:           opt,
		ctx:           ctx,
		ctxCancelFunc: cancel,
		stopCh:        make(chan struct{}),
	}
}

// Init helm manager server
func (hm *HelmManager) Init() error {
	hm.getServerAddress()
	for _, f := range []func() error{
		hm.initTLSConfig,
		hm.initModel,
		hm.initPlatform,
		hm.initReleaseHandler,
		hm.initRegistry,
		hm.initJWTClient,
		hm.initIAMClient,
		hm.InitComponentConfig,
		hm.initDiscovery,
		hm.initMicro,
		hm.initHTTPService,
		hm.initMetric,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

// Run helm manager server
func (hm *HelmManager) Run() error {
	// run the service
	if err := hm.microSvc.Run(); err != nil {
		blog.Fatal(err)
	}
	blog.CloseLogs()
	return nil
}

// RegistryStop registry stop signal
func (hm *HelmManager) RegistryStop() {
	go func() {
		// listening OS shutdown singal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan
		blog.Infof("Got OS shutdown signal, shutting down bcs-helm-manager server gracefully...")

		hm.ctxCancelFunc()
	}()
}

// initModel decode the connection info from the config and init a new store.HelmManagerModel
func (hm *HelmManager) initModel() error {
	if len(hm.opt.Mongo.Address) == 0 {
		return fmt.Errorf("mongo address cannot be empty")
	}
	if len(hm.opt.Mongo.Database) == 0 {
		return fmt.Errorf("mongo database cannot be empty")
	}
	password := hm.opt.Mongo.Password
	if password != "" && hm.opt.Mongo.Encrypted {
		realPwd, err := encrypt.DesDecryptFromBase([]byte(password))
		if err != nil {
			blog.Errorf("decrypt password failed, err %s", err.Error())
			return err
		}

		password = string(realPwd)
	}
	mongoOptions := &mongo.Options{
		Hosts:                 strings.Split(hm.opt.Mongo.Address, ","),
		ConnectTimeoutSeconds: int(hm.opt.Mongo.ConnectTimeout),
		AuthDatabase:          hm.opt.Mongo.AuthDatabase,
		Database:              hm.opt.Mongo.Database,
		Username:              hm.opt.Mongo.Username,
		Password:              password,
		MaxPoolSize:           uint64(hm.opt.Mongo.MaxPoolSize),
		MinPoolSize:           uint64(hm.opt.Mongo.MinPoolSize),
	}
	hm.mongoOptions = mongoOptions

	mongoDB, err := mongo.NewDB(mongoOptions)
	if err != nil {
		blog.Errorf("init mongo db failed, err %s", err.Error())
		return err
	}
	if err = mongoDB.Ping(); err != nil {
		blog.Errorf("ping mongo db failed, err %s", err.Error())
		return err
	}
	blog.Info("init mongo db successfully")
	hm.model = store.New(mongoDB)
	blog.Info("init store successfully")
	return nil
}

// initPlatform init a new repo.Platform, for handling operations to bk-repo
func (hm *HelmManager) initPlatform() error {
	password := hm.opt.Repo.Password
	if password != "" && hm.opt.Repo.Encrypted {
		realPwd, err := encrypt.DesDecryptFromBase([]byte(password))
		if err != nil {
			blog.Errorf("init platform decrypt password failed, err %s", err.Error())
			return err
		}

		password = string(realPwd)
	}

	hm.platform = bkrepo.New(repo.Config{
		URL:      hm.opt.Repo.URL,
		OciURL:   hm.opt.Repo.OciURL,
		AuthType: "Platform",
		Username: hm.opt.Repo.Username,
		Password: password,
	})
	blog.Infof("init repo platform successfully to %s", hm.opt.Repo.URL)
	return nil
}

// initReleaseHandler init a new release.Handler, for handling operations to helm-client
func (hm *HelmManager) initReleaseHandler() error {
	token := hm.opt.Release.Token
	if token != "" && hm.opt.Release.Encrypted {
		realToken, err := encrypt.DesDecryptFromBase([]byte(token))
		if err != nil {
			blog.Errorf("init release handler decode token failed: %s", err.Error())
			return err
		}

		token = string(realToken)
	}

	template, err := os.ReadFile(hm.opt.Release.KubeConfigTemplate)
	if err != nil {
		blog.Errorf("init release handler load template file %s failed: %s",
			hm.opt.Release.KubeConfigTemplate, err.Error())
		return err
	}

	// load patch template files from config
	patches, err := loadYamlFilesFromDir(hm.opt.Release.PatchDir)
	if err != nil {
		blog.Errorf("init release handler load patch dir %s failed: %s",
			hm.opt.Release.PatchDir, err.Error())
		return err
	}

	// load var template files from config
	vars, err := loadYamlFilesFromDir(hm.opt.Release.VarDir)
	if err != nil {
		blog.Errorf("init release handler load var dir %s failed: %s",
			hm.opt.Release.VarDir, err.Error())
		return err
	}

	hm.releaseHandler = bcs.New(release.Config{
		APIServer:          hm.opt.Release.APIServer,
		Token:              string(token),
		KubeConfigTemplate: string(template),
		HelmBinary:         hm.opt.Release.Binary,
		PatchTemplates:     patches,
		VarTemplates:       vars,
	})
	blog.Infof("init release handler successfully to %s", hm.opt.Release.APIServer)
	return nil
}

func (hm *HelmManager) initRegistry() error {
	etcdEndpoints := common.SplitAddrString(hm.opt.Etcd.EtcdEndpoints)
	etcdSecure := false

	var etcdTLS *tls.Config
	var err error
	if len(hm.opt.Etcd.EtcdCa) != 0 && len(hm.opt.Etcd.EtcdCert) != 0 && len(hm.opt.Etcd.EtcdKey) != 0 {
		etcdSecure = true
		etcdTLS, err = ssl.ClientTslConfVerity(hm.opt.Etcd.EtcdCa, hm.opt.Etcd.EtcdCert, hm.opt.Etcd.EtcdKey, "")
		if err != nil {
			return err
		}
	}

	blog.Infof("get etcd endpoints for registry: %v, with secure %t", etcdEndpoints, etcdSecure)

	hm.microRgt = microEtcd.NewRegistry(
		microRgt.Addrs(etcdEndpoints...),
		microRgt.Secure(etcdSecure),
		microRgt.TLSConfig(etcdTLS),
	)
	if err := hm.microRgt.Init(); err != nil {
		return err
	}
	return nil
}

func (hm *HelmManager) initDiscovery() error {
	hm.discovery = discovery.NewModuleDiscovery(common.ServiceDomain, hm.microRgt)
	blog.Info("init discovery for helm manager successfully")
	return nil
}

func (hm *HelmManager) initMicro() error {
	// server listen ip
	ipv4 := hm.opt.Address
	ipv6 := hm.opt.IPv6Address
	port := strconv.Itoa(int(hm.opt.Port))

	// service inject metadata to discovery center
	metadata := make(map[string]string)
	metadata[common.MicroMetaKeyHTTPPort] = strconv.Itoa(int(hm.opt.HTTPPort))

	// 适配单栈环境（ipv6注册地址不能是本地回环地址）
	if v := net.ParseIP(ipv6); v != nil && !v.IsLoopback() {
		metadata[types.IPV6] = net.JoinHostPort(ipv6, port)
	}

	authWrapper := middleauth.NewGoMicroAuth(auth.GetJWTClient()).
		EnableSkipHandler(auth.SkipHandler).
		EnableSkipClient(auth.SkipClient).
		SetCheckUserPerm(auth.CheckUserPerm)
	svc := microGrpc.NewService(
		microSvc.Name(common.ServiceDomain),
		microSvc.Metadata(metadata),
		microGrpc.WithTLS(hm.tlsConfig),
		microSvc.Address(net.JoinHostPort(ipv4, port)),
		microSvc.Registry(hm.microRgt),
		microSvc.Version(version.BcsVersion),
		microSvc.RegisterTTL(30*time.Second),
		microSvc.RegisterInterval(25*time.Second),
		microSvc.Context(hm.ctx),
		microSvc.BeforeStart(func() error {
			return nil
		}),
		microSvc.AfterStart(func() error {
			return hm.discovery.Start()
		}),
		microSvc.BeforeStop(func() error {
			hm.discovery.Stop()
			return nil
		}),
		microSvc.AfterStop(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			hm.httpServer.Shutdown(ctx)
			operation.GlobalOperator.TerminateOperation()
			operation.GlobalOperator.WaitTerminate(ctx, time.Second)
			return nil
		}),
		microSvc.WrapHandler(
			wrapper.RequestIDWrapper,
			authWrapper.AuthenticationFunc,
			wrapper.ParseProjectIDWrapper,
			authWrapper.AuthorizationFunc,
		),
	)
	svc.Init()

	if err := helmmanager.RegisterHelmManagerHandler(
		svc.Server(), handler.NewHelmManager(hm.model, hm.platform, hm.releaseHandler)); err != nil {
		blog.Errorf("register helm manager handler to micro failed: %s", err.Error())
		return nil
	}

	hm.microSvc = svc
	blog.Info("success to register helm manager handler to micro")
	return nil
}

func (hm *HelmManager) initHTTPService() error {
	router := mux.NewRouter()
	rmMux := ggRuntime.NewServeMux(
		ggRuntime.WithIncomingHeaderMatcher(runtimex.CustomHeaderMatcher),
		ggRuntime.WithMarshalerOption(ggRuntime.MIMEWildcard, &ggRuntime.JSONPb{OrigName: true, EmitDefaults: true}),
		ggRuntime.WithDisablePathLengthFallback(),
		ggRuntime.WithProtoErrorHandler(runtimex.CustomHTTPError),
	)

	grpcDialOpts := make([]grpc.DialOption, 0)
	if hm.tlsConfig != nil && hm.clientTLSConfig != nil {
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(gCred.NewTLS(hm.clientTLSConfig)))
	} else {
		grpcDialOpts = append(grpcDialOpts, grpc.WithInsecure())
	}
	err := helmmanager.RegisterHelmManagerGwFromEndpoint(
		context.TODO(),
		rmMux,
		hm.opt.Address+":"+strconv.Itoa(int(hm.opt.Port)),
		grpcDialOpts)
	if err != nil {
		blog.Errorf("register http service failed, err %s", err.Error())
		return fmt.Errorf("register http service failed, err %s", err.Error())
	}
	router.Handle("/{uri:.*}", rmMux)

	mux := http.NewServeMux()
	if len(hm.opt.Swagger.Dir) != 0 {
		blog.Info("swagger doc is enabled")
		mux.HandleFunc("/helmmanager/swagger/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path.Join(hm.opt.Swagger.Dir, strings.TrimPrefix(r.URL.Path, "/helmmanager/swagger/")))
		})
	}
	mux.Handle("/", router)
	blog.Info("register grpc service handler to path /")

	// server address
	addresses := []string{hm.opt.Address}
	if len(hm.opt.IPv6Address) > 0 {
		addresses = append(addresses, hm.opt.IPv6Address)
	}
	hm.httpServer = ipv6server.NewIPv6Server(addresses, strconv.Itoa(int(hm.opt.HTTPPort)), "", mux)

	go func() {
		var err error
		blog.Infof("start http gateway server on address %+v", addresses)
		if hm.tlsConfig != nil {
			hm.httpServer.TLSConfig = hm.tlsConfig
			err = hm.httpServer.ListenAndServeTLS("", "")
		} else {
			err = hm.httpServer.ListenAndServe()
		}
		if err != nil {
			blog.Errorf("start http gateway server failed, %s", err.Error())
			hm.stopCh <- struct{}{}
		}
	}()
	return nil
}

// initMetric brings up a service and listen on a metric port, for providing metric data
func (hm *HelmManager) initMetric() error {
	metricMux := http.NewServeMux()
	blog.Info("init metric handler")
	metricMux.Handle("/metrics", promhttp.Handler())
	// server address
	addresses := []string{hm.opt.Address}
	if len(hm.opt.IPv6Address) > 0 {
		addresses = append(addresses, hm.opt.IPv6Address)
	}
	hm.metricServer = ipv6server.NewIPv6Server(addresses, strconv.Itoa(int(hm.opt.MetricPort)), "", metricMux)

	go func() {
		var err error
		blog.Infof("start metric server on address %+v", addresses)
		if err = hm.metricServer.ListenAndServe(); err != nil {
			blog.Errorf("start metric server failed, %s", err.Error())
			hm.stopCh <- struct{}{}
		}
	}()

	operation.GlobalOperator.ReportOperatorCount()
	return nil
}

// initTLSConfig xxx
// init server and client tls config
func (hm *HelmManager) initTLSConfig() error {
	if len(hm.opt.ServerCert) != 0 && len(hm.opt.ServerKey) != 0 && len(hm.opt.ServerCa) != 0 {
		tlsConfig, err := ssl.ServerTslConfVerityClient(hm.opt.ServerCa, hm.opt.ServerCert,
			hm.opt.ServerKey, static.ServerCertPwd)
		if err != nil {
			blog.Errorf("load helm manager server tls config failed, err %s", err.Error())
			return err
		}
		hm.tlsConfig = tlsConfig
		blog.Info("load helm manager server tls config successfully")
	}

	if len(hm.opt.ClientCert) != 0 && len(hm.opt.ClientKey) != 0 && len(hm.opt.ClientCa) != 0 {
		tlsConfig, err := ssl.ClientTslConfVerity(hm.opt.ClientCa, hm.opt.ClientCert,
			hm.opt.ClientKey, static.ClientCertPwd)
		if err != nil {
			blog.Errorf("load helm manager client tls config failed, err %s", err.Error())
			return err
		}
		hm.clientTLSConfig = tlsConfig
		blog.Info("load helm manager client tls config successfully")
	}
	return nil
}

func (hm *HelmManager) initJWTClient() error {
	conf := auth.JWTClientConfig{
		Enable:         hm.opt.JWT.Enable,
		PublicKey:      hm.opt.JWT.PublicKey,
		PublicKeyFile:  hm.opt.JWT.PublicKeyFile,
		PrivateKey:     hm.opt.JWT.PrivateKey,
		PrivateKeyFile: hm.opt.JWT.PrivateKeyFile,
		ExemptClients:  hm.opt.ExemptClients.ClientIDs,
	}
	if _, err := auth.NewJWTClient(conf); err != nil {
		blog.Error("init jwt client error, %s", err.Error())
		return err
	}
	blog.Info("init jwt client successfully")
	return nil
}

// initIAMClient xxx
// init iam client for perm
func (hm *HelmManager) initIAMClient() error {
	iamClient, err := iam.NewIamClient(&iam.Options{
		SystemID:    hm.opt.IAM.SystemID,
		AppCode:     hm.opt.IAM.AppCode,
		AppSecret:   hm.opt.IAM.AppSecret,
		External:    hm.opt.IAM.External,
		GateWayHost: hm.opt.IAM.GatewayServer,
		IAMHost:     hm.opt.IAM.IAMServer,
		BkiIAMHost:  hm.opt.IAM.BkiIAMServer,
		Metric:      hm.opt.IAM.Metric,
		Debug:       hm.opt.IAM.Debug,
	})
	if err != nil {
		return err
	}
	auth.IAMClient = iamClient
	auth.InitPermClient(iamClient)
	blog.Info("init iam client successfully")
	return nil
}

func loadYamlFilesFromDir(dir string) ([]*release.File, error) {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	r := make([]*release.File, 0, len(fs))
	for _, f := range fs {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}

		r = append(r, &release.File{
			Name:    f.Name(),
			Content: data,
		})
	}

	return r, nil
}

func (hm *HelmManager) getServerAddress() error {
	// 通过环境变量获取LocalIP，这里是用的是podIP
	if hm.opt.UseLocalIP && envx.LocalIP != "" {
		hm.opt.Address = envx.LocalIP
		hm.opt.InsecureAddress = envx.LocalIP
	}
	hm.opt.IPv6Address = util.InitIPv6Address(hm.opt.IPv6Address)
	return nil
}

// InitComponentConfig init component config
func (hm *HelmManager) InitComponentConfig() error {
	err := project.NewClient(hm.clientTLSConfig, hm.microRgt)
	if err != nil {
		blog.Error("init project client error, %s", err.Error())
		return err
	}
	blog.Info("init project client successfully")
	return nil
}
