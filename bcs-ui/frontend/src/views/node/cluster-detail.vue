<template>
  <!-- 集群详情 -->
  <div>
    <ContentHeader
      :title="curCluster.name"
      :desc="`(${curCluster.clusterID})`"
      :hide-back="isSingleCluster"
    ></ContentHeader>
    <div class="biz-content-wrapper">
      <div class="cluster-detail">
        <div class="cluster-detail-tab">
          <div
            v-for="item in tabItems"
            :key="item.com"
            :class="['item', { active: activeCom === item.com }]"
            @click="handleChangeActive(item)"
          >
            <span class="icon"><i :class="item.icon"></i></span>
            {{item.title}}
          </div>
        </div>
        <div class="cluster-detail-content">
          <component
            :is="activeCom"
            :node-menu="false"
            :cluster-id="clusterId"
            :hide-cluster-select="true"
            :selected-fields="[
              'container_count',
              'pod_count',
              'cpu_usage',
              'memory_usage',
              'disk_usage',
              'diskio_usage'
            ]"
          ></component>
        </div>
      </div>
    </div>
  </div>
</template>
<script lang="ts">
import { computed, defineComponent, ref, toRefs } from '@vue/composition-api';
import ContentHeader from '@/components/layout/Header.vue';
import node from './node.vue';
import overview from '@/views/cluster/overview.vue';
import info from '@/views/cluster/info.vue';
import useDefaultClusterId from './use-default-clusterId';
import $i18n from '@/i18n/i18n-setup';
import AutoScaler from './autoscaler.vue';

export default defineComponent({
  components: {
    info,
    node,
    overview,
    ContentHeader,
    AutoScaler,
  },
  props: {
    active: {
      type: String,
      default: 'overview',
    },
    clusterId: {
      type: String,
      default: '',
      required: true,
    },
  },
  setup(props, ctx) {
    const { $store, $router, $INTERNAL } = ctx.root;
    const { active, clusterId } = toRefs(props);
    const activeCom = ref(active.value);
    const curCluster = computed(() => $store.state.cluster.clusterList
      ?.find(item => item.clusterID === clusterId.value) || {});
    const tabItems = computed(() => {
      const items = [
        {
          icon: 'bcs-icon bcs-icon-bar-chart',
          title: $i18n.t('总览'),
          com: 'overview',
        },
        {
          icon: 'bcs-icon bcs-icon-list',
          title: $i18n.t('节点管理'),
          com: 'node',
        },
        {
          icon: 'bcs-icon bcs-icon-machine',
          title: $i18n.t('集群信息'),
          com: 'info',
        },
      ];
      if (cloudDetail.value.confInfo && !cloudDetail.value?.confInfo?.disableNodeGroup && !$INTERNAL) {
        items.push({
          icon: 'bcs-icon bcs-icon-kuosuorong',
          title: $i18n.t('弹性扩缩容'),
          com: 'AutoScaler',
        });
      }
      return items;
    });
    const handleChangeActive = (item) => {
      if (activeCom.value === item.com) return;
      activeCom.value = item.com;
      $router.replace({
        name: 'clusterDetail',
        query: {
          active: item.com,
        },
      });
    };
    const { isSingleCluster } = useDefaultClusterId();
    const cloudDetail = ref<any>({});
    const isLoading = ref(false);
    const handleGetCloudDetail = async () => {
      isLoading.value = true;
      cloudDetail.value = await $store.dispatch('clustermanager/cloudDetail', {
        $cloudId: curCluster.value.provider,
      });
      isLoading.value = false;
    };
    handleGetCloudDetail();
    return {
      isLoading,
      isSingleCluster,
      curCluster,
      tabItems,
      activeCom,
      handleChangeActive,
    };
  },
});
</script>
<style lang="postcss" scoped>
.cluster-detail {
    border: 1px solid #dfe0e5;
    &-tab {
        display: flex;
        height: 60px;
        line-height: 60px;
        border-bottom: 1px solid #dfe0e5;
        font-size: 14px;
        .item {
            display: flex;
            align-items: center;
            justify-content: center;
            min-width: 140px;
            cursor: pointer;
            &.active {
                color: #3a84ff;
                background-color: #fff;
                border-right: 1px solid #dfe0e5;
                border-left: 1px solid #dfe0e5;
                font-weight: 700;
                i {
                    font-weight: 700;
                }
            }
            &:first-child {
                border-left: none;
            }
            .icon {
                font-size: 16px;
                margin-right: 8px;
                width: 16px;
                height: 16px;
                display: flex;
                align-items: center;
            }
        }
    }
    &-content {
        background-color: #fff;
    }
}
</style>
