package i18n

import (
	"syscall"
	"unsafe"
)

// systemLanguage reads the Windows display language, such as "ru-RU", which a
// terminal on Windows does not pass on through LANG.
func systemLanguage() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLang := kernel32.NewProc("GetUserDefaultUILanguage")
	toName := kernel32.NewProc("LCIDToLocaleName")
	if getLang.Find() != nil || toName.Find() != nil {
		return ""
	}
	id, _, _ := getLang.Call()
	var buf [85]uint16 // LOCALE_NAME_MAX_LENGTH
	n, _, _ := toName.Call(id, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
