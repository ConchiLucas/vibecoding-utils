import service from '../utils/request';

const api = service;

export const getPlaceholderList = () => {
  return api.get('/tbgenerateplaceholder/getTbGeneratePlaceHolderList');
};

export const createPlaceholder = (data: any) => {
  return api.post('/tbgenerateplaceholder/createTbGeneratePlaceHolder', data);
};

export const updatePlaceholder = (data: any) => {
  return api.put('/tbgenerateplaceholder/updateTbGeneratePlaceHolder', data);
};

export const deletePlaceholder = (data: any) => {
  return api.delete('/tbgenerateplaceholder/deleteTbGeneratePlaceHolder', { data });
};

// Project Placeholders
export const getProjectPlaceholderList = () => {
    return api.get('/tbgenerateprojectplaceholder/getTbGenerateProjectPlaceHolderList');
};
  
export const createProjectPlaceholder = (data: any) => {
    return api.post('/tbgenerateprojectplaceholder/createTbGenerateProjectPlaceHolder', data);
};
  
export const updateProjectPlaceholder = (data: any) => {
    return api.put('/tbgenerateprojectplaceholder/updateTbGenerateProjectPlaceHolder', data);
};

export const deleteProjectPlaceholder = (data: any) => {
    return api.delete('/tbgenerateprojectplaceholder/deleteTbGenerateProjectPlaceHolder', { data });
};
