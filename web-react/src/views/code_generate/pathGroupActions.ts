export type PathGroupDeleteState = {
  canDelete: boolean;
  reason: string;
};

const normalizeRelativeDir = (value: string) => {
  return String(value || '')
    .replace(/\\/g, '/')
    .replace(/\/+/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+$/, '')
    .trim();
};

export const getPathGroupDeleteState = (group: { paths?: any[] } | null | undefined): PathGroupDeleteState => {
  const pathCount = Array.isArray(group?.paths) ? group.paths.length : 0;
  if (pathCount > 0) {
    return {
      canDelete: false,
      reason: '该子目录下还有路径数据，不能删除',
    };
  }

  return {
    canDelete: true,
    reason: '',
  };
};

export const getPathGroupSwitchOptions = (groups: any[]) => {
  return (Array.isArray(groups) ? groups : [])
    .filter((group) => String(group?.key || '').trim() && String(group?.basePath || '').trim())
    .map((group) => ({
      key: String(group.key),
      label: normalizeRelativeDir(group.basePath || '') || '/',
    }));
};
