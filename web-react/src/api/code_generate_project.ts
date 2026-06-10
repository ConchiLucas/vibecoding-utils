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

export const copyProject = (data: any) => {
    return api.post('/tbgenerateproject/copy', data)
}

export const generateProjectCode = (data: any) => {
    return api.post('/tbgenerateproject/generateCode', data)
}

export const getGenerateFieldSnippetLatest = (businessType: string) => {
    return api.get('/tbgeneratefieldsnippet/latest', { params: { businessType } })
}

export const getGenerateFieldSnippetHistory = (businessType: string) => {
    return api.get('/tbgeneratefieldsnippet/history', { params: { businessType } })
}

export const previewGenerateFieldSnippet = (data: any) => {
    return api.post('/tbgeneratefieldsnippet/preview', data)
}

export const saveGenerateFieldSnippet = (data: any) => {
    return api.post('/tbgeneratefieldsnippet/save', data)
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
