package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

const (
	RemoteDBConnectionTimeoutSeconds = 20
	RemoteDBConnectionTimeout        = RemoteDBConnectionTimeoutSeconds * time.Second
)

func BuildOracleDSN(host string, port int, serviceName, userName, password string) string {
	descriptor := fmt.Sprintf(
		"(DESCRIPTION=(CONNECT_TIMEOUT=%d)(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%d))(CONNECT_DATA=(SERVICE_NAME=%s)(SERVER=DEDICATED)))",
		RemoteDBConnectionTimeoutSeconds,
		strings.TrimSpace(host),
		port,
		strings.TrimSpace(serviceName),
	)
	return go_ora.BuildJDBC(userName, password, descriptor, map[string]string{
		"CONNECTION TIMEOUT": strconv.Itoa(RemoteDBConnectionTimeoutSeconds),
	})
}
