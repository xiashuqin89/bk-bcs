<template>
  <div class="add-node-template" v-bkloading="{ isLoading }">
    <bcs-resize-layout
      placement="right"
      ext-cls="template-resize"
      collapsible
      :initial-divide="400"
      :border="false"
      :min="3"
      disabled>
      <template #aside>
        <div class="node-template-aside">
          <div class="title">
            {{$t('Worker节点初始化配置说明')}}
          </div>
          <div class="content-wrapper">
            <div class="content">
              <div class="content-item">
                <div class="label">{{$t('内置变量')}}</div>
                <bcs-table class="mt15" :data="configList">
                  <bcs-table-column :label="$t('变量名')">
                    <template #default="{ row }">
                      <span
                        v-bk-tooltips.top="{
                          content: $t('点击复制变量名 {name}', { name: row.refer })
                        }"
                        @click="handleCopyVar(row)">
                        {{row.refer}}
                      </span>
                    </template>
                  </bcs-table-column>
                  <bcs-table-column
                    :label="$t('含义')"
                    prop="desc"
                    show-overflow-tooltip
                  ></bcs-table-column>
                </bcs-table>
              </div>
              <BcsMd class="mt15" :code="postActionDescMd"></BcsMd>
            </div>
          </div>
        </div>
      </template>
      <template #main>
        <bk-form :model="formData" :rules="rules" ref="formRef">
          <FormGroup :title="$t('基本信息')" :allow-toggle="false">
            <bk-form-item
              :label="$t('模板名称')"
              required
              property="name"
              error-display-type="normal">
              <bk-input class="mw524" :disabled="isEdit" v-model="formData.name"></bk-input>
            </bk-form-item>
            <bk-form-item
              :label="$t('描述')"
              property="desc"
              error-display-type="normal">
              <bk-input
                v-model="formData.desc"
                type="textarea"
                :rows="3"
                :maxlength="100"
                class="mw524"
              ></bk-input>
            </bk-form-item>
          </FormGroup>
          <FormGroup class="mt15" :title="$t('标签 & 污点')" :allow-toggle="false">
            <bk-form-item :label="$t('标签')">
              <KeyValue
                style="max-width: 420px;"
                :disable-delete-item="false"
                :min-item="0"
                v-model="formData.labels">
              </KeyValue>
            </bk-form-item>
            <bk-form-item
              :label="$t('污点')"
              property="taints"
              error-display-type="normal">
              <span class="add-btn" v-if="!formData.taints.length" @click="handleAddTaints">
                <i class="bk-icon icon-plus-circle-shape mr5"></i>
                {{$t('添加')}}
              </span>
              <Taints
                style="max-width: 600px;"
                v-else
                v-model="formData.taints"
                @add="handleValidateForm"
                @delete="handleValidateForm"
              ></Taints>
            </bk-form-item>
          </FormGroup>
          <FormGroup class="mt15" :title="$t('Kubelet组件参数配置')" :allow-toggle="false">
            <div style="padding: 0 24px">
              <div class="bcs-flex-between kubelet mb15">
                <span class="left"></span>
                <div class="right">
                  <bcs-input
                    v-model="searchValue"
                    :placeholder="$t('请输入参数名称')"
                    right-icon="bk-icon icon-search">
                  </bcs-input>
                  <i
                    class="bcs-icon bcs-icon-zhongzhishuju ml15"
                    v-bk-tooltips.top="$t('重置参数')"
                    @click="handleReset"></i>
                  <i
                    class="bcs-icon bcs-icon-yulan ml15"
                    v-bk-tooltips.top="$t('预览修改值')"
                    @click="handlePreview"></i>
                </div>
              </div>
              <bcs-table
                :data="curPageData"
                :pagination="pagination"
                v-bkloading="{ isLoading: loading }"
                @row-mouse-enter="handlekubeletMouseEnter"
                @page-change="pageChange"
                @page-limit-change="pageSizeChange">
                <bcs-table-column :label="$t('参数名称')" prop="flagName"></bcs-table-column>
                <bcs-table-column
                  :label="$t('参数说明')"
                  prop="flagDesc"
                  show-overflow-tooltip>
                </bcs-table-column>
                <bcs-table-column :label="$t('默认值')" prop="defaultValue"></bcs-table-column>
                <bcs-table-column :label="$t('当前值')">
                  <template #default="{ row }">
                    <div class="kubelet-value">
                      <InputType
                        v-if="editKey === row.flagName"
                        :type="row.flagType"
                        :options="row.flagValueList"
                        :range="row.range"
                        ref="editInputRef"
                        v-model="kubeletParams[row.flagName]"
                        @blur="handleEditBlur"
                        @enter="handleEditBlur"
                      ></InputType>
                      <template v-else>
                        <span>{{kubeletParams[row.flagName] || '--'}}</span>
                        <i
                          class="bcs-icon bcs-icon-edit2 ml5"
                          v-show="activeKubeletFlagName === row.flagName"
                          @click="handleEditkubelet(row)"></i>
                      </template>
                      <span
                        class="error-tips" v-if="row.regex
                          && kubeletParams[row.flagName]
                          && !new RegExp(row.regex.validator).test(kubeletParams[row.flagName])">
                        <i
                          v-bk-tooltips="row.regex ? row.regex.message : ''"
                          class="bk-icon icon-exclamation-circle-shape"></i>
                      </span>
                    </div>
                  </template>
                </bcs-table-column>
              </bcs-table>
            </div>
          </FormGroup>
          <FormGroup class="mt15" :title="$t('Worker节点初始化配置')" :allow-toggle="false">
            <bk-form-item :label="$t('前置初始化')" :desc="$t('暂时只支持bash脚本')">
              <bcs-input
                type="textarea"
                class="mt10 mw524"
                :rows="6"
                placeholder="#!/bin/bash"
                v-model="formData.preStartUserScript"></bcs-input>
            </bk-form-item>
            <bk-form-item
              :label="$t('后置初始化')" :desc="$t('简单脚本执行只支持bash脚本，复杂场景请使用标准运维流程执行')">
              <bcs-select class="mw524" :clearable="false" v-model="postActionType">
                <bcs-option id="simple" :name="$t('简单脚本执行')"></bcs-option>
                <bcs-option id="complex" :name="$t('标准运维流程执行')"></bcs-option>
              </bcs-select>
              <bcs-input
                type="textarea"
                class="mt10 mw524"
                :rows="6"
                placeholder="#!/bin/bash"
                v-model="formData.userScript"
                v-if="postActionType === 'simple'">
              </bcs-input>
            </bk-form-item>
            <bk-form-item :label="$t('标准运维流程')" v-if="postActionType === 'complex'">
              <div class="sops-wrapper">
                <bcs-select
                  :loading="bkSopsLoading"
                  :clearable="false"
                  class="mw524"
                  searchable
                  style="flex: 1"
                  v-model="bkSopsTemplateID">
                  <bcs-option
                    v-for="item in bkSopsList"
                    :key="item.templateID"
                    :id="item.templateID"
                    :name="item.templateName">
                  </bcs-option>
                </bcs-select>
                <span
                  class="ml10"
                  v-if="templateUrl"
                  v-bk-tooltips.top="$t('前往标准运维')"
                  @click="handleGotoSops">
                  <i class="bcs-icon bcs-icon-fenxiang"></i>
                </span>
                <span
                  class="ml10"
                  v-bk-tooltips.top="$t('刷新列表')"
                  @click="handleRefreshList">
                  <i class="bcs-icon bcs-icon-reset"></i>
                </span>
              </div>
              <div class="bk-sops-params mw524" v-bkloading="{ isLoading: sopsParamsLoading }">
                <div class="title">
                  <span v-bk-tooltips.top="{ content: $t('目前仅支持输入框类型参数') }" class="name">
                    {{$t('任务参数')}}
                  </span>
                </div>
                <div class="content">
                  <div class="content-item mb15" v-for="item in sopsParamsList" :key="item.key">
                    <div class="content-item-label">
                      <span
                        :class="{ desc: !!item.desc }"
                        v-bk-tooltips.top="{
                          content: item.desc,
                          disabled: !item.desc
                        }"
                      >{{item.name}}</span>
                    </div>
                    <bcs-input
                      behavior="simplicity"
                      :placeholder="$t('参数值留空代表使用标准运维流程参数默认值')"
                      v-model="sopsParams[item.key]">
                    </bcs-input>
                  </div>
                  <span
                    v-bk-tooltips="{
                      disabled: !isSopsParamsExitVar,
                      content: $t('调试时请勿使用内置变量')
                    }">
                    <bcs-button
                      theme="primary"
                      outline
                      :disabled="isSopsParamsExitVar"
                      @click="handleDebug">
                      {{$t('调试')}}
                    </bcs-button>
                  </span>
                </div>
              </div>
            </bk-form-item>
          </FormGroup>
        </bk-form>
        <div class="footer">
          <bcs-button class="mw88" theme="primary" :loading="btnLoading" @click="handleCreateOrUpdate">
            {{isEdit ? $t('保存') : $t('创建')}}
          </bcs-button>
          <bcs-button class="mw88 ml10" @click="handleCancel">{{$t('取消')}}</bcs-button>
        </div>
        <bcs-dialog
          :title="$t('预览修改值')"
          :show-footer="false"
          header-position="left"
          width="640"
          v-model="showPreview">
          <bcs-table :data="kubeletDiffData" :key="JSON.stringify(kubeletDiffData)">
            <bcs-table-column :label="$t('组件名称')" prop="moduleID"></bcs-table-column>
            <bcs-table-column :label="$t('组件参数')" prop="flagName"></bcs-table-column>
            <bcs-table-column :label="$t('修改前值')" prop="origin">
              <template #default="{ row }">
                {{row.origin || '--'}}
              </template>
            </bcs-table-column>
            <bcs-table-column :label="$t('修改后值')" prop="value"></bcs-table-column>
          </bcs-table>
        </bcs-dialog>
        <!-- 任务调试状态 -->
        <bcs-dialog
          :show-footer="false"
          :mask-close="false"
          width="400"
          v-model="showDebugStatus"
          :on-close="handleDebugDialogClose">
          <div class="task-status">
            <div
              class="loading-icon"
              v-show="['INITIALIZING', 'RUNNING'].includes(taskData.status)"
              v-bkloading="{
                isLoading: ['INITIALIZING', 'RUNNING'].includes(taskData.status),
                opacity: 1,
                theme: 'primary',
                mode: 'spin'
              }"></div>
            <template v-if="['INITIALIZING', 'RUNNING'].includes(taskData.status)">
              <div class="title mt15">{{$t('调试正在进行中')}}...</div>
              <div class="operator mt15">
                <bcs-button
                  text
                  size="small"
                  :disabled="!taskUrl"
                  @click="handleGotoTaskDetail">
                  {{$t('查看详情')}}
                </bcs-button>
              </div>
            </template>
            <template v-else-if="taskData.status === 'SUCCESS'">
              <div class="bcs-flex-center">
                <span class="status-icon success"><i class="bcs-icon bcs-icon-check-1"></i></span>
              </div>
              <div class="title mt20">{{$t('调试成功')}}</div>
              <div class="operator mt20">
                <bcs-button
                  class="mw88"
                  theme="primary"
                  :disabled="!taskUrl"
                  @click="handleGotoTaskDetail"
                >{{$t('查看详情')}}</bcs-button>
                <bcs-button
                  class="ml5"
                  style="min-width: 88px;"
                  @click="showDebugStatus = false">{{$t('完成')}}</bcs-button>
              </div>
            </template>
            <template v-else-if="taskData.status === 'FAILURE'">
              <div class="bcs-flex-center">
                <span class="status-icon failure"><i class="bcs-icon bcs-icon-close"></i></span>
              </div>
              <div class="title mt20">{{$t('调试失败')}}</div>
              <div class="operator mt20">
                <bcs-button
                  class="mw88"
                  theme="primary"
                  :disabled="!taskUrl"
                  @click="handleGotoTaskDetail"
                >{{$t('查看详情')}}</bcs-button>
                <bcs-button
                  class="mw88 ml5"
                  theme="primary"
                  @click="handleDebug">{{$t('再次调试')}}</bcs-button>
              </div>
            </template>
          </div>
        </bcs-dialog>
      </template>
    </bcs-resize-layout>
  </div>
