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

package release

import (
	"context"
	"errors"
	"fmt"

	"github.com/Tencent/bk-bcs/bcs-common/common/blog"
	"github.com/Tencent/bk-bcs/bcs-common/pkg/odm/drivers"
	helmrelease "helm.sh/helm/v3/pkg/release"

	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/auth"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/common"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/operation"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/operation/actions"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/release"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/repo"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/store"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/store/entity"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/utils/contextx"
	helmmanager "github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/proto/bcs-helm-manager"
)

// NewUpgradeReleaseV1Action return a new UpgradeReleaseAction instance
func NewUpgradeReleaseV1Action(
	model store.HelmManagerModel, platform repo.Platform, releaseHandler release.Handler) *UpgradeReleaseV1Action {
	return &UpgradeReleaseV1Action{
		model:          model,
		platform:       platform,
		releaseHandler: releaseHandler,
	}
}

// UpgradeReleaseV1Action provides the actions to do upgrade release
type UpgradeReleaseV1Action struct {
	ctx context.Context

	model          store.HelmManagerModel
	platform       repo.Platform
	releaseHandler release.Handler

	req  *helmmanager.UpgradeReleaseV1Req
	resp *helmmanager.UpgradeReleaseV1Resp
}

// Handle the upgrading process
func (u *UpgradeReleaseV1Action) Handle(ctx context.Context,
	req *helmmanager.UpgradeReleaseV1Req, resp *helmmanager.UpgradeReleaseV1Resp) error {
	u.ctx = ctx
	u.req = req
	u.resp = resp

	if err := u.req.Validate(); err != nil {
		blog.Errorf("upgrade release failed, invalid request, %s, param: %v", err.Error(), u.req)
		u.setResp(common.ErrHelmManagerRequestParamInvalid, err.Error())
		return nil
	}

	if err := u.upgrade(); err != nil {
		blog.Errorf("upgrade release %s failed, clusterID: %s, namespace: %s, error: %s",
			u.req.GetName(), u.req.GetClusterID(), u.req.GetNamespace(), err.Error())
		u.setResp(common.ErrHelmManagerUpgradeActionFailed, err.Error())
		return nil
	}

	blog.Infof("dispatch release successfully, projectCode: %s, clusterID: %s, namespace: %s, name: %s, operator: %s",
		u.req.GetProjectCode(), u.req.GetClusterID(), u.req.GetNamespace(), u.req.GetName(), auth.GetUserFromCtx(u.ctx))
	u.setResp(common.ErrHelmManagerSuccess, "ok")
	return nil
}

func (u *UpgradeReleaseV1Action) upgrade() error {
	if err := u.saveDB(); err != nil {
		return fmt.Errorf("db error, %s", err.Error())
	}

	// dispatch release
	options := &actions.ReleaseUpgradeActionOption{
		Model:          u.model,
		Platform:       u.platform,
		ReleaseHandler: u.releaseHandler,
		ProjectCode:    u.req.GetProjectCode(),
		ProjectID:      contextx.GetProjectIDFromCtx(u.ctx),
		ClusterID:      u.req.GetClusterID(),
		Name:           u.req.GetName(),
		Namespace:      u.req.GetNamespace(),
		RepoName:       u.req.GetRepository(),
		ChartName:      u.req.GetChart(),
		Version:        u.req.GetVersion(),
		Values:         u.req.GetValues(),
		Args:           u.req.GetArgs(),
		Username:       auth.GetUserFromCtx(u.ctx),
	}
	action := actions.NewReleaseUpgradeAction(options)
	_, err := operation.GlobalOperator.Dispatch(action, releaseDefaultTimeout)
	if err != nil {
		return fmt.Errorf("dispatch failed, %s", err.Error())
	}
	return nil
}

func (u *UpgradeReleaseV1Action) saveDB() error {
	create := false
	_, err := u.model.GetRelease(u.ctx, u.req.GetClusterID(), u.req.GetNamespace(), u.req.GetName())
	if err != nil {
		if errors.Is(err, drivers.ErrTableRecordNotFound) {
			create = true
		} else {
			return err
		}
	}

	username := auth.GetUserFromCtx(u.ctx)
	if create {
		if err := u.model.CreateRelease(u.ctx, &entity.Release{
			Name:         u.req.GetName(),
			Namespace:    u.req.GetNamespace(),
			ClusterID:    u.req.GetClusterID(),
			ChartName:    u.req.GetChart(),
			ChartVersion: u.req.GetVersion(),
			Values:       u.req.GetValues(),
			Args:         u.req.GetArgs(),
			CreateBy:     username,
			Status:       helmrelease.StatusPendingUpgrade.String(),
		}); err != nil {
			return err
		}
	} else {
		rl := entity.M{
			entity.FieldKeyUpdateBy: username,
			entity.FieldKeyStatus:   helmrelease.StatusPendingUpgrade.String(),
		}
		if err := u.model.UpdateRelease(u.ctx, u.req.GetClusterID(), u.req.GetNamespace(),
			u.req.GetName(), rl); err != nil {

		}
	}
	return nil
}

func (u *UpgradeReleaseV1Action) setResp(err common.HelmManagerError, message string) {
	code := err.Int32()
	msg := err.ErrorMessage(message)
	u.resp.Code = &code
	u.resp.Message = &msg
	u.resp.Result = err.OK()
}
