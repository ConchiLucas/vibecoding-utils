import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbInterfaceEnv {
  ID: number;
  projectName: string;
  envName: string;
  baseUrl: string;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbInterfaceEnv = (data: Partial<TbInterfaceEnv>) => {
  return service.post<any, ApiResponse>('/interfaceEnv/createTbInterfaceEnv', data);
};

export const deleteTbInterfaceEnv = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/interfaceEnv/deleteTbInterfaceEnv', { data });
};

export const updateTbInterfaceEnv = (data: Partial<TbInterfaceEnv>) => {
  return service.put<any, ApiResponse>('/interfaceEnv/updateTbInterfaceEnv', data);
};

export const getTbInterfaceEnvList = (params: { page: number; pageSize: number; projectName?: string; envName?: string; userName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbInterfaceEnv>>>('/interfaceEnv/getTbInterfaceEnvList', { params });
};
