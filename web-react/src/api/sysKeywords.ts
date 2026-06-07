import request from '@/utils/request'

export const fuzzyQuery = (data: any) => {
  return request({
    url: '/table/fuzzyQuery',
    method: 'post',
    data
  })
}

export const getClientData = (data: any) => {
  return request({
    url: '/tableRelate/getClientData',
    method: 'post',
    data
  })
}

export const getPreferVOByParams = (data: any) => {
  return request({
    url: '/tablePrefer/getPreferVOByParams',
    method: 'post',
    data
  })
}

export const getPreferColumnValueList = (data: any) => {
  return request({
    url: '/tablePrefer/getPreferColumnValueList',
    method: 'post',
    data
  })
}

export const getHistoryTableNames = (params?: { projectConfigId?: number | null; connectionId?: number | null }) => {
  return request({
    url: '/tablePrefer/getHistoryTableNames',
    method: 'get',
    params
  })
}
