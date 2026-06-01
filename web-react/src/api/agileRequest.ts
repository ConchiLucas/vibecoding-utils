import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export type AgileMethod = 'GET' | 'POST' | 'PUT' | 'DELETE';

export interface AgileRequestLog {
  ID: number;
  userName: string;
  method: AgileMethod;
  url: string;
  requestHeaders: string;
  requestBody: string;
  responseStatus: number;
  responseHeaders: string;
  responseBody: string;
  durationMs: number;
  isSuccess: number;
  errorMessage: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface AgileRequestPayload {
  method: AgileMethod;
  url: string;
  requestHeaders: string;
  requestBody: string;
}

export const sendAgileRequest = (data: AgileRequestPayload) => {
  return service.post<any, ApiResponse<AgileRequestLog>>('/agileRequest/send', data);
};

export const getAgileRequestHistory = (params: { page: number; pageSize: number; keyword?: string; method?: string }) => {
  return service.get<any, ApiResponse<PageResult<AgileRequestLog>>>('/agileRequest/history', { params });
};

export const getAgileRequestDetail = (params: { id: number }) => {
  return service.get<any, ApiResponse<AgileRequestLog>>('/agileRequest/detail', { params });
};

export const deleteAgileRequestHistory = (id: number) => {
  return service.delete<any, ApiResponse>('/agileRequest/history', { params: { id } });
};

export const clearAgileRequestHistory = () => {
  return service.delete<any, ApiResponse>('/agileRequest/history/clear');
};
