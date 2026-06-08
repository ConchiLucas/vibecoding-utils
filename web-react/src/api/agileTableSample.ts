import service from '../utils/request';
import { ApiResponse } from './sysDict';

export interface AgileTableSampleRecord {
  ID: number;
  projectConfigId?: number;
  connectionId: number;
  databaseName: string;
  tableName: string;
  businessName?: string;
  tableComment?: string;
  sortIndex: number;
  userName?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface AgileTableSampleItem {
  databaseName: string;
  tableName: string;
  tableComment?: string;
}

export interface AgileTableSampleHistory {
  ID: number;
  projectConfigId?: number;
  connectionId: number;
  userName?: string;
  historyName?: string;
  tableCount: number;
  tables: AgileTableSampleItem[];
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const getAgileTableSamples = (params: { projectConfigId?: number; connectionId: number }) => {
  return service.get<any, ApiResponse<AgileTableSampleRecord[]>>('/agileTableSample/list', { params });
};

export const saveAgileTableSamples = (data: {
  projectConfigId?: number;
  connectionId: number;
  historyName?: string;
  tables: AgileTableSampleItem[];
}) => {
  return service.post<any, ApiResponse<AgileTableSampleRecord[]>>('/agileTableSample/save', data);
};

export const getAgileTableSampleHistory = (params: { projectConfigId?: number; connectionId: number }) => {
  return service.get<any, ApiResponse<AgileTableSampleHistory[]>>('/agileTableSample/history', { params });
};
