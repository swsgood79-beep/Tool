//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

type erpJobBasic struct {
	ProcessTime, JobTime   int64
	Flags                  uint32
	MinWorking, MaxWorking uintptr
	ActiveLimit            uint32
	Affinity               uintptr
	Priority, Scheduling   uint32
}
type erpJobExtended struct {
	Basic                                                      erpJobBasic
	IO                                                         [6]uint64
	ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
}

func newERPJob() (syscall.Handle, error) {
	h, _, e := kernel32.NewProc("CreateJobObjectW").Call(0, 0)
	if h == 0 {
		return 0, fmt.Errorf("ERP 수집 프로세스 관리 준비 실패: %v", e)
	}
	info := erpJobExtended{}
	info.Basic.Flags = 0x2000 // KILL_ON_JOB_CLOSE, owned children only.
	ok, _, e := kernel32.NewProc("SetInformationJobObject").Call(h, 9, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(h))
		return 0, fmt.Errorf("ERP 수집 종료 관리 설정 실패: %v", e)
	}
	return syscall.Handle(h), nil
}
func attachERPJob(job syscall.Handle, pid int) error {
	p, e := syscall.OpenProcess(0x0101, false, uint32(pid))
	if e != nil {
		return e
	}
	defer syscall.CloseHandle(p)
	ok, _, e := kernel32.NewProc("AssignProcessToJobObject").Call(uintptr(job), uintptr(p))
	if ok == 0 {
		return fmt.Errorf("ERP 수집 프로세스 연결 관리 실패: %v", e)
	}
	return nil
}
