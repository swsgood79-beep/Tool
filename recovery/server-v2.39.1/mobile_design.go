package main

import (
	"html/template"
	"strconv"
	"strings"
)

// Modern design is the default; the optional legacy preference stays browser-local. Never decorate card/poll fragments or
// login/handoff pages, and never export a board token on an unauthenticated page.
func mobileDesignPreview(s, name string, v any) string {
	if !strings.Contains(s, "</body>") {
		return s
	}
	allowed := map[string]bool{"admin.html": true, "work.html": true, "worker-client-directory.html": true, "inspection-queue.html": true, "inspection-review.html": true, "project-info.html": true, "form": true, "designers.html": true, "client-notes.html": true, "client-note-edit.html": true, "order-import.html": true, "unread.html": true, "edit.html": true, "project-delete.html": true, "report-settings.html": true, "tts-settings.html": true, "market-settings.html": true, "schedule.html": true, "settings.html": true, "backup.html": true, "phone.html": true, "mobile-access.html": true, "passkey-settings.html": true}
	if !allowed[name] {
		return s
	}
	role, workerURL, mobile := "admin", "", false
	activeCount, dueCount := 0, 0
	if root, ok := v.(map[string]any); ok {
		workerURL, _ = root["WorkerURL"].(string)
		mobile, _ = root["MobileAdmin"].(bool)
		if projects, ok := root["Projects"].([]ProjectView); ok {
			date, _ := root["BusinessDate"].(string)
			for _, p := range projects {
				if p.Active {
					activeCount++
					if date != "" && p.ShippingDueDate == date {
						dueCount++
					}
				}
			}
		}
		if name == "work.html" {
			role = "worker"
			if board, ok := root["CSRF"].(string); ok {
				workerURL = "/work/" + board
			}
		}
		if name == "worker-client-directory.html" {
			role = "worker"
			if board, ok := root["Board"].(string); ok {
				workerURL = "/work/" + board
			}
		}
		if name == "project-info.html" && root["AdminInfo"] != true {
			role = "worker"
			if board, ok := root["CSRF"].(string); ok {
				workerURL = "/work/" + board
			}
		}
	}
	metadata := `<div id="alpha-design" hidden data-page="` + template.HTMLEscapeString(name) + `" data-role="` + role + `" data-worker-url="` + template.HTMLEscapeString(workerURL) + `" data-mobile="` + map[bool]string{true: "true", false: "false"}[mobile] + `"></div>`
	if role == "admin" {
		metadata = strings.Replace(metadata, ` data-mobile=`, ` data-active-count="`+strconv.Itoa(activeCount)+`" data-due-count="`+strconv.Itoa(dueCount)+`" data-mobile=`, 1)
	}
	assets := `<link rel="stylesheet" href="/mobile-design-v1.4.24.css"><link rel="stylesheet" href="/market-calendar-v1.4.26.css"><script defer src="/mobile-design-v1.4.24.js"></script>`
	s = strings.Replace(s, "<html", `<html data-alpha-design="modern"`, 1)
	s = strings.Replace(s, "</head>", assets+"</head>", 1)
	return strings.Replace(s, "</body>", metadata+"</body>", 1)
}
