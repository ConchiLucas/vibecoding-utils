import service from '../utils/request'

// Create an alias for compatibility with existing code in this file
const api = service

export const getProjectList = (params?: { projectConfigId?: number | null }) => {
    return api.get('/tbgenerateproject/getTbGenerateProjectList', { params })
}

export const createProject = (data: any) => {
    return api.post('/tbgenerateproject/createTbGenerateProject', data)
}

export const updateProject = (data: any) => {
    return api.put('/tbgenerateproject/updateTbGenerateProject', data)
}

export const deleteProject = (data: any) => {
    return api.delete(`/tbgenerateproject/deleteTbGenerateProject`, { data })
}

export const copyProject = (id: string) => {
    return api.get(`/tbgenerateproject/copy?id=${id}`)
}
