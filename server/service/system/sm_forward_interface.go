package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

// ForwardRequest is the input from the frontend
type ForwardRequest struct {
	ID            uint   `json:"id"`            // tb_interface.id
	ParamsID      uint   `json:"paramsId"`      // existing tb_interface_params.id for upsert
	Environment   string `json:"environment"`   // env name
	RequestParam  string `json:"requestParam"`  // JSON string for the request body
	ClientID      uint   `json:"clientId"`      // tb_interface_server_user.id
	RequestHeader string `json:"requestHeader"` // override header (sent from frontend)
	UserName      string `json:"-"`             // filled by handler from JWT
}

// ForwardInterface mirrors Python forward_interface:
//  1. Resolve interface → paths, request_type
//  2. Resolve environment → env base URL
//  3. Use requestHeader from selected user for auth
//  4. Send HTTP request (multi-method aware)
//  5. Save result to tb_interface_params (upsert) and tb_interface_log
func (s *TbInterfaceService) ForwardInterface(req ForwardRequest) (map[string]interface{}, error) {
	// ---- 1. Get interface info ----
	var iface system.TbInterface
	if err := global.GVA_DB.Where("id = ?", req.ID).First(&iface).Error; err != nil {
		return nil, fmt.Errorf("接口不存在: %w", err)
	}

	// ---- 1.1 Update last_tested_at timestamp ----
	now := time.Now()
	global.GVA_DB.Model(&iface).Update("last_tested_at", now)

	// ---- 2. Get environment to find base URL ----
	var envServer system.TbInterfaceEnv
	if err := global.GVA_DB.
		Where("env_name = ?", req.Environment).
		First(&envServer).Error; err != nil {
		return nil, fmt.Errorf("找不到环境配置 [%s]: %w", req.Environment, err)
	}

	// Build full URL: env base URL + interface path
	envPrefix := strings.TrimSpace(envServer.BaseURL)
	envPrefix = strings.TrimRight(envPrefix, "/")
	paths := strings.TrimSpace(iface.Paths)
	paths = strings.TrimLeft(paths, "/")

	// Clean double slashes in paths only, to avoid breaking http://
	paths = strings.ReplaceAll(paths, "///", "/")
	paths = strings.ReplaceAll(paths, "//", "/")

	requestURL := envPrefix + "/" + paths

	// ---- 4. Parse request body ----
	var bodyBytes []byte
	if strings.TrimSpace(req.RequestParam) != "" {
		var jsonBody interface{}
		if err := json.Unmarshal([]byte(req.RequestParam), &jsonBody); err == nil {
			bodyBytes, _ = json.Marshal(jsonBody)
		} else {
			bodyBytes = []byte(req.RequestParam)
		}
	}

	// ---- 5. Parse request header ----
	headerMap := map[string]string{}
	if req.RequestHeader != "" {
		// Try JSON map first
		if err := json.Unmarshal([]byte(req.RequestHeader), &headerMap); err != nil {
			// Might be a plain token string — put it as Authorization
			headerMap["Authorization"] = req.RequestHeader
		}
	}

	// ---- 6. Map request type to HTTP method ----
	method := strings.ToUpper(iface.RequestType)
	if method == "" {
		method = "POST"
	}

	// ---- 7. Execute HTTP request ----
	var respBody []byte
	var statusCode int
	var httpErr error

	client := &http.Client{Timeout: 30 * time.Second}

	// <-- 打印最终请求的 URL 帮助排查问题 -->
	fmt.Printf("[ForwardInterface] 即将发送请求, Method: %s, URL: %s\n", method, requestURL)

	httpReq, err := http.NewRequest(method, requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headerMap {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		httpErr = err
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		respBody, _ = io.ReadAll(resp.Body)
	}

	// ---- 8. Parse response ----
	var responseJSON map[string]interface{}
	isSuccess := 0
	responseStr := ""
	if httpErr != nil {
		responseJSON = map[string]interface{}{
			"success": false,
			"code":    -1,
			"msg":     httpErr.Error(),
			"data":    nil,
		}
		responseStr = fmt.Sprintf(`{"success":false,"code":-1,"msg":"%s"}`, httpErr.Error())
	} else {
		if err := json.Unmarshal(respBody, &responseJSON); err != nil {
			responseJSON = map[string]interface{}{
				"success":    statusCode >= 200 && statusCode < 300,
				"code":       statusCode,
				"rawContent": string(respBody),
			}
		}
		if v, ok := responseJSON["success"].(bool); ok && v {
			isSuccess = 1
		} else if statusCode >= 200 && statusCode < 300 {
			isSuccess = 1
		}
		b, _ := json.MarshalIndent(responseJSON, "", "  ")
		responseStr = string(b)
	}

	// ---- 9. Get user identity for log ----
	identity := ""
	if req.ClientID > 0 {
		var user system.TbInterfaceServerUser
		if global.GVA_DB.Where("id = ?", req.ClientID).First(&user).Error == nil {
			identity = user.RoleCode
			if identity == "" {
				identity = user.LoginAccount
			}
		}
	}

	// ---- 10. Upsert tb_interface_params (save most recent request for the path) ----
	paramsRecord := system.TbInterfaceParams{
		InterfacePaths:  iface.Paths,
		UserName:        req.UserName,
		Environment:     req.Environment,
		Identity:        identity,
		InterfaceParams: req.RequestParam,
		ResponseParams:  responseStr,
	}
	saveExisting := func(existing *system.TbInterfaceParams) {
		existing.Environment = req.Environment
		existing.Identity = identity
		existing.InterfaceParams = req.RequestParam
		existing.ResponseParams = responseStr
		global.GVA_DB.Save(existing)
	}
	if req.ParamsID > 0 {
		var existing system.TbInterfaceParams
		if global.GVA_DB.
			Where("id = ? AND interface_paths = ? AND user_name = ?", req.ParamsID, iface.Paths, req.UserName).
			First(&existing).Error == nil {
			saveExisting(&existing)
		} else {
			var currentPathRecord system.TbInterfaceParams
			if global.GVA_DB.Where("interface_paths = ? AND user_name = ?", iface.Paths, req.UserName).
				Order("id DESC").First(&currentPathRecord).Error == nil {
				saveExisting(&currentPathRecord)
			} else {
				global.GVA_DB.Create(&paramsRecord)
			}
		}
	} else {
		// See if there's an existing record we can update
		var existing system.TbInterfaceParams
		if global.GVA_DB.Where("interface_paths = ? AND user_name = ?", iface.Paths, req.UserName).
			Order("id DESC").First(&existing).Error == nil {
			saveExisting(&existing)
		} else {
			global.GVA_DB.Create(&paramsRecord)
		}
	}

	// ---- 11. Insert tb_interface_log ----
	logRecord := system.TbInterfaceLog{
		InterfacePaths: iface.Paths,
		UserName:       req.UserName,
		IsSuccess:      isSuccess,
		ReqParams:      req.RequestParam,
		ResParams:      responseStr,
		Environment:    req.Environment,
		Identity:       identity,
	}
	global.GVA_DB.Create(&logRecord)

	return responseJSON, nil
}
