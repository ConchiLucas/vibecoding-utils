package system

import (
	"encoding/json"
	"fmt"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

// SwaggerJSON represents the top-level structure of a Swagger/OpenAPI 3 JSON file
type SwaggerJSON struct {
	Paths      map[string]map[string]SwaggerOperation `json:"paths"`
	Components *SwaggerComponents                     `json:"components"`
}

type SwaggerOperation struct {
	Summary     string                     `json:"summary"`
	Description string                     `json:"description"`
	RequestBody *SwaggerRequestBody        `json:"requestBody"`
	Responses   map[string]SwaggerResponse `json:"responses"`
}

type SwaggerRequestBody struct {
	Content map[string]SwaggerMediaType `json:"content"`
}

type SwaggerMediaType struct {
	Schema SwaggerSchema `json:"schema"`
}

type SwaggerSchema struct {
	Ref        string                 `json:"$ref"`
	Type       string                 `json:"type"`
	Properties map[string]SwaggerProp `json:"properties"`
	Required   []string               `json:"required"`
	Items      *SwaggerSchema         `json:"items"`
}

type SwaggerProp struct {
	Ref         string         `json:"$ref"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Default     interface{}    `json:"default"`
	Format      string         `json:"format"`
	MaxLength   int            `json:"maxLength"`
	MinLength   int            `json:"minLength"`
	Enum        []interface{}  `json:"enum"`
	Items       *SwaggerSchema `json:"items"`
}

type SwaggerResponse struct {
	Content map[string]SwaggerMediaType `json:"content"`
}

type SwaggerComponents struct {
	Schemas map[string]SwaggerSchema `json:"schemas"`
}

// ImportSwaggerInterfaces parses a swagger JSON file and imports server, interface, entity, column data.
func (s *TbInterfaceServerService) ImportSwaggerInterfaces(fileContent []byte, projectName, serverName, userName string) error {
	var swagger SwaggerJSON
	if err := json.Unmarshal(fileContent, &swagger); err != nil {
		return fmt.Errorf("解析Swagger JSON失败: %w", err)
	}

	db := global.GVA_DB

	// 1. Delete old data for this server+user
	db.Where("server_name = ? AND user_name = ?", serverName, userName).Unscoped().Delete(&system.TbInterfaceServer{})
	db.Where("server_name = ? AND user_name = ?", serverName, userName).Unscoped().Delete(&system.TbColumn{})
	db.Where("server_name = ? AND user_name = ?", serverName, userName).Unscoped().Delete(&system.TbEntity{})
	db.Where("server_name = ? AND user_name = ?", serverName, userName).Unscoped().Delete(&system.TbInterface{})

	// 2. Create the server record
	newServer := system.TbInterfaceServer{
		ProjectName: projectName,
		ServerName:  serverName,
		UserName:    userName,
	}
	if err := db.Create(&newServer).Error; err != nil {
		return fmt.Errorf("创建服务记录失败: %w", err)
	}

	// 3. Parse interfaces from paths
	var interfaceList []system.TbInterface
	for path, methods := range swagger.Paths {
		for requestType, op := range methods {
			interfaceName := op.Summary
			description := op.Description
			if description == "" {
				description = op.Summary
			}

			// Extract request param ref
			requestParam := ""
			if op.RequestBody != nil {
				for _, mediaType := range op.RequestBody.Content {
					if mediaType.Schema.Ref != "" {
						requestParam = extractSchemaName(mediaType.Schema.Ref)
					}
				}
			}

			// Extract response param ref
			responseParam := ""
			if resp, ok := op.Responses["200"]; ok {
				for contentType, mediaType := range resp.Content {
					if contentType == "application/json" && mediaType.Schema.Ref != "" {
						responseParam = extractSchemaName(mediaType.Schema.Ref)
					}
				}
			}

			// Derive method from the last segment of path
			method := path
			if idx := lastIndex(path, '/'); idx >= 0 && idx < len(path)-1 {
				method = path[idx+1:]
			}

			iface := system.TbInterface{
				InterfaceName: interfaceName,
				Paths:         path,
				Description:   description,
				Method:        method,
				RequestParam:  requestParam,
				ResponseParam: responseParam,
				UserName:      userName,
				ServerName:    serverName,
				ProjectName:   projectName,
				RequestType:   requestType,
			}
			interfaceList = append(interfaceList, iface)
		}
	}

	if len(interfaceList) > 0 {
		if err := db.CreateInBatches(&interfaceList, 100).Error; err != nil {
			return fmt.Errorf("批量导入接口失败: %w", err)
		}
	}

	// 4. Parse entities from components.schemas
	if swagger.Components != nil && swagger.Components.Schemas != nil {
		var entityList []system.TbEntity
		var columnList []system.TbColumn

		for entityName, schema := range swagger.Components.Schemas {
			requiredColumn := ""
			if len(schema.Required) > 0 {
				for i, r := range schema.Required {
					if i > 0 {
						requiredColumn += ", "
					}
					requiredColumn += r
				}
			}

			columnCount := 0
			containEntity := 0
			if schema.Properties != nil {
				for propName, prop := range schema.Properties {
					columnCount++
					if prop.Ref != "" {
						containEntity = 1
					}

					colType := prop.Type
					if colType == "" {
						colType = "object"
					}

					defaultValue := ""
					if prop.Default != nil {
						defaultValue = fmt.Sprintf("%v", prop.Default)
					}

					enumValue := ""
					if len(prop.Enum) > 0 {
						for i, e := range prop.Enum {
							if i > 0 {
								enumValue += ","
							}
							enumValue += fmt.Sprintf("%v", e)
						}
					}

					columnRef := ""
					if prop.Ref != "" {
						columnRef = extractSchemaName(prop.Ref)
					} else if prop.Items != nil && prop.Items.Ref != "" {
						columnRef = extractSchemaName(prop.Items.Ref)
					}

					required := 0
					for _, r := range schema.Required {
						if r == propName {
							required = 1
							break
						}
					}

					col := system.TbColumn{
						EntityName:   entityName,
						ColumnName:   propName,
						ColumnType:   colType,
						Description:  prop.Description,
						DefaultValue: defaultValue,
						FormatValue:  prop.Format,
						MaxLength:    prop.MaxLength,
						MinLength:    prop.MinLength,
						Required:     required,
						EnumValue:    enumValue,
						ColumnRef:    columnRef,
						UserName:     userName,
						ServerName:   serverName,
					}
					columnList = append(columnList, col)
				}
			}

			entity := system.TbEntity{
				EntityName:     entityName,
				RequiredColumn: requiredColumn,
				ColumnCount:    columnCount,
				ContainEntity:  containEntity,
				UserName:       userName,
				ServerName:     serverName,
			}
			entityList = append(entityList, entity)
		}

		if len(entityList) > 0 {
			if err := db.CreateInBatches(&entityList, 100).Error; err != nil {
				return fmt.Errorf("批量导入实体失败: %w", err)
			}
		}
		if len(columnList) > 0 {
			if err := db.CreateInBatches(&columnList, 100).Error; err != nil {
				return fmt.Errorf("批量导入字段失败: %w", err)
			}
		}
	}

	return nil
}

func extractSchemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if len(ref) > len(prefix) {
		return ref[len(prefix):]
	}
	return ref
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}
