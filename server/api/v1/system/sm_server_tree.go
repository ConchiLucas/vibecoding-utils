package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TreeNode struct {
	ID            uint       `json:"id"`
	PID           *uint      `json:"pid"`
	ProjectName   string     `json:"projectName,omitempty"`
	ServerName    string     `json:"serverName,omitempty"`
	InterfaceName string     `json:"interfaceName"`
	Children      []TreeNode `json:"children"`
}

// BuildServerTree builds a two-level tree: Project -> Server.
// Service nodes expose the underlying tb_interface_server primary key for
// rename/delete actions from the left tree.
func (a *TbInterfaceServerApi) BuildServerTree(c *gin.Context) {

	// Parse optional projectName filter from request body
	var req struct {
		ProjectName string `json:"projectName"`
	}
	_ = c.ShouldBindJSON(&req)

	// Query imported service rows.
	var servers []system.TbInterfaceServer
	db := global.GVA_DB
	if req.ProjectName != "" {
		db = db.Where("project_name = ?", req.ProjectName)
	}
	if err := db.Find(&servers).Error; err != nil {
		global.GVA_LOG.Error("获取服务列表失败!", zap.Error(err))
		response.FailWithMessage("获取服务列表失败", c)
		return
	}

	// Build Project -> Server two-level tree from tb_interface_server
	type projectInfo struct {
		node    *TreeNode
		servers map[string]bool
	}
	projectMap := make(map[string]*projectInfo)
	var projectOrder []string

	var nextProjectID uint = 100000

	for _, srv := range servers {
		pName := srv.ProjectName
		if pName == "" {
			continue // skip servers without project
		}
		sName := srv.ServerName
		if sName == "" {
			continue // skip servers without name
		}

		pi, exists := projectMap[pName]
		if !exists {
			nextProjectID++
			pi = &projectInfo{
				node: &TreeNode{
					ID:            nextProjectID,
					PID:           nil,
					ProjectName:   pName,
					InterfaceName: pName,
					Children:      []TreeNode{},
				},
				servers: make(map[string]bool),
			}
			projectMap[pName] = pi
			projectOrder = append(projectOrder, pName)
		}

		if !pi.servers[sName] {
			pi.servers[sName] = true
			pid := pi.node.ID
			serverNode := TreeNode{
				ID:            srv.ID,
				PID:           &pid,
				ServerName:    sName,
				ProjectName:   pName,
				InterfaceName: sName,
				Children:      []TreeNode{},
			}
			pi.node.Children = append(pi.node.Children, serverNode)
		}
	}

	// Collect results in insertion order
	result := make([]TreeNode, 0, len(projectOrder))
	for _, pName := range projectOrder {
		result = append(result, *projectMap[pName].node)
	}

	response.OkWithDetailed(result, "获取成功", c)
}
