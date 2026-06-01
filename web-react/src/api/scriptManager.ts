import service from '../utils/request';

export type ScriptStepType = 'local_exec' | 'local_upload' | 'target_download' | 'target_exec';
export type ScriptResourceCategoryType = 'fixed' | 'dynamic' | 'constant';

export interface ScriptCategory {
  ID: number;
  categoryName: string;
  description?: string;
}

export interface ScriptStep {
  ID: number;
  workflowId: number;
  stepName: string;
  stepType: ScriptStepType;
  scriptContent: string;
  placeholders?: string;
}

export interface ScriptWorkflow {
  ID: number;
  categoryId: number;
  workflowName: string;
  description?: string;
  lastStatus?: string;
  lastRunAt?: string;
  steps?: ScriptStep[];
}

export interface ScriptExecution {
  ID: number;
  workflowId: number;
  stepId: number;
  scope: 'workflow' | 'step';
  status: 'running' | 'success' | 'failed';
  logText: string;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  durationMs?: number;
}

export interface ScriptResourceRow {
  name: string;
  placeholder: string;
  value: string;
}

export interface ScriptResourceConfig {
  ID: number;
  categoryId: number;
  workflowId?: number;
  configName: string;
  placeholderKey?: string;
  rows: string;
}

export interface ScriptResourceCategory {
  ID: number;
  categoryName: string;
  categoryType: ScriptResourceCategoryType;
  configs?: ScriptResourceConfig[];
}

export const getScriptCategories = () => {
  return service.get<any, any>('/script-manager/categories');
};

export const createScriptCategory = (data: Partial<ScriptCategory>) => {
  return service.post<any, any>('/script-manager/categories', data);
};

export const updateScriptCategory = (id: number, data: Partial<ScriptCategory>) => {
  return service.put<any, any>(`/script-manager/categories/${id}`, data);
};

export const deleteScriptCategory = (id: number) => {
  return service.delete<any, any>(`/script-manager/categories/${id}`);
};

export const getScriptWorkflows = (params: { page?: number; pageSize?: number; categoryId?: number; keyword?: string }) => {
  return service.get<any, any>('/script-manager/workflows', { params });
};

export const createScriptWorkflow = (data: Partial<ScriptWorkflow>) => {
  return service.post<any, any>('/script-manager/workflows', data);
};

export const updateScriptWorkflow = (id: number, data: Partial<ScriptWorkflow>) => {
  return service.put<any, any>(`/script-manager/workflows/${id}`, data);
};

export const deleteScriptWorkflow = (id: number) => {
  return service.delete<any, any>(`/script-manager/workflows/${id}`);
};

export const createScriptStep = (data: Partial<ScriptStep>) => {
  return service.post<any, any>('/script-manager/steps', data);
};

export const updateScriptStep = (id: number, data: Partial<ScriptStep>) => {
  return service.put<any, any>(`/script-manager/steps/${id}`, data);
};

export const deleteScriptStep = (id: number) => {
  return service.delete<any, any>(`/script-manager/steps/${id}`);
};

export const getScriptResourceCategories = () => {
  return service.get<any, any>('/script-manager/resource-categories');
};

export const createScriptResourceCategory = (data: Partial<ScriptResourceCategory>) => {
  return service.post<any, any>('/script-manager/resource-categories', data);
};

export const updateScriptResourceCategory = (id: number, data: Partial<ScriptResourceCategory>) => {
  return service.put<any, any>(`/script-manager/resource-categories/${id}`, data);
};

export const deleteScriptResourceCategory = (id: number) => {
  return service.delete<any, any>(`/script-manager/resource-categories/${id}`);
};

export const createScriptResourceConfig = (data: Partial<ScriptResourceConfig>) => {
  return service.post<any, any>('/script-manager/resource-configs', data);
};

export const updateScriptResourceConfig = (id: number, data: Partial<ScriptResourceConfig>) => {
  return service.put<any, any>(`/script-manager/resource-configs/${id}`, data);
};

export const deleteScriptResourceConfig = (id: number) => {
  return service.delete<any, any>(`/script-manager/resource-configs/${id}`);
};

export const getScriptExecutions = (params: { page?: number; pageSize?: number; workflowId?: number; stepId?: number; scope?: 'workflow' | 'step' }) => {
  return service.get<any, any>('/script-manager/executions', { params });
};

export const getScriptExecution = (id: number) => {
  return service.get<any, any>(`/script-manager/executions/${id}`);
};
