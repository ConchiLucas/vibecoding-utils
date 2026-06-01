import service from '../utils/request';

const api = service;

export const getProjectPlaceHolderList = () => {
  return api.get('/tbgenerateprojectplaceholder/getTbGenerateProjectPlaceHolderList');
};

export const createProjectPlaceHolder = (data: any) => {
  return api.post('/tbgenerateprojectplaceholder/createTbGenerateProjectPlaceHolder', data);
};

export const updateProjectPlaceHolder = (data: any) => {
  return api.put('/tbgenerateprojectplaceholder/updateTbGenerateProjectPlaceHolder', data);
};

export const deleteProjectPlaceHolder = (data: any) => {
  return api.delete('/tbgenerateprojectplaceholder/deleteTbGenerateProjectPlaceHolder', { data });
};
