import service from '../utils/request';

// 获取项目组列表
export const getGroupList = (data?: any) => {
  return service({
    url: '/projectGroup/list',
    method: 'post',
    data: data || {},
  });
};

// 新增或修改项目组
export const saveOrUpdateGroup = (data: any) => {
  return service({
    url: '/projectGroup/saveOrUpdate',
    method: 'post',
    data: data,
  });
};

// 更新项目组随 VibeDeploy 启动的联动开关
export const updateGroupAutoStart = (groupId: number, enabled: boolean) => {
  return service({
    url: '/projectGroup/autoStart',
    method: 'post',
    data: { groupId, enabled },
  });
};

// 删除项目组
export const deleteGroup = (id: number) => {
  return service({
    url: '/projectGroup/delete/' + id,
    method: 'delete',
  });
};
