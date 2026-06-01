import service from '../utils/request';

// 分页获取项目列表
export const getProjectPage = (data: any) => {
  return service({
    url: '/project/page',
    method: 'post',
    data: data,
  });
};

// 获取项目列表（不分页）
export const getProjectList = (data?: any) => {
  return service({
    url: '/project/list',
    method: 'post',
    data: data || {},
  });
};

// 根据ID获取项目
export const getProjectById = (id: number | string) => {
  return service({
    url: '/project/getById/' + id,
    method: 'get',
  });
};

// 获取下一次部署建议端口
export const getNextDeployPort = (type: 'frontend' | 'backend') => {
  return service({
    url: '/project/nextPort',
    method: 'get',
    params: { type },
  });
};

// 新增或修改项目
export const saveOrUpdateProject = (data: any) => {
  return service({
    url: '/project/saveOrUpdate',
    method: 'post',
    data: data,
  });
};

// 删除项目
export const deleteProject = (ids: any) => {
  return service({
    url: '/project/delete/' + ids,
    method: 'delete',
  });
};

// 执行部署
export const processDeploy = (id: number | string, env?: string) => {
  const query = env ? `?env=${env}` : '';
  return service({
    url: '/project/processDeploy/' + id + query,
    method: 'post',
  });
};
