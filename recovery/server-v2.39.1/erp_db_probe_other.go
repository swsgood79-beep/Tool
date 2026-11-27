//go:build !windows

package main

import "context"

func platformERPDBProbe(context.Context, string) ERPDBReport {
	return ERPDBReport{Status: "unsupported"}
}
