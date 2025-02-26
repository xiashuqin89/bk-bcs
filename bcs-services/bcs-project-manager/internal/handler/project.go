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

package handler

import (
	"context"

	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/actions/project"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/auth"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/component/iam"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/store"
	pm "github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/internal/store/project"
	proto "github.com/Tencent/bk-bcs/bcs-services/bcs-project-manager/proto/bcsproject"
	"github.com/Tencent/bk-bcs/bcs-services/pkg/bcs-auth/middleware"
)

// ProjectHandler xxx
type ProjectHandler struct {
	model store.ProjectModel
}

// NewProject return a project service hander
func NewProject(model store.ProjectModel) *ProjectHandler {
	return &ProjectHandler{
		model: model,
	}
}

// CreateProject implement for CreateProject interface
func (p *ProjectHandler) CreateProject(ctx context.Context,
	req *proto.CreateProjectRequest, resp *proto.ProjectResponse) error {
	// 创建项目
	ca := project.NewCreateAction(p.model)
	projectInfo, e := ca.Do(ctx, req)
	if e != nil {
		return e
	}
	authUser, ok := ctx.Value(middleware.AuthUserKey).(middleware.AuthUser)
	if ok && authUser.Username != "" {
		// 授权创建者项目编辑和查看权限
		iam.GrantResourceCreatorActions(authUser.Username, projectInfo.ProjectID, projectInfo.Name)
	}
	// 处理返回数据及权限
	setResp(resp, projectInfo)
	return nil
}

// GetProject get project info
func (p *ProjectHandler) GetProject(ctx context.Context,
	req *proto.GetProjectRequest, resp *proto.ProjectResponse) error {
	// 查询项目信息
	ga := project.NewGetAction(p.model)
	projectInfo, err := ga.Do(ctx, req)
	if err != nil {
		return err
	}
	// 处理返回数据及权限
	setResp(resp, projectInfo)
	return nil
}

// DeleteProject delete a project record
func (p *ProjectHandler) DeleteProject(ctx context.Context,
	req *proto.DeleteProjectRequest, resp *proto.ProjectResponse) error {
	// 删除项目
	da := project.NewDeleteAction(p.model)
	if err := da.Do(ctx, req); err != nil {
		return err
	}
	// 处理返回数据及权限
	setResp(resp, nil)
	return nil
}

// UpdateProject update a project record
func (p *ProjectHandler) UpdateProject(ctx context.Context,
	req *proto.UpdateProjectRequest, resp *proto.ProjectResponse) error {
	ua := project.NewUpdateAction(p.model)
	projectInfo, e := ua.Do(ctx, req)
	if e != nil {
		return e
	}
	// 处理返回数据及权限
	setResp(resp, projectInfo)
	return nil
}

// ListProjects list projects reocrds
func (p *ProjectHandler) ListProjects(ctx context.Context,
	req *proto.ListProjectsRequest, resp *proto.ListProjectsResponse) error {
	la := project.NewListAction(p.model)
	projects, e := la.Do(ctx, req)
	if e != nil {
		return e
	}
	authUser, ok := ctx.Value(middleware.AuthUserKey).(middleware.AuthUser)
	if ok && authUser.Username != "" {
		// with username
		// 获取 project id, 用以获取对应的权限
		ids := getProjectIDs(projects)
		perms, err := auth.ProjectIamClient.GetMultiProjectMultiActionPermission(
			authUser.Username, ids,
			[]string{auth.ProjectCreate, auth.ProjectView, auth.ProjectEdit, auth.ProjectDelete},
		)
		if err != nil {
			return err
		}
		// 处理返回
		setListPermsResp(resp, projects, perms)
		return nil
	}
	// without username
	setListPermsResp(resp, projects, nil)
	return nil
}

// ListAuthorizedProjects query authorized project info list
func (p *ProjectHandler) ListAuthorizedProjects(ctx context.Context,
	req *proto.ListAuthorizedProjReq, resp *proto.ListAuthorizedProjResp) error {
	lap := project.NewListAuthorizedProj(p.model)
	projects, e := lap.Do(ctx, req)
	if e != nil {
		return e
	}
	setListResp(resp, projects)
	return nil
}

// getProjectIDs 获取项目ID
func getProjectIDs(p *map[string]interface{}) []string {
	var ids []string
	results := (*p)["results"]
	if val, ok := results.([]*pm.Project); ok {
		for _, i := range val {
			ids = append(ids, i.ProjectID)
		}
	}
	return ids
}
