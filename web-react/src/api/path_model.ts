import service from '../utils/request';

const api = service;

// Paths API
export const getPathList = (projectId: number) => {
  return api.get('/tbgenerateprojectpath/getTbGenerateProjectPathList', {
    params: { projectInstanceId: projectId },
  });
};

export const createPath = (data: any) => {
  return api.post('/tbgenerateprojectpath/createTbGenerateProjectPath', data);
};

export const updatePath = (data: any) => {
  return api.put('/tbgenerateprojectpath/updateTbGenerateProjectPath', data);
};

export const copyPathSet = (data: any) => {
  return api.post('/tbgenerateprojectpath/copyPathSet', data);
};

export const deletePathSet = (data: any) => {
  return api.post('/tbgenerateprojectpath/deletePathSet', data);
};

export const renamePathSet = (data: any) => {
  return api.post('/tbgenerateprojectpath/renamePathSet', data);
};

export const buildPromptSummary = (data: any) => {
  return api.post('/tbgenerateprojectpath/buildPromptSummary', data);
};

export const deletePath = (data: any) => {
  return api.delete('/tbgenerateprojectpath/deleteTbGenerateProjectPath', { data });
};

export const updatePathEnabled = (data: any) => {
  return api.post('/tbgenerateprojectpath/updateEnabled', data);
};

// Models API
export const getModelListByPathId = (pathId: number) => {
  return api.get(`/tbgenerateprojectpathmodel/getTbGenerateProjectPathModelList?pathId=${pathId}`);
};

export const createModel = (data: any) => {
  return api.post('/tbgenerateprojectpathmodel/createTbGenerateProjectPathModel', data);
};

export const updateModel = (data: any) => {
  return api.put('/tbgenerateprojectpathmodel/updateTbGenerateProjectPathModel', data);
};
