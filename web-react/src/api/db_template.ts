import service from '../utils/request';

const api = service;

export const getDbTemplateTypes = (projectId: number) => {
  return api.get('/tbgeneratedbtemplatetype/list', { params: { projectId } });
};

export const createDbTemplateType = (data: any) => {
  return api.post('/tbgeneratedbtemplatetype/create', data);
};

export const updateDbTemplateType = (data: any) => {
  return api.put('/tbgeneratedbtemplatetype/update', data);
};

export const deleteDbTemplateType = (data: any) => {
  return api.delete('/tbgeneratedbtemplatetype/delete', { data });
};

export const getDbTemplateScripts = (projectId: number, typeId: number) => {
  return api.get('/tbgeneratedbtemplatescript/list', { params: { projectId, typeId } });
};

export const createDbTemplateScript = (data: any) => {
  return api.post('/tbgeneratedbtemplatescript/create', data);
};

export const updateDbTemplateScript = (data: any) => {
  return api.put('/tbgeneratedbtemplatescript/update', data);
};

export const deleteDbTemplateScript = (data: any) => {
  return api.delete('/tbgeneratedbtemplatescript/delete', { data });
};
