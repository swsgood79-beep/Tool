package main

import (
	"html/template"
	"strings"
)

// Install recovery only on the Pages/quick-tunnel worker page. The browser
// additionally checks its current origin, leaving local/LAN/admin flows alone.
func (a *App) workerReconnectMarkup(name string, value any) string {
	root, _ := value.(map[string]any)
	worker := name == "shipping-records.html" && root["ShippingAdmin"] != true || name == "work.html" || name == "worker-client-directory.html" || (name == "project-info.html" && root["AdminInfo"] != true) || (strings.HasPrefix(name, "community") && root["CommunityAdmin"] != true) || (strings.HasPrefix(name, "announcement") && root["NoticeWorker"] == true)
	if !worker || a.testMode {
		return ""
	}
	c := a.state.Integration.Connection
	if c.Provider != "pages" || cloudURL.FindString(c.PublicURL) != c.PublicURL || c.PublicURL == "" || !validPhotoID.MatchString(a.state.Board) {
		return ""
	}
	return `<div id="worker-reconnect-config" hidden data-origin="` + template.HTMLEscapeString(c.PublicURL) + `" data-board="` + template.HTMLEscapeString(a.state.Board) + `" data-release="` + releaseVersion + `"></div><link rel="stylesheet" href="/worker-reconnect-v1.4.42.css"><script defer src="/worker-reconnect-v1.4.61.js"></script>`
}
func workerServiceWorkerCSP() string {
	return "default-src 'none'; script-src 'self'; connect-src 'self' https://api.github.com https://raw.githubusercontent.com; base-uri 'none'"
}
func isWorkerRecoveryAsset(path string) bool {
	return strings.EqualFold(path, "/worker-reconnect-sw-v1.4.42.js") || strings.EqualFold(path, "/worker-reconnect-sw-v1.4.49.js")
}
