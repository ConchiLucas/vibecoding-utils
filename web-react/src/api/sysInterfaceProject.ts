import service from '../utils/request';
import { ApiResponse } from './sysDict';

export interface TbProject {
  ID: number;
  projectName: string;
  projectDesc?: string;
  userName: string;
  CreatedAt?: string;
}

export const createTbProject = (data: Partial<TbProject>) => {
  return service.post<any, ApiResponse>('/project/createTbInterfaceProject', data);
};

export const deleteTbProject = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/project/deleteTbInterfaceProject', { data });
};

export const updateTbProject = (data: Partial<TbProject>) => {
  return service.put<any, ApiResponse>('/project/updateTbInterfaceProject', data);
};

export const getTbProjectList = () => {
  return service.get<any, ApiResponse<TbProject[]>>('/project/getTbInterfaceProjectList');
};
