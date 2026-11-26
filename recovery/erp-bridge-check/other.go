//go:build !windows

package main

import "errors"

func prepareConsole() {}
func collectWindows(string) ([]Window, error) {
	return nil, errors.New("이 도구는 ERP가 실행되는 Windows PC 전용입니다")
}
func showResult(message, folder string) {}
func exportTest() error                 { return errors.New("엑셀 저장 테스트는 Windows PC 전용입니다") }

func backgroundTest() error {
	return errors.New("가려진 ERP 창 시험은 Windows PC 전용입니다")
}
