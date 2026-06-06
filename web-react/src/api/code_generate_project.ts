import service from '../utils/request'

// Create an alias for compatibility with existing code in this file
const api = service

export const getProjectList = (params?: { projectConfigId?: number | null }) => {
    return api.get('/tbgenerateproject/getTbGenerateProjectList', { params })
}

export const getProject = (id: number | string) => {
    return api.get('/tbgenerateproject/getTbGenerateProject', { params: { id } })
}

export const createProject = (data: any) => {
    return api.post('/tbgenerateproject/createTbGenerateProject', data)
}

export const updateProject = (data: any) => {
    return api.put('/tbgenerateproject/updateTbGenerateProject', data)
}

export const updateProjectSelectedInstance = (data: any) => {
    return api.put('/tbgenerateproject/updateSelectedProjectInstance', data)
}

export const deleteProject = (data: any) => {
    return api.delete(`/tbgenerateproject/deleteTbGenerateProject`, { data })
}

export const copyProject = (id: string) => {
    return api.get(`/tbgenerateproject/copy?id=${id}`)
}

export const getProjectInstanceList = (templateProjectId: number, ensureDefault = true) => {
    return api.get('/tbgenerateprojectinstance/getTbGenerateProjectInstanceList', {
        params: {
            templateProjectId,
            ensureDefault: ensureDefault ? 1 : 0,
        },
    })
}

export const getProjectInstance = (id: number | string) => {
    return api.get('/tbgenerateprojectinstance/getTbGenerateProjectInstance', { params: { id } })
}

export const createProjectInstance = (data: any) => {
    return api.post('/tbgenerateprojectinstance/createTbGenerateProjectInstance', data)
}

export const updateProjectInstance = (data: any) => {
    return api.put('/tbgenerateprojectinstance/updateTbGenerateProjectInstance', data)
}

export const updateProjectInstanceSelectedPathSet = (data: any) => {
    return api.put('/tbgenerateprojectinstance/updateSelectedPathSet', data)
}

export const deleteProjectInstance = (data: any) => {
    return api.delete('/tbgenerateprojectinstance/deleteTbGenerateProjectInstance', { data })
}
