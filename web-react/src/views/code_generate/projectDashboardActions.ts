export const PROJECT_TYPE_BACKEND = 'backend';
export const PROJECT_TYPE_FRONTEND = 'frontend';

export const normalizeProjectType = (value: any) => {
  return String(value || '').trim().toLowerCase() === PROJECT_TYPE_FRONTEND
    ? PROJECT_TYPE_FRONTEND
    : PROJECT_TYPE_BACKEND;
};

export const isBackendProject = (project: any) => normalizeProjectType(project?.projectType) === PROJECT_TYPE_BACKEND;

export const shouldShowDbTemplateActions = (project: any) => isBackendProject(project);

export const getProjectTypeLabel = (value: any) => {
  return normalizeProjectType(value) === PROJECT_TYPE_FRONTEND ? '前端' : '后端';
};

export const matchesProjectCardSearch = (project: any, keyword: string) => {
  const normalizedKeyword = String(keyword || '').trim().toLowerCase();
  if (!normalizedKeyword) return true;

  return (
    String(project?.projectName || '').toLowerCase().includes(normalizedKeyword) ||
    String(project?.remark || '').toLowerCase().includes(normalizedKeyword) ||
    getProjectTypeLabel(project?.projectType).toLowerCase().includes(normalizedKeyword)
  );
};
