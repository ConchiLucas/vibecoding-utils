import { useState, useCallback } from 'react';

interface ConfirmState {
  open: boolean;
  title: string;
  message: string;
  confirmText: string;
  cancelText: string;
  resolve: ((value: boolean) => void) | null;
}

/**
 * useConfirm hook — 提供类似 window.confirm() 的 Promise API
 * 用法：
 *   const { confirm, dialogProps } = useConfirm();
 *   const ok = await confirm('确定要删除吗？');
 *   if (ok) { ... }
 *
 *   return <ConfirmDialog {...dialogProps} />;
 */
export function useConfirm() {
  const [state, setState] = useState<ConfirmState>({
    open: false,
    title: '确认操作',
    message: '',
    confirmText: '确定',
    cancelText: '取消',
    resolve: null,
  });

  const confirm = useCallback(
    (
      message: string,
      options?: { title?: string; confirmText?: string; cancelText?: string }
    ): Promise<boolean> => {
      return new Promise((resolve) => {
        setState({
          open: true,
          message,
          title: options?.title ?? '确认操作',
          confirmText: options?.confirmText ?? '确定',
          cancelText: options?.cancelText ?? '取消',
          resolve,
        });
      });
    },
    []
  );

  const handleConfirm = useCallback(() => {
    state.resolve?.(true);
    setState((prev) => ({ ...prev, open: false, resolve: null }));
  }, [state.resolve]);

  const handleCancel = useCallback(() => {
    state.resolve?.(false);
    setState((prev) => ({ ...prev, open: false, resolve: null }));
  }, [state.resolve]);

  const dialogProps = {
    open: state.open,
    title: state.title,
    message: state.message,
    confirmText: state.confirmText,
    cancelText: state.cancelText,
    onConfirm: handleConfirm,
    onCancel: handleCancel,
  };

  return { confirm, dialogProps };
}
