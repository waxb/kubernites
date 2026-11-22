package app

import (
	"context"
	"net/http"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func TestACardPreviewThatPanicsAnswersAnErrorAndNeverEndsTheProcess(t *testing.T) {
	ans, fields := previewSafely(context.Background(), nil, write.Order{Request: write.Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"}})
	if ans.OK() || ans.Err == nil || fields != nil {
		t.Fatalf("a preview that panicked answered %+v %v", ans, fields)
	}
}
