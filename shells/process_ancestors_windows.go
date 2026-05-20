//go:build windows

package shells

import (
	"os"
	"syscall"
	"unsafe"
)

func processAncestorNames() []string {
	processes := processSnapshot()
	current, ok := processes[uint32(os.Getpid())]
	if !ok {
		return nil
	}

	var names []string
	parentID := current.ParentProcessID
	for range 32 {
		parent, ok := processes[parentID]
		if !ok || parent.ProcessID == parent.ParentProcessID {
			break
		}
		names = append(names, syscall.UTF16ToString(parent.ExeFile[:]))
		parentID = parent.ParentProcessID
	}
	return names
}

func processSnapshot() map[uint32]syscall.ProcessEntry32 {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snapshot)

	processes := make(map[uint32]syscall.ProcessEntry32)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := syscall.Process32First(snapshot, &entry); err != nil {
		return nil
	}

	for {
		processes[entry.ProcessID] = entry
		if err := syscall.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return processes
}
