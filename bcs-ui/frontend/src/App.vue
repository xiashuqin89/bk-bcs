<template>
  <div :class="systemCls" id="app">
    <Navigation @create-project="handleCreateProject">
      <router-view :key="routerKey" v-if="!isLoading" />
    </Navigation>
    <!-- 项目创建弹窗 -->
    <ProjectCreate v-model="showCreateDialog" :project-data="null"></ProjectCreate>
    <!-- 权限弹窗 -->
    <app-apply-perm ref="bkApplyPerm"></app-apply-perm>
    <!-- 登录弹窗 -->
    <BkPaaSLogin ref="login" :width="width" :height="height"></BkPaaSLogin>
    <SharedClusterTips ref="sharedClusterTips"></SharedClusterTips>
  </div>
</template>
<script>
import Navigation from '@/views/app/navigation.vue';
import ProjectCreate from '@/views/project/project-create.vue';
import SharedClusterTips from '@/views/app/shared-cluster-tips.vue';
import BkPaaSLogin from '@/views/app/login.vue';
import { bus } from '@/common/bus';
import { userPermsByAction } from '@/api/base';
import { newUserPermsByAction } from '@/api/modules/cluster-manager';
import AppApplyPerm from '@/views/app/apply-perm.vue';

export default {
  name: 'App',
  components: { Navigation, ProjectCreate, BkPaaSLogin, SharedClusterTips, AppApplyPerm },
  data() {
    return {
      isLoading: true,
      showCreateDialog: false,
    };
  },
  computed: {
    systemCls() {
      const platform = window.navigator.platform.toLowerCase();
      const cls = platform.indexOf('win') === 0 ? 'win' : 'mac';
      return this.$store.state.isEn ? `${cls} english` : cls;
    },
    routerKey() {
      // 切换不同界面时刷新路由
      return this.$route.params.projectCode || '';
    },
    width() {
      return this.$INTERNAL ? 700 : 400;
    },
    height() {
      return this.$INTERNAL ? 510 : 400;
    },
    projectList() {
      return this.$store.state.sideMenu.onlineProjectList;
    },
  },
  beforeCreate() {
    const allowDomains = (window.PREFERRED_DOMAINS || '').split(',');
    const item = allowDomains.find(item => item.trim() === location.hostname);
    if (!item && allowDomains[0]) {
      window.location.href = `//${allowDomains[0]}${location.pathname}`;
    }
    if (!this.$INTERNAL) {
      localStorage.setItem('appViewMode', 'namespace');
    }
  },
  created() {
    // 异步权限弹窗
    bus.$on('show-apply-perm-modal-async', async ({ $actionId, permCtx, resourceName, newPerms }) => {
      if (!this.$refs.bkApplyPerm) return;
      this.$refs.bkApplyPerm.dialogConf.isShow = true;
      this.$refs.bkApplyPerm.isLoading = true;
      const data = newPerms
        ? await newUserPermsByAction({
          $actionId,
          perm_ctx: permCtx,
        }).catch(() => ({}))
        : await userPermsByAction({
          $actionId,
          perm_ctx: permCtx,
        }).catch(() => ({}));
      if (data?.perms?.[$actionId]) {
        this.$bkMessage({
          theme: 'warning',
          message: this.$t('当前操作有权限，请刷新界面'),
        });
        this.$refs.bkApplyPerm.hide();
      } else {
        // eslint-disable-next-line camelcase
        this.$refs.bkApplyPerm.applyUrl = data?.perms?.apply_url;
        this.$refs.bkApplyPerm.actionList = [
          {
            action_id: $actionId,
            resource_name: resourceName,
          },
        ];
      }

      this.$refs.bkApplyPerm.isLoading = false;
    });
    // 权限弹窗
    bus.$on('show-apply-perm-modal', (data) => {
      if (!data) return;
      this.$refs.bkApplyPerm?.show(data);
    });
    // 登录弹窗
    bus.$on('close-login-modal', () => {
      window.location.reload();
    });
    bus.$on('show-shared-cluster-tips', () => {
      this.$refs.sharedClusterTips?.show();
    });
    window.addEventListener('message', (event) => {
      if (event.data === 'closeLoginModal') {
        window.location.reload();
      }
    });
    this.initBcsBaseData();
  },
  beforeDestroy() {
    bus.$off('show-apply-perm-modal');
    bus.$off('close-login-modal');
    bus.$off('show-shared-cluster-tips');
    bus.$off('show-apply-perm-modal-async');
  },
  mounted() {
    window.$loginModal = this.$refs.login;
  },
  methods: {
    // 初始化BCS基本数据
    async initBcsBaseData() {
      this.isLoading = true;
      await Promise.all([
        this.$store.dispatch('userInfo'),
        this.$store.dispatch('getProjectList'),
      ]).catch((err) => {
        console.error(err);
      });
      this.isLoading = false;
      document.title = this.$t('容器管理平台 | 腾讯蓝鲸智云');
    },
    handleCreateProject() {
      this.showCreateDialog = true;
    },
  },
};
</script>
<style lang="postcss">
    @import '@/css/reset.css';
    @import '@/css/app.css';
    @import '@/fonts/style.css';
    @import '@/css/main.css';

    .app-container {
        min-width: 1280px;
        min-height: 768px;
        position: relative;
        display: flex;
        background: #fafbfd;
        min-height: 100% !important;
        padding-top: 0;
    }
    .biz-guide-box {
        .desc {
            width: auto;
            margin: 0 auto 25px;
            position: relative;
            top: 12px;
        }
        .biz-app-form {
            .form-item {
                .form-item-inner {
                    width: 340px;
                    .bk-form-radio {
                        width: 115px;
                    }
                }
            }
        }
    }
    .biz-list-operation {
        .item {
            float: none;
        }
    }

    .not-ieg-user-infobox {
        .bk-dialog-style {
            width: 500px;
        }
    }
    .text-subtitle {
        color: #979BA5;
        font-size: 14px;
        text-align: center;
        margin-top: 14px;
    }
    .text-wrap {
        display: flex;
        align-items: center;
        justify-content: center;
        color: #3A84FF;
        font-size: 14px;
        margin-top: 12px;
    }
</style>
