package tuning

import "syscall"

// osGetpagesize คืนค่า page size ของระบบเป็นไบต์ ผ่าน syscall.Getpagesize()
// ซึ่งบน Linux เทียบเท่ากับคำสั่ง `getconf PAGESIZE`
func osGetpagesize() int {
	return syscall.Getpagesize()
}
