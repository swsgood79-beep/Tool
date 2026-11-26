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
	allowed := map[string]bool{"manuals.html": true, "android-admin-settings.html": true, "admin-accounts.html": true, "inspection-inspectors.html": true, "shipping-records.html": true, "community.html": true, "community-edit.html": true, "community-detail.html": true, "data-storage.html": true, "announcements.html": true, "announcement-detail.html": true, "announcement-edit.html": true, "admin.html": true, "work.html": true, "worker-client-directory.html": true, "inspection-queue.html": true, "inspection-review.html": true, "project-info.html": true, "form": true, "designers.html": true, "client-notes.html": true, "client-note-edit.html": true, "order-import.html": true, "unread.html": true, "edit.html": true, "project-delete.html": true, "report-settings.html": true, "tts-settings.html": true, "market-settings.html": true, "schedule.html": true, "settings.html": true, "backup.html": true, "phone.html": true, "mobile-access.html": true, "passkey-settings.html": true}
	if !allowed[name] {
		return s
	}
	role, workerURL, mobile := "admin", "", false
	initialCalendar := false
	activeCount, dueCount := 0, 0
	if root, ok := v.(map[string]any); ok {
		workerURL, _ = root["WorkerURL"].(string)
		initialCalendar = name == "work.html" && root["InitialDesignTab"] == "calendar"
		mobile, _ = root["MobileAdmin"].(bool)
		if projects, ok := root["Projects"].([]ProjectView); ok {
			date, _ := root["BusinessDate"].(string)
			for _, p := range projects {
				if p.Active {
					activeCount++
					if date != "" && p.DueOn(date) {
						dueCount++
					}
				}
			}
		}
		if (strings.HasPrefix(name, "community") || name == "manuals.html") && root["CommunityAdmin"] != true {
			role = "worker"
			workerURL = "/work/" + root["Board"].(string)
		}
		if name == "shipping-records.html" && root["ShippingAdmin"] != true {
			role = "worker"
			workerURL = "/work/" + root["Board"].(string)
		}
		if root["NoticeWorker"] == true {
			role = "worker"
			workerURL = "/work/" + root["Board"].(string)
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
	assets := `<link rel="stylesheet" href="/community-v1.4.38.css"><link rel="stylesheet" href="/mobile-design-v1.4.24.css"><link rel="stylesheet" href="/market-calendar-v1.4.26.css"><link rel="stylesheet" href="/calendar-complete-v1.4.27.css"><link rel="stylesheet" href="/order-card-v1.4.41.css"><link rel="stylesheet" href="/calendar-flat-v1.4.32.css"><link rel="stylesheet" href="/talk-board-v1.4.40.css"><link rel="stylesheet" href="/talk-tabs-v1.4.99.css"><script defer src="/mobile-design-worker-v1.4.105.js"></script><script defer src="/order-card-v1.4.105.js"></script><script defer src="/talk-unread-v1.4.61.js"></script>`
	if name == "manuals.html" {
		assets = strings.Replace(assets, "/mobile-design-worker-v1.4.105.js", "/mobile-design-v1.4.105.js", 1)
	}
	assets += `<script defer src="/schedule-retention-v1.4.95.js"></script><script defer src="/work-tools-v1.4.97.js"></script>`
	if name == "admin.html" {
		assets += `<link rel="stylesheet" href="/admin-order-grid-v1.4.59.css">`
	}
	s = strings.Replace(s, "<html", `<html data-alpha-design="modern"`, 1)
	if initialCalendar {
		// Set the requested tab before the browser paints any worker content.
		// Deferred design scripts still open the existing calendar normally.
		s = strings.Replace(s, "<body", `<body data-design-tab="calendar"`, 1)
		assets += `<style id="calendar-initial-view">html[data-alpha-design="modern"] body[data-design-tab="calendar"] main>:not(#shipping-calendar){display:none!important}</style>`
	}
	s = strings.Replace(s, "</head>", assets+"</head>", 1)
	return strings.Replace(s, "</body>", metadata+"</body>", 1)
}
