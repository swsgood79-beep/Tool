package main

import "strings"

// Only full worker pages receive the public board-scoped alarm receiver.
func workerAlarmPage(name string, root map[string]any) bool {
	return name == "work.html" || name == "worker-client-directory.html" ||
		(name == "shipping-records.html" && root["ShippingAdmin"] != true) ||
		(name == "project-info.html" && root["AdminInfo"] != true) ||
		(strings.HasPrefix(name, "community") && root["CommunityAdmin"] != true) ||
		(strings.HasPrefix(name, "announcement") && root["NoticeWorker"] == true)
}
