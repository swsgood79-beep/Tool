//go:build !windows

package main

import "fmt"

func platformReceiptHelper(task receiptHelperTask) error {
	return fmt.Errorf("Windows desktop required")
}
