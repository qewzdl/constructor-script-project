package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetNavigationStateActivePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &TemplateHandler{}

	cases := []struct {
		name     string
		request  string
		expected string
	}{
		{name: "Trailing slash", request: "/page/about/", expected: "/page/about"},
		{name: "Double slash", request: "//page/about//", expected: "/page/about"},
		{name: "Root", request: "/", expected: "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, tc.request, nil)
			ctx.Request = req

			data := gin.H{}
			handler.setNavigationState(ctx, data)

			if got := data["ActivePath"]; got != tc.expected {
				t.Fatalf("expected ActivePath %s, got %v", tc.expected, got)
			}
		})
	}
}
