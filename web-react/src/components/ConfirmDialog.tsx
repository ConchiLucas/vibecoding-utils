import React from 'react';
import { AlertTriangle } from 'lucide-react';

interface ConfirmDialogProps {
  open: boolean;
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * 自定义确认弹窗组件，替代 window.confirm()
 * 解决 Wails WebView 中 window.confirm 不可用的问题
 */
export default function ConfirmDialog({
  open,
  title = '确认操作',
  message,
  confirmText = '确定',
  cancelText = '取消',
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center"
      onClick={onCancel}
    >
      {/* 遮罩 */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* 弹窗 */}
      <div
        className="relative bg-white rounded-2xl shadow-2xl max-w-md w-[90%] mx-4 overflow-hidden animate-in zoom-in-95 duration-200"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 标题栏 */}
        <div className="flex items-center gap-3 px-6 pt-6 pb-2">
          <div className="flex-shrink-0 w-10 h-10 rounded-full bg-amber-50 flex items-center justify-center">
            <AlertTriangle className="text-amber-500" size={20} />
          </div>
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
        </div>

        {/* 内容 */}
        <div className="px-6 py-4">
          <p className="text-gray-600 text-sm leading-relaxed">{message}</p>
        </div>

        {/* 按钮 */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 bg-gray-50 border-t border-gray-100">
          <button
            onClick={onCancel}
            className="px-5 py-2.5 rounded-xl text-sm font-medium text-gray-700 bg-white border border-gray-200 hover:bg-gray-100 hover:border-gray-300 transition-all duration-150"
          >
            {cancelText}
          </button>
          <button
            onClick={onConfirm}
            className="px-5 py-2.5 rounded-xl text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 shadow-sm hover:shadow-md transition-all duration-150"
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
