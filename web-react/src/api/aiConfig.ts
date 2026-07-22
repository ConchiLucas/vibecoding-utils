import service from '../utils/request';

export interface AIProviderConfigItem {
  id: string;
  label: string;
  type: string;
  base_url: string;
  api_key: string;
  model: string;
  max_tokens: number;
  active?: boolean;
}

export interface AIConfigResponse {
  active: string;
  providers: AIProviderConfigItem[];
}

interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

export const getAIConfig = () => {
  return service.get<any, ApiResponse<AIConfigResponse>>('/ai/config');
};

export const saveAIConfig = (data: AIConfigResponse) => {
  return service.post<any, ApiResponse<AIProviderConfigItem[]>>('/ai/config', data);
};

export const saveAIActiveProvider = (active: string) => {
  return service.post<any, ApiResponse<AIProviderConfigItem[]>>('/ai/config/active', { active });
};
