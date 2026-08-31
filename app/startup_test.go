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
		`.app-shell[aria-hidden="true"]{visibility:hidden}`,
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
	if strings.Contains(html, `.app-shell{visibility:hidden}`) {
		t.Fatal("startup styles must not keep the application hidden after loading")
	}
	criticalStyle := strings.Index(html, `.loading-screen{position:fixed`)
	externalStylesheet := strings.Index(html, `<link rel="stylesheet" href="app.css">`)
	if criticalStyle < 0 || externalStylesheet < 0 || criticalStyle > externalStylesheet {
		t.Fatal("critical startup styles must be embedded before the external stylesheet")
	}
}

func TestNativeWindowIsNotRevealedBeforeApplicationIsReady(t *testing.T) {
	content, err := applicationAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	javascript := string(content)
	showApplication := strings.Index(javascript, `document.getElementById("app-shell").setAttribute("aria-hidden", "false")`)
	hideLoading := strings.Index(javascript, `document.getElementById("loading-screen").classList.add("is-hidden")`)
	notifyNativeWindow := strings.Index(javascript, `await notifyApplicationReady()`)
	if showApplication < 0 || hideLoading < 0 || notifyNativeWindow < 0 || showApplication > notifyNativeWindow || hideLoading > notifyNativeWindow {
		t.Fatal("native window must not be revealed before the application interface is ready")
	}
	stylesheet, err := applicationAssets.ReadFile("web/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stylesheet), `background: var(--surface); color: var(--ink);`) {
		t.Fatal("application stylesheet must restore its normal background and text colors")
	}
}
