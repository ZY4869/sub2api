import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

export function useAccountsEditActions(ctx: any) {
  const {
    activeEditAbortController,
    activeEditRequestToken,
    appStore,
    edAcc,
    editLoading,
    showEdit,
    t
  } = ctx

const isEditDetailAbortError = (error: unknown) => {
  const maybeError = error as { name?: string; code?: string } | null;
  return (
    maybeError?.name === "AbortError" ||
    maybeError?.name === "CanceledError" ||
    maybeError?.code === "ERR_CANCELED"
  );
};

const handleCloseEdit = () => {
  activeEditRequestToken.value += 1;
  activeEditAbortController.value?.abort();
  activeEditAbortController.value = null;
  editLoading.value = false;
  showEdit.value = false;
  edAcc.value = null;
};
const handleEdit = async (a: Account) => {
  activeEditAbortController.value?.abort();
  const currentAbortController = new AbortController();
  activeEditAbortController.value = currentAbortController;
  const requestToken = activeEditRequestToken.value + 1;
  activeEditRequestToken.value = requestToken;
  editLoading.value = true;
  edAcc.value = null;
  showEdit.value = true;

  try {
    const detail = await adminAPI.accounts.getById(a.id, {
      signal: currentAbortController.signal,
    });
    if (
      currentAbortController.signal.aborted ||
      activeEditRequestToken.value !== requestToken ||
      !showEdit.value
    ) {
      return;
    }
    edAcc.value = detail;
  } catch (error: any) {
    if (
      currentAbortController.signal.aborted ||
      activeEditRequestToken.value !== requestToken ||
      isEditDetailAbortError(error)
    ) {
      return;
    }
    console.error("Failed to load account detail for edit:", error);
    handleCloseEdit();
    appStore.showError(error?.message || t("common.error"));
    return;
  } finally {
    if (activeEditAbortController.value === currentAbortController) {
      activeEditAbortController.value = null;
    }
    if (activeEditRequestToken.value === requestToken) {
      editLoading.value = false;
    }
  }
};

  return {
    isEditDetailAbortError,
    handleCloseEdit,
    handleEdit
  }
}
