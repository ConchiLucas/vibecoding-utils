package system

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

func normalizeComposeSharedNetwork(content string) (string, bool, error) {
	document, root, err := parseComposeDocument(content)
	if err != nil {
		return "", false, err
	}
	if err := validateComposeSharedNetworkNode(root); err == nil {
		return content, false, nil
	}

	services := mappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return "", false, fmt.Errorf("Compose services 必须是映射")
	}
	for index := 1; index < len(services.Content); index += 2 {
		service := services.Content[index]
		if service.Kind != yaml.MappingNode {
			return "", false, fmt.Errorf("Compose service %q 必须是映射", services.Content[index-1].Value)
		}
		removeMappingKey(service, "networks")
	}

	setMappingValue(root, "networks", sharedDefaultNetworkNode())
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return "", false, fmt.Errorf("编码 Compose YAML 失败: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", false, fmt.Errorf("完成 Compose YAML 编码失败: %w", err)
	}
	normalized := output.String()
	if err := validateComposeSharedNetwork(normalized); err != nil {
		return "", false, fmt.Errorf("规范化后的 Compose 网络无效: %w", err)
	}
	return normalized, normalized != content, nil
}

func validateComposeSharedNetwork(content string) error {
	_, root, err := parseComposeDocument(content)
	if err != nil {
		return err
	}
	return validateComposeSharedNetworkNode(root)
}

func parseComposeDocument(content string) (*yaml.Node, *yaml.Node, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil, nil, fmt.Errorf("解析 Compose YAML 失败: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("Compose YAML 顶层必须是映射")
	}
	return &document, document.Content[0], nil
}

func validateComposeSharedNetworkNode(root *yaml.Node) error {
	services := mappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return fmt.Errorf("Compose services 必须是映射")
	}
	for index := 1; index < len(services.Content); index += 2 {
		serviceName := services.Content[index-1].Value
		service := services.Content[index]
		if service.Kind != yaml.MappingNode {
			return fmt.Errorf("Compose service %q 必须是映射", serviceName)
		}
		if mappingValue(service, "networks") != nil {
			return fmt.Errorf("Compose service %q 仍声明了独立 networks", serviceName)
		}
	}

	networks := mappingValue(root, "networks")
	if networks == nil || networks.Kind != yaml.MappingNode || len(networks.Content) != 2 || networks.Content[0].Value != "default" {
		return fmt.Errorf("Compose 顶层 networks 必须仅声明 default")
	}
	shared := networks.Content[1]
	if shared.Kind != yaml.MappingNode || len(shared.Content) != 4 {
		return fmt.Errorf("Compose default 网络声明不完整")
	}
	name := mappingValue(shared, "name")
	external := mappingValue(shared, "external")
	if name == nil || name.Value != SharedDockerNetworkName {
		return fmt.Errorf("Compose default 网络 name 必须是 %s", SharedDockerNetworkName)
	}
	if external == nil || external.Tag != "!!bool" || external.Value != "true" {
		return fmt.Errorf("Compose default 网络必须声明 external: true")
	}
	return nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func removeMappingKey(mapping *yaml.Node, key string) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content = append(mapping.Content[:index], mapping.Content[index+2:]...)
			return
		}
	}
}

func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func sharedDefaultNetworkNode() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "default"},
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: SharedDockerNetworkName},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "external"},
					{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
				},
			},
		},
	}
}

func isComposeFileName(fileName string) bool {
	switch strings.ToLower(filepath.Base(strings.TrimSpace(fileName))) {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		return true
	default:
		return false
	}
}

func reconcileStoredComposeScripts(db *gorm.DB) (int, error) {
	var changed int
	err := db.Transaction(func(tx *gorm.DB) error {
		var scripts []modelSystem.TbProjectScript
		if err := tx.Where("script_type <> ?", 2).Order("id asc").Find(&scripts).Error; err != nil {
			return fmt.Errorf("读取本地 Compose 脚本失败: %w", err)
		}
		for _, script := range scripts {
			if !isComposeFileName(script.FileName) || strings.TrimSpace(script.Content) == "" {
				continue
			}
			normalized, contentChanged, err := normalizeComposeSharedNetwork(script.Content)
			if err != nil {
				return fmt.Errorf("规范化 Compose 脚本失败(project=%d script=%d file=%s): %w", script.ProjectId, script.ID, script.FileName, err)
			}
			if !contentChanged {
				continue
			}
			if err := tx.Model(&modelSystem.TbProjectScript{}).Where("id = ?", script.ID).Update("content", normalized).Error; err != nil {
				return fmt.Errorf("更新 Compose 脚本失败(project=%d script=%d): %w", script.ProjectId, script.ID, err)
			}
			changed++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return changed, nil
}

func ReconcileStoredComposeScriptsOnStartup() {
	go func() {
		if global.GVA_DB == nil {
			return
		}
		changed, err := reconcileStoredComposeScripts(global.GVA_DB)
		if err != nil {
			zap.L().Warn("启动时规范化 Compose 共享网络失败", zap.Error(err))
			return
		}
		zap.L().Info("启动时 Compose 共享网络检查完成", zap.Int("updated_scripts", changed))
	}()
}
