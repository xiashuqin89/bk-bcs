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

package handler

import (
	"context"

	actionChart "github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/internal/actions/chart"
	helmmanager "github.com/Tencent/bk-bcs/bcs-services/bcs-helm-manager/proto/bcs-helm-manager"
)

// ListChartV1 provide the actions to list charts
func (hm *HelmManager) ListChartV1(ctx context.Context,
	req *helmmanager.ListChartV1Req, resp *helmmanager.ListChartV1Resp) error {

	defer recorder(ctx, "ListChartV1", req, resp)()
	action := actionChart.NewListChartV1Action(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// ListChartVersionV1 provide the actions to list chart versions
func (hm *HelmManager) ListChartVersionV1(ctx context.Context,
	req *helmmanager.ListChartVersionV1Req, resp *helmmanager.ListChartVersionV1Resp) error {

	defer recorder(ctx, "ListChartVersionV1", req, resp)()
	action := actionChart.NewListChartVersionV1Action(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// GetChartDetailV1 provide the actions the get chart detail
func (hm *HelmManager) GetChartDetailV1(ctx context.Context,
	req *helmmanager.GetChartDetailV1Req, resp *helmmanager.GetChartDetailV1Resp) error {

	defer recorder(ctx, "GetChartDetailV1", req, resp)()
	action := actionChart.NewGetChartDetailV1Action(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// ListChart provide the actions to list charts
func (hm *HelmManager) ListChart(ctx context.Context,
	req *helmmanager.ListChartReq, resp *helmmanager.ListChartResp) error {

	defer recorder(ctx, "ListChart", req, resp)()
	action := actionChart.NewListChartAction(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// ListChartVersion provide the actions to list chart versions
func (hm *HelmManager) ListChartVersion(ctx context.Context,
	req *helmmanager.ListChartVersionReq, resp *helmmanager.ListChartVersionResp) error {

	defer recorder(ctx, "ListChartVersion", req, resp)()
	action := actionChart.NewListChartVersionAction(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// GetChartDetail provide the actions the get chart detail
func (hm *HelmManager) GetChartDetail(ctx context.Context,
	req *helmmanager.GetChartDetailReq, resp *helmmanager.GetChartDetailResp) error {

	defer recorder(ctx, "GetChartDetail", req, resp)()
	action := actionChart.NewGetChartDetailAction(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// DeleteChart provide the actions to delete chart
func (hm *HelmManager) DeleteChart(ctx context.Context,
	req *helmmanager.DeleteChartReq, resp *helmmanager.DeleteChartResp) error {

	defer recorder(ctx, "DeleteChart", req, resp)()
	action := actionChart.NewDeleteChartAction(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}

// DeleteChartVersion provide the actions delete chart version
func (hm *HelmManager) DeleteChartVersion(ctx context.Context,
	req *helmmanager.DeleteChartVersionReq, resp *helmmanager.DeleteChartVersionResp) error {

	defer recorder(ctx, "DeleteChartVersion", req, resp)()
	action := actionChart.NewDeleteChartVersionAction(hm.model, hm.platform)
	return action.Handle(ctx, req, resp)
}
