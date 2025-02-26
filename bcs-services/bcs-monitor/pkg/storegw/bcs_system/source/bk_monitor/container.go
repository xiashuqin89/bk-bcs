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
 */

package bkmonitor

import (
	"context"
	"strings"
	"time"

	bcsmonitor "github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/component/bcs_monitor"
	"github.com/Tencent/bk-bcs/bcs-services/bcs-monitor/pkg/storegw/bcs_system/source/base"
	"github.com/prometheus/prometheus/prompb"
)

// handleContainerMetric 处理公共函数
func (m *BKMonitor) handleContainerMetric(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, promql string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {

	params := map[string]interface{}{
		"clusterId":     clusterId,
		"namespace":     namespace,
		"podname":       podname,
		"containerName": strings.Join(containerNameList, "|"),
		"provider":      PROVIDER,
	}

	matrix, _, err := bcsmonitor.QueryRangeMatrix(ctx, projectId, promql, params, start, end, step)
	if err != nil {
		return nil, err
	}

	return base.MatrixToSeries(matrix), nil
}

// GetContainerCPUUsage 容器CPU使用率
func (m *BKMonitor) GetContainerCPUUsage(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql :=
		`sum by(container_name) (rate(container_cpu_usage_seconds_total{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s}[2m])) * 100`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}

// GetContainerMemoryUsed 容器内存使用率
func (m *BKMonitor) GetContainerMemoryUsed(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql :=
		`max by(container_name) (container_memory_working_set_bytes{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s})`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}

// GetContainerCPULimit 容器CPU限制
func (m *BKMonitor) GetContainerCPULimit(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql := `
		label_replace(max by(container_name) (container_spec_cpu_quota{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s}) / 1000 > 0,
		"container_name", "cpu_limit", "container_name", ".*")`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}

// GetContainerMemoryLimit 容器内存限制
func (m *BKMonitor) GetContainerMemoryLimit(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql := `
		label_replace(max by(container_name) (container_spec_memory_limit_bytes{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s}) > 0,
		"container_name", "memory_limit", "container_name", ".*")`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}

// GetContainerDiskReadTotal 容器磁盘读
func (m *BKMonitor) GetContainerDiskReadTotal(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql :=
		`sum by(container_name) (container_fs_reads_bytes_total{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s})`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}

// GetContainerDiskWriteTotal 容器磁盘写
func (m *BKMonitor) GetContainerDiskWriteTotal(ctx context.Context, projectId, clusterId, namespace, podname string,
	containerNameList []string, start, end time.Time, step time.Duration) ([]*prompb.TimeSeries, error) {
	promql :=
		`sum by(container_name) (container_fs_writes_bytes_total{cluster_id="%<clusterId>s", namespace="%<namespace>s", pod_name=~"%<podname>s", container_name=~"%<containerName>s", %<provider>s})`

	return m.handleContainerMetric(ctx, projectId, clusterId, namespace, podname, containerNameList, promql, start, end,
		step)
}