</template>
<script lang="ts">
import { defineComponent, ref, computed, onMounted, watch } from '@vue/composition-api';
import FormGroup from '@/views/cluster/create-cluster/form-group.vue';
import KeyValue from './key-value.vue';
import Taints from './new-taints.vue';
import $store from '@/store/index';
import usePage from '@/views/dashboard/common/use-page';
import useSearch from '@/views/dashboard/common/use-search';
import $router from '@/router';
import $i18n from '@/i18n/i18n-setup';
import { copyText } from '@/common/util';
import useInterval from '@/views/dashboard/common/use-interval';
import BcsMd from '@/components/bcs-md/index.vue';
import postActionDescMd from './postaction-desc.md';
import InputType from './input-type.vue';

export default defineComponent({
  components: { FormGroup, KeyValue, Taints, BcsMd, InputType },
  props: {
    nodeTemplateID: {
      type: [String, Number],
      default: '',
    },
  },
  setup(props, ctx) {
    const { $bkMessage } = ctx.root;
    const curProject = computed(() => $store.state.curProject);
    const user = computed(() => $store.state.user);
    const isEdit = computed(() => !!props.nodeTemplateID);

    const postActionType = ref<'complex' | 'simple'>('simple');
    watch(postActionType, () => {
      if (postActionType.value === 'complex' && !bkSopsList.value.length) {
        handleGetbkSopsList();
      }
    });
    const formRef = ref<any>(null);
    const formData = ref({
      projectID: curProject.value.project_id,
      name: '',
      desc: '',
      labels: {},
      taints: [],
      preStartUserScript: '',
      userScript: '',
      extraArgs: {
        kubelet: '',
      },
      scaleOutExtraAddons: {
        plugins: {},
      },
    });
    const rules = ref({
      name: [{
        required: true,
        message: $i18n.t('必填项'),
        trigger: 'blur',
      }],
      taints: [{
        validator: () => formData.value.taints.every((item: any) => item.key && item.value && item.effect),
        message: $i18n.t('污点键、值和effect不能为空'),
        trigger: 'custom',

      }],
    });
    const handleValidateForm = async () => {
      await formRef.value?.validate();
    };

    // kubelet 组件参数
    const loading = ref(false);
    const editKey = ref('');
    const showPreview = ref(false);
    const kubeletParams = ref({});
    const originKubeletParams = ref<any>({});
    const kubeletDiffData = computed(() => Object.keys(kubeletParams.value).reduce<any[]>((pre, key) => {
      if (kubeletParams.value[key] !== ''
        && kubeletParams.value[key] !== originKubeletParams.value[key]) {
        pre.push({
          moduleID: 'kubelet',
          flagName: key,
          origin: originKubeletParams.value[key],
          value: kubeletParams.value[key],
        });
      }
      return pre;
    }, []));
    const kubeletList = ref<any[]>([]);
    const handleGetkubeletData = async () => {
      loading.value = true;
      kubeletList.value = await $store.dispatch('clustermanager/cloudModulesParamsList', {
        $cloudID: 'tencentCloud',
        $version: '1.20.6',
        $moduleID: 'kubelet',
      });
      loading.value = false;
    };
    const keys = ref(['flagName']);
    const { searchValue, tableDataMatchSearch } = useSearch(kubeletList, keys);
    const {
      pagination,
      curPageData,
      pageChange,
      pageSizeChange,
    } = usePage(tableDataMatchSearch);
    const editInputRef = ref<any>(null);
    const activeKubeletFlagName = ref('');
    const handlekubeletMouseEnter = (index, event, row) => {
      activeKubeletFlagName.value = row.flagName;
    };
    const handleEditkubelet = (row) => {
      editKey.value = row.flagName;
      setTimeout(() => {
        (ctx.refs.editInputRef as any).focus();
      }, 0);
    };
    const handleEditBlur = () => {
      editKey.value = '';
    };
    const handleReset = () => {
      kubeletParams.value = JSON.parse(JSON.stringify(originKubeletParams.value));
    };
    const handlePreview = () => {
      showPreview.value = true;
    };
    // 校验kubelet参数
    const validateKubeletParams = () => kubeletList.value.every((item) => {
      if (!kubeletParams.value[item.flagName] || !item.regex?.validator) return true;

      const regx = new RegExp(item.regex.validator);
      return regx.test(kubeletParams.value[item.flagName]);
    });

    // 添加污点
    const handleAddTaints = () => {
      (formData.value.taints as any[]).push({
        key: '',
        value: '',
        effect: 'PreferNoSchedule',
      });
    };

    // 获取标准运维任务
    const bkSopsLoading = ref(false);
    const bkSopsList = ref<any[]>([]);
    const bkSopsTemplateID = ref('');
    watch(bkSopsTemplateID, () => {
      if (!bkSopsTemplateID.value) return;
      // 清空数据
      sopsParams.value = {};
      sopsParamsList.value = [];
      handleGetSopsParams();
    });
    const handleGetbkSopsList = async () => {
      bkSopsLoading.value = true;
      bkSopsList.value = await $store.dispatch('clustermanager/bkSopsList', {
        $businessID: curProject.value.cc_app_id,
        operator: user.value.username,
        templateSource: 'business',
        scope: 'cmdb_biz',
      });
      if (!bkSopsTemplateID.value) {
        bkSopsTemplateID.value = bkSopsList.value[0]?.templateID;
      }
      bkSopsLoading.value = false;
    };
    const handleRefreshList = async () => {
      await handleGetbkSopsList();
      await handleGetSopsParams();
    };
    const sopsParamsLoading = ref(false);
    const sopsParams = ref({});
    const isSopsParamsExitVar = computed(() => Object.values(sopsParams.value).some(value => /{{.*}}/.test(value as string)));
    const sopsParamsList = ref([]);
    const templateUrl = ref('');
    const handleGetSopsParams = async () => {
      sopsParamsLoading.value = true;
      const data = await $store.dispatch('clustermanager/bkSopsParamsList', {
        $templateID: bkSopsTemplateID.value,
        $businessID: curProject.value.cc_app_id,
        operator: user.value.username,
        templateSource: 'business',
        scope: 'cmdb_biz',
      });
      sopsParamsList.value = data.values;
      // 优先还原历史详情数据
      sopsParams.value = JSON.parse(JSON.stringify(
        formData.value.scaleOutExtraAddons?.plugins?.[bkSopsTemplateID.value]?.params
        || data.values.reduce((pre, item) => {
          pre[item.key] = '';
          return pre;
        }, {}),
        (key, value) => {
          if (['template_biz_id', 'template_id', 'template_user'].includes(key)) {
            return undefined;
          }
          return value;
        },
      ));
      templateUrl.value = data.templateUrl;
      sopsParamsLoading.value = false;
    };
    const handleGotoSops = () => {
      window.open(templateUrl.value);
    };
    // 调试标准运维任务
    const showDebugStatus = ref(false);
    const taskData = ref<any>({});
    const taskUrl = computed(() => {
      const [stepID] = taskData.value.stepSequence || [];
      return taskData.value?.steps?.[stepID]?.params?.taskUrl;
    });
    const handleDebugDialogClose = () => {
      taskData.value = {};
      stop();
    };
    const handlePollTask = async () => {
      taskData.value = await $store.dispatch('clustermanager/taskDetail', {
        $taskId: taskData.value.taskID,
      });
      if (['SUCCESS', 'FAILURE'].includes(taskData.value.status)) {
        stop();
      }
    };
    const { start, stop } = useInterval(handlePollTask, 5000, true);
    const handleDebug = async () => {
      const { task } = await $store.dispatch('clustermanager/bkSopsDebug', {
        businessID: String(curProject.value.cc_app_id),
        templateID: String(bkSopsTemplateID.value),
        operator: user.value.username,
        templateSource: 'business',
        constant: {
          ...sopsParams.value,
        },
      });
      taskData.value = task || {};
      if (taskData.value.taskID) {
        showDebugStatus.value = true;
        start();
      }
    };
    // 跳转任务详情
    const handleGotoTaskDetail = () => {
      window.open(taskUrl.value);
    };

    // 配置说明
    const configLoading = ref(false);
    const configList = ref([]);
    const handleGetConfigList = async () => {
      configLoading.value = true;
      configList.value = await $store.dispatch('clustermanager/bkSopsTemplatevalues');
      configLoading.value = false;
    };
    const handleCopyVar = (row) => {
      copyText(row.refer);
      $bkMessage({
        theme: 'success',
        message: $i18n.t('复制成功'),
      });
    };

    // 创建和更新节点模板
    const btnLoading = ref(false);
    const handleCreateOrUpdate = async () => {
      const validate = await formRef.value?.validate();
      const validateKubelet = validateKubeletParams();
      if (!validate || !validateKubelet) return;

      btnLoading.value = true;
      // 后置初始化参数处理
      const data: Record<string, any> = {
        extraArgs: {
          kubelet: handleTransformParamsToKubelet(kubeletParams.value),
        },
      };
      if (postActionType.value === 'complex') {
        data.userScript = '';
        data.scaleOutExtraAddons = {
          postActions: [bkSopsTemplateID.value],
          plugins: {
            [bkSopsTemplateID.value]: {
              params: {
                template_biz_id: String(curProject.value.cc_app_id),
                template_id: bkSopsTemplateID.value,
                template_user: user.value.username,
                ...sopsParams.value,
              },
            },
          },
        };
      } else {
        data.scaleOutExtraAddons = {};
      }
      let result = false;
      if (isEdit.value) {
        result = await $store.dispatch('clustermanager/updateNodeTemplate', {
          $nodeTemplateId: props.nodeTemplateID,
          ...formData.value,
          ...data,
          updater: user.value.username,
        });
      } else {
        result = await $store.dispatch('clustermanager/createNodeTemplate', {
          ...formData.value,
          ...data,
          creator: user.value.username,
        });
      }
      if (result) {
        $bkMessage({
          theme: 'success',
          message: isEdit.value ? $i18n.t('编辑成功') : $i18n.t('创建成功'),
        });
        $router.push({ name: 'nodeTemplate' });
      }
      btnLoading.value = false;
    };
    const handleCancel = () => {
      $router.back();
    };

    // 获取详情
    const handleTransformKubeletToParams = (kubelet = '') => {
      if (!kubelet) return {};

      return kubelet.split(';').reduce((pre, current) => {
        const index = current.indexOf('=');
        const key = current.slice(0, index);
        const value = current.slice(index + 1, current.length);
        if (key) {
          pre[key] = value;
        }
        return pre;
      }, {}) || {};
    };
    const handleTransformParamsToKubelet = (params = {}) => Object.keys(params || {})
      .filter(key => params[key] !== '')
      .reduce<string[]>((pre, key) => {
      pre.push(`${key}=${params[key]}`);
      return pre;
    }, [])
      .join(';');
    const isLoading = ref(false);
    const handleGetDetail = async () => {
      if (!isEdit.value) return;

      isLoading.value = true;
      const data = await $store.dispatch('clustermanager/nodeTemplateDetail', {
        $nodeTemplateId: props.nodeTemplateID,
      });
      formData.value = data;
      // 处理标准运维相关回显参数
      postActionType.value = data.scaleOutExtraAddons?.postActions?.length ? 'complex' : 'simple';
      // eslint-disable-next-line camelcase, max-len
      bkSopsTemplateID.value = data.scaleOutExtraAddons?.plugins?.[data.scaleOutExtraAddons?.postActions?.[0]]?.params?.template_id;
      // 转换kubelet参数，便于回显
      kubeletParams.value = handleTransformKubeletToParams(formData.value?.extraArgs?.kubelet);
      // kubelet原始数据（用于diff）
      originKubeletParams.value = JSON.parse(JSON.stringify(kubeletParams.value));
      isLoading.value = false;
    };
    onMounted(() => {
      handleGetkubeletData();
      handleGetConfigList();
      handleGetDetail();
    });
    return {
      editInputRef,
      formRef,
      isLoading,
      btnLoading,
      bkSopsTemplateID,
      bkSopsLoading,
      sopsParamsLoading,
      sopsParamsList,
      sopsParams,
      templateUrl,
      bkSopsList,
      isEdit,
      postActionType,
      loading,
      rules,
      formData,
      searchValue,
      pagination,
      curPageData,
      editKey,
      showPreview,
      kubeletParams,
      pageChange,
      pageSizeChange,
      handleAddTaints,
      handleCancel,
      handleEditkubelet,
      handleCreateOrUpdate,
      handlePreview,
      handleReset,
      handleEditBlur,
      configList,
      handleCopyVar,
      handlekubeletMouseEnter,
      activeKubeletFlagName,
      kubeletDiffData,
      handleDebug,
      showDebugStatus,
      taskData,
      taskUrl,
      handleDebugDialogClose,
      handleGotoTaskDetail,
      postActionDescMd,
      handleGetbkSopsList,
      handleRefreshList,
      isSopsParamsExitVar,
      handleGotoSops,
      handleValidateForm,
    };
  },
});
</script>
<style lang="postcss" scoped>
.add-node-template {
    padding: 24px 0 24px 24px;
    max-height: calc(100vh - 172px);
    height: 100%;
    overflow: auto;
    >>> .mw524 {
        max-width: 524px;
    }
    >>> .mw920 {
        max-width: 920px;
    }
    >>> .add-btn {
        font-size: 14px;
        color: #3a84ff;
        cursor: pointer;
        display: flex;
        align-items: center;
        height: 32px;
    }
    >>> .sops-wrapper {
        display: flex;
        align-items: center;
        .bcs-icon {
            color: #3a84ff;
            cursor: pointer;
        }
    }
    .node-template-aside {
        border: 1px solid #dcdee5;
        border-left: none;
        height: 100%;
        overflow: auto;
        background: #fff;
        .title {
            height: 52px;
            padding: 0 16px;
            font-size: 16px;
            color: #313238;
            display: flex;
            align-items: center;
            box-shadow: inset 0 -1px 0 0 #DCDEE5;
        }
        .content-wrapper {
            max-height: calc(100vh - 275px);
            overflow: auto;
        }
        .content {
            padding: 16px 0;
            .content-item {
                padding: 0 24px;
                .label {
                    font-weight: 600;
                    line-height: 1.25;
                    font-size: 1em;
                    color: #24292e;
                }
            }
            >>> .bcs-md-preview {
                overflow: hidden;
            }
        }
    }
    .mw88 {
        min-width: 88px;
    }
    .kubelet-value {
        position: relative;
        height: 32px;
        display: flex;
        align-items: center;
        .bcs-icon-edit2 {
            cursor: pointer;
            &:hover {
                color: #3a84ff;
            }
        }
        .error-tips {
            position: absolute;
            z-index: 10;
            right: 8px;
            top: 8px;
            color: #ea3636;
            cursor: pointer;
            font-size: 16px;
            display: flex;
            background-color: #fff;
        }
    }
    .kubelet {
        .left {
            font-weight: Bold;
            font-size: 14px;
        }
        .right {
            display: flex;
            align-items: center;
            min-width: 300px;
            i {
                font-size: 14px;
                cursor: pointer;
                &:hover {
                    color: #3a84ff;
                }
            }
        }
    }
    >>> .bk-sops-params {
        margin-top: 12px;
        border: 1px solid #DCDEE5;
        border-radius: 2px;
        .title {
            background: #F5F7FA;
            border-bottom: 1px solid #DCDEE5;
            height: 36px;
            padding: 0 16px;
            display: flex;
            align-items: center;
            .name {
              border-bottom: 1px dashed #979ba5;
              line-height: 20px;
            }
        }
        .content {
            padding: 20px 16px;
            &-item-label {
                padding-left: 10px;
                line-height: 1;
                .desc {
                    border-bottom: 1px dashed #979ba5;
                    display: inline-block;
                    padding-bottom: 2px;
                }
            }
        }
    }
    .bcs-icon-copy:hover {
        color: #3a84ff;
        cursor: pointer;
    }
    .footer {
        position: fixed;
        bottom: 0px;
        height: 60px;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0 24px;
        background-color: #fff;
        border-top: 1px solid #dcdee5;
        box-shadow: 0 -2px 4px 0 rgb(0 0 0 / 5%);
        z-index: 200;
        right: 0;
        width: calc(100% - 261px);
        .btn {
            width: 88px;
        }
    }
}
>>> .task-status {
    .loading-icon {
        height: 70px;
    }
    .title {
        font-size: 20px;
        color: #313238;
        text-align: center;
    }
    .sub-title {
        text-align: center;
        font-size: 14px;
        color: #63656E;
    }
    .status-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 42px;
        height: 42px;
        border-radius: 50%;
        i {
            font-weight: bold;
        }
        &.success {
            background-color: #E5F6EA;
            color: #3FC06D;
        }
        &.failure {
            background-color: #FFDDDD;
            color: #EA3636;
        }
    }
    .operator {
        display: flex;
        justify-content: center;
    }
}
>>> .template-resize {
    height: auto;
    .bk-resize-layout-aside:after {
        width: 0;
    }
    .bk-resize-layout-aside {
        margin: 0 24px;
    }
    .bk-resize-layout-main {
        height: calc(100vh - 220px);
        overflow: auto;
    }
    &.bk-resize-layout-collapsed {
        .bk-resize-layout-aside {
            border-left: none;
            margin-right: 0px;
        }
    }
}
</style>
