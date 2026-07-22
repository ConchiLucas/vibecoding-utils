package system

import (
	"reflect"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIConfigRouterRegistersOnlyConfigRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&AIConfigRouter{}).InitAIConfigRouter(engine.Group(""))

	got := make([]string, 0, len(engine.Routes()))
	for _, route := range engine.Routes() {
		got = append(got, route.Method+" "+route.Path)
	}
	sort.Strings(got)
	want := []string{
		"GET /ai/config",
		"POST /ai/config",
		"POST /ai/config/active",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registered routes = %#v, want %#v", got, want)
	}
}
