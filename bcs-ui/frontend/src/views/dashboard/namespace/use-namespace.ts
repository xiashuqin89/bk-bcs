import { ref } from '@vue/composition-api';
import { CUR_SELECT_NAMESPACE } from '@/common/constant';

import {
  getNamespaceList,
  deleteNamespace,
  updateNamespace,
  getClusterNamespaceVariable,
  updateClusterNamespaceVariable,
  createdNamespace,
} from '@/api/modules/project';

export function useNamespace() {
  const namespaceData = ref<any>([]);
  const variablesList = ref<any>([]);
  const variableLoading = ref(false);
  const namespaceLoading = ref(false);

  async function getNamespaceData(params) {
    if (!params || !params.$clusterId) return;
    namespaceLoading.value = true;
    const result = await getNamespaceList(params).catch(() => []);
    namespaceData.value = result;
    namespaceLoading.value = false;
    return result;
  }

  async function handleGetVariablesList(params) {
    variableLoading.value = true;
    const { results, total } = await getClusterNamespaceVariable(params)
      .catch(() => ({ results: [], total: 0 }));
    variablesList.value = results;
    variableLoading.value = false;
    return { results, total };
  }

  async function handleUpdateNameSpace(params) {
    const result = await updateNamespace(params).then(() => true)
      .catch(() => false);
    return result;
  }

  async function handleDeleteNameSpace(params) {
    const result = await deleteNamespace(params).then(() => true)
      .catch(() => false);
    return result;
  }

  async function handleUpdateVariablesList(params) {
    const result = await updateClusterNamespaceVariable(params).then(() => true)
      .catch(() => false);
    return result;
  }

  async function handleCreatedNamespace(params) {
    const result = await createdNamespace(params).then(() => true)
      .catch(() => false);
    return result;
  }

  return {
    namespaceLoading,
    namespaceData,
    variablesList,
    variableLoading,
    getNamespaceData,
    handleGetVariablesList,
    handleUpdateNameSpace,
    handleDeleteNameSpace,
    handleUpdateVariablesList,
    handleCreatedNamespace,
  };
}

export function useSelectItemsNamespace() {
  const namespaceValue = ref('');
  const namespaceLoading = ref(false);
  const namespaceList = ref<any[]>([]);

  const getNamespaceData = async ({ clusterId }) => {
    namespaceLoading.value = true;
    const data = await getNamespaceList({
      $clusterId: clusterId,
    });
    namespaceList.value = data || [];
    // 初始化默认选中命名空间
    const defaultSelectNamespace = namespaceList.value
      .find(data => data.name === localStorage.getItem(`${clusterId}-${CUR_SELECT_NAMESPACE}`));
    namespaceValue.value = defaultSelectNamespace?.name || namespaceList.value[0]?.name;
    localStorage.setItem(`${clusterId}-${CUR_SELECT_NAMESPACE}`, namespaceValue.value);
    namespaceLoading.value = false;
    return data;
  };

  return {
    namespaceLoading,
    namespaceValue,
    namespaceList,
    getNamespaceData,
  };
}
