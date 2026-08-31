package main

import (
	"strings"
	"testing"
)

func TestStartupScreenHasCriticalStylesWithoutExternalImage(t *testing.T) {
	content, err := applicationAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(content)
	required := []string{
		`html,body{width:100%;height:100%;margin:0;background:#17231f`,
		`.loading-screen{position:fixed;inset:0;z-index:50;display:grid`,
		`.loading-mark{display:grid;place-items:center;width:54px;height:54px`,
		`<div class="loading-mark" aria-hidden="true">Z</div>`,
	}
	for _, fragment := range required {
		if !strings.Contains(html, fragment) {
			t.Fatalf("startup HTML is missing critical fragment %q", fragment)
		}
	}
	if strings.Contains(html, `<img class="loading-mark"`) {
		t.Fatal("startup logo must not wait for an external image")
	}
	criticalStyle := strings.Index(html, `.loading-screen{position:fixed`)
	externalStylesheet := strings.Index(html, `<link rel="stylesheet" href="app.css">`)
	if criticalStyle < 0 || externalStylesheet < 0 || criticalStyle > externalStylesheet {
		t.Fatal("critical startup styles must be embedded before the external stylesheet")
	}
}
