import service from '../utils/request';

const api = service;

// Paths API
export const getPathList = (projectId: number) => {
  return api.get(`/tbgenerateprojectpath/getTbGenerateProjectPathList?projectId=${projectId}`);
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
