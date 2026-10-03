package viewhelpers

import (
	"github.com/gofiber/template/html/v3"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAllTemplatesLoad(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	engine := html.New(filepath.Join(filepath.Dir(source), "..", "views"), ".html")
	Register(engine)
	if err := engine.Load(); err != nil {
		t.Fatal(err)
	}
}
