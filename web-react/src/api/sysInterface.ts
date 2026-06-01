import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbInterface {
  ID: number;
  interfaceName: string;
  paths: string;
  description: string;
  method: string;
  requestParam?: string;
  responseParam?: string;
  userName: string;
  serverName: string;
  projectName: string;
  requestType: string;
  lastTestedAt?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbInterface = (data: Partial<TbInterface>) => {
  return service.post<any, ApiResponse>('/interface/createTbInterface', data);
};

export const deleteTbInterface = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/interface/deleteTbInterface', { data });
};

export const updateTbInterface = (data: Partial<TbInterface>) => {
  return service.put<any, ApiResponse>('/interface/updateTbInterface', data);
};

export const getTbInterfaceList = (params: { page: number; pageSize: number; interfaceName?: string; serverName?: string; projectName?: string; paths?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbInterface>>>('/interface/getTbInterfaceList', { params });
};

export const forwardInterfaceApi = (data: {
  id?: number;
  paramsId?: number;
  environment?: string;
  requestParam?: string;
  clientId?: number;
  requestHeader?: string;
}) => {
  return service.post<any, ApiResponse>('/interface/forwardInterface', data);
};

