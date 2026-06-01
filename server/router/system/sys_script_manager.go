package system

import "github.com/gin-gonic/gin"

type ScriptManagerRouter struct{}

func (r *ScriptManagerRouter) InitScriptManagerRouter(Router *gin.RouterGroup) {
	scriptManagerRouter := Router.Group("script-manager")
	{
		scriptManagerRouter.GET("categories", scriptManagerApi.ListCategories)
		scriptManagerRouter.POST("categories", scriptManagerApi.CreateCategory)
		scriptManagerRouter.PUT("categories/:id", scriptManagerApi.UpdateCategory)
		scriptManagerRouter.DELETE("categories/:id", scriptManagerApi.DeleteCategory)

		scriptManagerRouter.GET("workflows", scriptManagerApi.ListWorkflows)
		scriptManagerRouter.POST("workflows", scriptManagerApi.CreateWorkflow)
		scriptManagerRouter.GET("workflows/:id", scriptManagerApi.GetWorkflow)
		scriptManagerRouter.PUT("workflows/:id", scriptManagerApi.UpdateWorkflow)
		scriptManagerRouter.DELETE("workflows/:id", scriptManagerApi.DeleteWorkflow)
		scriptManagerRouter.GET("workflows/:id/executeStream", scriptManagerApi.ExecuteWorkflowStream)

		scriptManagerRouter.POST("steps", scriptManagerApi.CreateStep)
		scriptManagerRouter.PUT("steps/:id", scriptManagerApi.UpdateStep)
		scriptManagerRouter.DELETE("steps/:id", scriptManagerApi.DeleteStep)
		scriptManagerRouter.GET("steps/:id/executeStream", scriptManagerApi.ExecuteStepStream)

		scriptManagerRouter.GET("resource-categories", scriptManagerApi.ListResourceCategories)
		scriptManagerRouter.POST("resource-categories", scriptManagerApi.CreateResourceCategory)
		scriptManagerRouter.PUT("resource-categories/:id", scriptManagerApi.UpdateResourceCategory)
		scriptManagerRouter.DELETE("resource-categories/:id", scriptManagerApi.DeleteResourceCategory)
		scriptManagerRouter.POST("resource-configs", scriptManagerApi.CreateResourceConfig)
		scriptManagerRouter.PUT("resource-configs/:id", scriptManagerApi.UpdateResourceConfig)
		scriptManagerRouter.DELETE("resource-configs/:id", scriptManagerApi.DeleteResourceConfig)

		scriptManagerRouter.GET("executions", scriptManagerApi.ListExecutions)
		scriptManagerRouter.GET("executions/:id", scriptManagerApi.GetExecutionLog)
	}
}
