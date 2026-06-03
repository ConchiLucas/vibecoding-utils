package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbConnectionRouter struct{}

func (s *TbConnectionRouter) InitTbConnectionRouter(Router *gin.RouterGroup) {
	connectionRouter := Router.Group("connection")
	connectionRouterWithoutRecord := Router.Group("connection")
	var connectionApi = v1.ApiGroupApp.SystemApiGroup.TbConnectionApi
	{
		connectionRouter.POST("createTbConnection", connectionApi.CreateTbConnection)
		connectionRouter.DELETE("deleteTbConnection", connectionApi.DeleteTbConnection)
		connectionRouter.DELETE("deleteTbConnectionByIds", connectionApi.DeleteTbConnectionByIds)
		connectionRouter.PUT("updateTbConnection", connectionApi.UpdateTbConnection)
		connectionRouter.POST("updateRemoteTableRecord", connectionApi.UpdateRemoteTableRecord)
		connectionRouter.POST("generateRemoteTableData", connectionApi.GenerateRemoteTableData)
		connectionRouter.POST("saveRemoteSQLHistory", connectionApi.SaveRemoteSQLHistory)
		connectionRouter.DELETE("deleteRemoteSQLHistory", connectionApi.DeleteRemoteSQLHistory)
		connectionRouter.DELETE("clearRemoteSQLHistory", connectionApi.ClearRemoteSQLHistory)
	}
	{
		connectionRouterWithoutRecord.GET("getTbConnection", connectionApi.GetTbConnection)
		connectionRouterWithoutRecord.GET("getTbConnectionList", connectionApi.GetTbConnectionList)
		connectionRouterWithoutRecord.GET("testConnection", connectionApi.TestConnection)
		connectionRouterWithoutRecord.POST("testConnectionPayload", connectionApi.TestConnectionPayload)
		connectionRouterWithoutRecord.GET("initConnection", connectionApi.InitConnection)
		connectionRouterWithoutRecord.GET("getRemoteDatabases", connectionApi.GetRemoteDatabases)
		connectionRouterWithoutRecord.GET("getRemoteTables", connectionApi.GetRemoteTables)
		connectionRouterWithoutRecord.GET("getRemoteTablePreview", connectionApi.GetRemoteTablePreview)
		connectionRouterWithoutRecord.GET("getRemoteTablePage", connectionApi.GetRemoteTablePage)
		connectionRouterWithoutRecord.GET("getRemoteTableDDL", connectionApi.GetRemoteTableDDL)
		connectionRouterWithoutRecord.GET("getRemoteTableComments", connectionApi.GetRemoteTableComments)
		connectionRouterWithoutRecord.POST("queryRemoteSQL", connectionApi.QueryRemoteSQL)
		connectionRouterWithoutRecord.GET("getRemoteSQLHistory", connectionApi.GetRemoteSQLHistory)
	}
}
