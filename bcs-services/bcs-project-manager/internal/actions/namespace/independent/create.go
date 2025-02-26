/*
 * Tencent is pleased to support the open source community by making Blueking Container Service available.
 * Copyright (C) 2022 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * 	http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package independent

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/component/bcscc"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/component/clientset"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/logging"
	vvm "github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/store/variablevalue"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/util/errorx"
	quotautils "github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/util/quota"
	proto "github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/proto/bcsproject"
	"github.com/Tencent/bk-bcs/bcs-services/pkg/bcs-auth/middleware"
)

// CreateNamespace implement for CreateNamespace interface
func (c *IndependentNamespaceAction) CreateNamespace(ctx context.Context,
	req *proto.CreateNamespaceRequest, resp *proto.CreateNamespaceResponse) error {
	_, err := c.createNamespace(ctx, req)
	if err != nil {
		return errorx.NewClusterErr(err)
	}
	if req.GetQuota() != nil {
		if _, err := c.createResourceQuota(ctx, req); err != nil {
			return errorx.NewClusterErr(err)
		}
	}
	for _, variable := range req.GetVariables() {
		entity := &vvm.VariableValue{
			VariableID: variable.GetId(),
			Value:      variable.GetValue(),
		}
		if err := c.model.UpsertVariableValue(ctx, entity); err != nil {
			return err
		}
	}
	var creator string
	if authUser, err := middleware.GetUserFromContext(ctx); err == nil {
		creator = authUser.GetUsername()
	}
	go func() {
		if err := bcscc.CreateNamespace(req.GetProjectCode(), req.GetClusterID(), req.GetName(), creator); err != nil {
			// TODO: 添加日志告警
			logging.Error("create namespace %s/%s/%s in paas-cc failed, err: %s",
				req.GetProjectCode(), req.GetClusterID(), req.GetName(), err.Error())
		}
	}()
	return nil
}

func (c *IndependentNamespaceAction) createNamespace(ctx context.Context,
	req *proto.CreateNamespaceRequest) (*corev1.Namespace, error) {
	client, err := clientset.GetClientGroup().Client(req.GetClusterID())
	if err != nil {
		logging.Error("get clientset for cluster %s failed, err: %s", req.GetClusterID(), err.Error())
		return nil, err
	}
	ns := &corev1.Namespace{}
	ns.SetName(req.GetName())
	labels := map[string]string{}
	for _, label := range req.GetLabels() {
		labels[label.GetKey()] = label.GetValue()
	}
	ns.SetLabels(labels)
	annotations := map[string]string{}
	for _, annotation := range req.GetAnnotations() {
		annotations[annotation.GetKey()] = annotation.GetValue()
	}
	ns.SetAnnotations(annotations)
	return client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
}

func (c *IndependentNamespaceAction) createResourceQuota(ctx context.Context,
	req *proto.CreateNamespaceRequest) (*corev1.ResourceQuota, error) {
	client, err := clientset.GetClientGroup().Client(req.GetClusterID())
	if err != nil {
		logging.Error("get clientset for cluster %s failed, err: %s", req.GetClusterID(), err.Error())
		return nil, err
	}
	quota := &corev1.ResourceQuota{
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{},
		},
	}
	quota.SetName(req.GetName())
	quota.SetNamespace(req.GetName())

	if err := quotautils.LoadFromProto(quota, req.GetQuota()); err != nil {
		return nil, err
	}

	return client.CoreV1().ResourceQuotas(req.GetName()).Create(ctx, quota, metav1.CreateOptions{})
}
