import React, { useState, useEffect, useCallback, useMemo } from 'react';
import './TableDataPreview.css';
import { X, Loader2, ChevronUp, ChevronDown, Database, FileSpreadsheet, Pencil, Save, RotateCcw, AlertTriangle, Filter, Code2, Copy } from 'lucide-react';
import { getRemoteTablePreview, updateRemoteTableRecord, getRemoteTableDDL, ColumnPreview } from '@/api/sysConnection';
import toast from 'react-hot-toast';

interface TableDataPreviewProps {
  open: boolean;
  onClose: () => void;
  connectionId: number;
  databaseName: string;
  tableName: string;
}

interface PendingChange {
  name: string;
  before: string;
  after: string;
  primaryKey: boolean;
}

interface ActiveFilter {
  column: string;
  value: string;
}

export default function TableDataPreview({
  open,
  onClose,
  connectionId,
  databaseName,
  tableName,
}: TableDataPreviewProps) {
  const [columns, setColumns] = useState<ColumnPreview[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [draftValues, setDraftValues] = useState<Record<string, string>>({});
  const [activeFilterColumn, setActiveFilterColumn] = useState('');
  const [filterValue, setFilterValue] = useState('');
  const [appliedFilter, setAppliedFilter] = useState<ActiveFilter | null>(null);
  const [ddlOpen, setDdlOpen] = useState(false);
  const [ddlLoading, setDdlLoading] = useState(false);
  const [ddlSQL, setDdlSQL] = useState('');
  const [ddlError, setDdlError] = useState('');

  const resetDraftValues = useCallback((nextColumns: ColumnPreview[]) => {
    const nextDraft: Record<string, string> = {};
    nextColumns.forEach(col => {
      nextDraft[col.name] = col.value ?? '';
    });
    setDraftValues(nextDraft);
  }, []);

  const fetchRecord = useCallback(async (newOffset: number, filter: ActiveFilter | null) => {
    setLoading(true);
    try {
      const res = await getRemoteTablePreview({
        ID: connectionId,
        databaseName,
        tableName,
        offset: newOffset,
        ...(filter ? { filterColumn: filter.column, filterValue: filter.value } : {}),
      });
      if (res.code === 0 && res.data) {
        const nextColumns = res.data.columns || [];
        setColumns(nextColumns);
        setTotal(res.data.total);
        setOffset(res.data.offset);
        setEditing(false);
        setConfirmOpen(false);
        resetDraftValues(nextColumns);
      } else {
        toast.error(res.msg || '获取记录失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('获取记录失败');
    } finally {
      setLoading(false);
    }
  }, [connectionId, databaseName, tableName, resetDraftValues]);

  useEffect(() => {
    if (open && connectionId && tableName) {
      setOffset(0);
      setColumns([]);
      setTotal(0);
      setEditing(false);
      setConfirmOpen(false);
      setDraftValues({});
      setActiveFilterColumn('');
      setFilterValue('');
      setAppliedFilter(null);
      setDdlOpen(false);
      setDdlSQL('');
      setDdlError('');
      fetchRecord(0, null);
    }
  }, [open, connectionId, databaseName, tableName, fetchRecord]);

  // Keyboard navigation
  useEffect(() => {
    if (!open) return;
    const handleKey = (e: KeyboardEvent) => {
      if (confirmOpen) {
        if (e.key === 'Escape') {
          setConfirmOpen(false);
        }
        return;
      }
      if (ddlOpen) {
        if (e.key === 'Escape') {
          setDdlOpen(false);
        }
        return;
      }
      if (editing) {
        if (e.key === 'Escape') {
          e.preventDefault();
          setEditing(false);
          resetDraftValues(columns);
        }
        return;
      }
      if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
        e.preventDefault();
        if (offset > 0) {
          fetchRecord(offset - 1, appliedFilter);
        }
      } else if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
        e.preventDefault();
        if (offset < total - 1) {
          fetchRecord(offset + 1, appliedFilter);
        }
      } else if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [open, offset, total, fetchRecord, onClose, editing, confirmOpen, ddlOpen, columns, resetDraftValues, appliedFilter]);

  const pendingChanges = useMemo(() => {
    return columns
      .map(col => {
        const nextValue = draftValues[col.name] ?? '';
        const currentValue = col.value ?? '';
        if (nextValue === currentValue) return null;
        return {
          name: col.name,
          before: col.isNull ? 'NULL' : currentValue === '' ? "''" : currentValue,
          after: nextValue === '' ? "''" : nextValue,
          primaryKey: Boolean(col.primaryKey),
        };
      })
      .filter((change): change is PendingChange => Boolean(change));
  }, [columns, draftValues]);

  const hasChanges = pendingChanges.length > 0;

  const handleStartEdit = () => {
    resetDraftValues(columns);
    setEditing(true);
  };

  const handleCancelEdit = () => {
    resetDraftValues(columns);
    setEditing(false);
    setConfirmOpen(false);
  };

  const handleSaveClick = () => {
    if (!hasChanges) {
      toast.error('没有需要保存的修改');
      return;
    }
    setConfirmOpen(true);
  };

  const handleConfirmSave = async () => {
    setSaving(true);
    try {
      const res = await updateRemoteTableRecord({
        ID: connectionId,
        databaseName,
        tableName,
        offset,
        ...(appliedFilter ? { filterColumn: appliedFilter.column, filterValue: appliedFilter.value } : {}),
        changes: pendingChanges.map(change => ({
          name: change.name,
          value: draftValues[change.name] ?? '',
        })),
      });
      if (res.code === 0 && res.data) {
        const nextColumns = res.data.columns || [];
        setColumns(nextColumns);
        setTotal(res.data.total);
        setOffset(res.data.offset);
        resetDraftValues(nextColumns);
        setEditing(false);
        setConfirmOpen(false);
        toast.success(res.msg || '修改成功');
      } else {
        toast.error(res.msg || '修改失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('修改失败');
    } finally {
      setSaving(false);
    }
  };

  const renderValue = (col: ColumnPreview) => {
    if (col.isNull) return 'NULL';
    if (col.value === '') return "''";
    return col.value;
  };

  const handleFilterColumnClick = (column: string) => {
    if (editing && hasChanges) {
      toast.error('请先保存或取消当前修改');
      return;
    }
    setActiveFilterColumn(column);
    setFilterValue(appliedFilter?.column === column ? appliedFilter.value : '');
    if (appliedFilter && appliedFilter.column !== column) {
      setAppliedFilter(null);
      fetchRecord(0, null);
    }
  };

  const handleApplyFilter = () => {
    if (!activeFilterColumn) return;
    const nextFilter = { column: activeFilterColumn, value: filterValue };
    setAppliedFilter(nextFilter);
    fetchRecord(0, nextFilter);
  };

  const handleClearFilter = () => {
    setAppliedFilter(null);
    setActiveFilterColumn('');
    setFilterValue('');
    fetchRecord(0, null);
  };

  const handleOpenDDL = async () => {
    setDdlOpen(true);
    setDdlError('');
    if (ddlSQL) return;

    setDdlLoading(true);
    try {
      const res = await getRemoteTableDDL({
        ID: connectionId,
        databaseName,
        tableName,
      });
      if (res.code === 0 && res.data?.sql) {
        setDdlSQL(res.data.sql);
      } else {
        setDdlError(res.msg || '获取建表 SQL 失败');
      }
    } catch (e) {
      console.error(e);
      setDdlError('获取建表 SQL 失败');
    } finally {
      setDdlLoading(false);
    }
  };

  const handleCopyDDL = async () => {
    if (!ddlSQL) return;
    try {
      await navigator.clipboard.writeText(ddlSQL);
      toast.success('已复制建表 SQL');
    } catch (e) {
      console.error(e);
      toast.error('复制失败');
    }
  };

  if (!open) return null;

  return (
    <div className="tdp-overlay" onClick={onClose}>
      <div className="tdp-modal" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="tdp-header">
          <div className="tdp-header-left">
            <h2 className="tdp-title">
              <FileSpreadsheet size={18} className="tdp-title-icon" />
              数据预览
            </h2>
            <span className="tdp-subtitle">{databaseName} → {tableName}</span>
          </div>
          <button className="tdp-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        {/* Navigation */}
        <div className="tdp-nav">
          <span className="tdp-nav-info">
            {total > 0 ? (
              <>第 <strong>{offset + 1}</strong> / <strong>{total}</strong> 条记录</>
            ) : (
              '暂无记录'
            )}
            {editing && <em>编辑中</em>}
            {appliedFilter && <em>{appliedFilter.column} = {appliedFilter.value || "''"}</em>}
          </span>
          <div className="tdp-nav-actions">
            {activeFilterColumn && (
              <form
                className="tdp-filter-form"
                onSubmit={e => {
                  e.preventDefault();
                  handleApplyFilter();
                }}
              >
                <span className="tdp-filter-label">
                  <Filter size={14} />
                  {activeFilterColumn}
                </span>
                <input
                  className="tdp-filter-input"
                  value={filterValue}
                  placeholder="输入过滤值"
                  spellCheck={false}
                  onChange={e => setFilterValue(e.target.value)}
                />
                <button className="tdp-filter-btn" type="submit" disabled={loading || saving}>
                  过滤
                </button>
                <button className="tdp-filter-clear" type="button" disabled={loading || saving} onClick={handleClearFilter} title="清除过滤">
                  <X size={14} />
                </button>
              </form>
            )}
            {editing ? (
              <div className="tdp-edit-actions">
                <button
                  className="tdp-action-btn tdp-action-primary"
                  disabled={loading || saving || !hasChanges}
                  onClick={handleSaveClick}
                  title="保存修改"
                >
                  <Save size={16} />
                  保存
                </button>
                <button
                  className="tdp-action-btn"
                  disabled={saving}
                  onClick={handleCancelEdit}
                  title="取消编辑"
                >
                  <RotateCcw size={16} />
                  取消
                </button>
              </div>
            ) : (
              <>
                <button
                  className="tdp-action-btn"
                  disabled={loading}
                  onClick={handleOpenDDL}
                  title="查看建表 SQL"
                >
                  <Code2 size={16} />
                  建表 SQL
                </button>
                <button
                  className="tdp-action-btn"
                  disabled={loading || columns.length === 0 || total === 0}
                  onClick={handleStartEdit}
                  title="编辑字段值"
                >
                  <Pencil size={16} />
                  编辑
                </button>
              </>
            )}
            <div className="tdp-nav-buttons">
              <button
                className="tdp-nav-btn"
                disabled={loading || editing || offset <= 0}
                onClick={() => fetchRecord(offset - 1, appliedFilter)}
                title="上一条 (↑)"
              >
                <ChevronUp size={18} />
              </button>
              <button
                className="tdp-nav-btn"
                disabled={loading || editing || offset >= total - 1}
                onClick={() => fetchRecord(offset + 1, appliedFilter)}
                title="下一条 (↓)"
              >
                <ChevronDown size={18} />
              </button>
            </div>
          </div>
        </div>

        {/* Body */}
        <div className="tdp-body">
          {loading ? (
            <div className="tdp-loading">
              <Loader2 size={24} className="tdp-spinner" />
              <span>加载中...</span>
            </div>
          ) : columns.length === 0 ? (
            <div className="tdp-empty">
              <Database size={36} className="tdp-empty-icon" />
              <span>暂无数据</span>
            </div>
          ) : (
            <table className="tdp-table">
              <thead>
                <tr>
                  <th>字段名</th>
                  <th>字段值</th>
                  <th>字段注释</th>
                </tr>
              </thead>
              <tbody>
                {columns.map((col, idx) => (
                  <tr key={idx}>
                    <td className="tdp-col-name">
                      <div className="tdp-col-name-inner">
                        <button
                          type="button"
                          className={`tdp-col-name-btn ${activeFilterColumn === col.name ? 'tdp-col-name-btn-active' : ''}`}
                          onClick={() => handleFilterColumnClick(col.name)}
                          title="按这个字段过滤"
                        >
                          {col.name}
                        </button>
                        {col.primaryKey && <b>PK</b>}
                      </div>
                    </td>
                    <td className={`tdp-col-value ${col.isNull ? 'tdp-col-value-empty' : ''}`}>
                      {editing ? (
                        <textarea
                          className="tdp-value-input"
                          value={draftValues[col.name] ?? ''}
                          placeholder={col.isNull ? 'NULL' : ''}
                          spellCheck={false}
                          onChange={e => {
                            const value = e.target.value;
                            setDraftValues(prev => ({ ...prev, [col.name]: value }));
                          }}
                        />
                      ) : (
                        renderValue(col)
                      )}
                    </td>
                    <td className={`tdp-col-desc ${!col.description ? 'tdp-col-desc-empty' : ''}`}>
                      {col.description || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
      {confirmOpen && (
        <div
          className="tdp-confirm-layer"
          onClick={e => {
            e.stopPropagation();
            if (!saving) setConfirmOpen(false);
          }}
        >
          <div className="tdp-confirm" onClick={e => e.stopPropagation()}>
            <div className="tdp-confirm-head">
              <span className="tdp-confirm-icon">
                <AlertTriangle size={20} />
              </span>
              <div>
                <h3>确认修改字段值</h3>
                <p>{databaseName} → {tableName}，第 {offset + 1} 条记录</p>
              </div>
            </div>
            <div className="tdp-confirm-body">
              <div className="tdp-confirm-summary">
                本次将修改 <strong>{pendingChanges.length}</strong> 个字段，提交后会直接写入远程数据库。
              </div>
              <div className="tdp-change-list">
                {pendingChanges.slice(0, 6).map(change => (
                  <div className="tdp-change-item" key={change.name}>
                    <div className="tdp-change-name">
                      <span>{change.name}</span>
                      {change.primaryKey && <b>PK</b>}
                    </div>
                    <div className="tdp-change-values">
                      <span>{change.before}</span>
                      <i>→</i>
                      <span>{change.after}</span>
                    </div>
                  </div>
                ))}
                {pendingChanges.length > 6 && (
                  <div className="tdp-change-more">还有 {pendingChanges.length - 6} 个字段未展开</div>
                )}
              </div>
            </div>
            <div className="tdp-confirm-actions">
              <button className="tdp-action-btn" disabled={saving} onClick={() => setConfirmOpen(false)}>
                取消
              </button>
              <button className="tdp-action-btn tdp-action-danger" disabled={saving} onClick={handleConfirmSave}>
                {saving ? <Loader2 size={16} className="tdp-spinner" /> : <Save size={16} />}
                确认保存
              </button>
            </div>
          </div>
        </div>
      )}
      {ddlOpen && (
        <div
          className="tdp-ddl-layer"
          onClick={e => {
            e.stopPropagation();
            if (!ddlLoading) setDdlOpen(false);
          }}
        >
          <div className="tdp-ddl" onClick={e => e.stopPropagation()}>
            <div className="tdp-ddl-head">
              <div>
                <h3>
                  <Code2 size={18} />
                  建表 SQL
                </h3>
                <p>{databaseName} → {tableName}</p>
              </div>
              <div className="tdp-ddl-actions">
                <button className="tdp-action-btn" disabled={!ddlSQL || ddlLoading} onClick={handleCopyDDL}>
                  <Copy size={16} />
                  复制
                </button>
                <button className="tdp-close" disabled={ddlLoading} onClick={() => setDdlOpen(false)} title="关闭">
                  <X size={18} />
                </button>
              </div>
            </div>
            <div className="tdp-ddl-body">
              {ddlLoading ? (
                <div className="tdp-ddl-empty">
                  <Loader2 size={22} className="tdp-spinner" />
                  <span>正在读取建表 SQL...</span>
                </div>
              ) : ddlError ? (
                <div className="tdp-ddl-empty tdp-ddl-error">{ddlError}</div>
              ) : (
                <pre className="tdp-ddl-code">{ddlSQL}</pre>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
