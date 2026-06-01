import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbInterfaceParams {
  ID: number;
  interfacePaths: string;
  userName: string;
  environment: string;
  identity: string;
  interfaceParams: string;
  responseParams: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbInterfaceParams = (data: Partial<TbInterfaceParams>) => {
  return service.post<any, ApiResponse>('/interfaceParams/createTbInterfaceParams', data);
};

export const deleteTbInterfaceParams = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/interfaceParams/deleteTbInterfaceParams', { data });
};

export const updateTbInterfaceParams = (data: Partial<TbInterfaceParams>) => {
  return service.put<any, ApiResponse>('/interfaceParams/updateTbInterfaceParams', data);
};

export const getTbInterfaceParamsList = (params: { page: number; pageSize: number; interfacePaths?: string; environment?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbInterfaceParams>>>('/interfaceParams/getTbInterfaceParamsList', { params });
};

export interface ParamsEntityResult {
  id: number;
  paramsId: number;
  environment: string;
  identity: string;
  interfaceParams: string;
  responseParams: string;
}

export const getParamsEntity = (data: { id: number; paths: string }) => {
  return service.post<any, ApiResponse<ParamsEntityResult>>('/interfaceParams/getParamsEntity', data);
};

