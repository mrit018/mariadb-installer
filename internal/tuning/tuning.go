// Package tuning คำนวณค่าพารามิเตอร์ tuning ของ MariaDB และ kernel
// จากแรมจริงของเครื่อง Linux ปลายทาง (อ่านผ่าน SSH ไม่ใช่เครื่องที่รันโปรแกรมนี้)
// โดยใช้สูตรเดียวกับที่เห็นในสคริปต์ต้นฉบับ:
//
//   - innodb_buffer_pool_size = ~75% ของ RAM ทั้งหมด (ปัดเป็น MiB)
//   - kernel.shmmax            = 75% ของ RAM ทั้งหมด (เป็นไบต์)
//   - kernel.shmall            = shmmax / page size (ปกติ 4096 ไบต์)
//   - vm.nr_hugepages          = (75% ของ RAM) / hugepage size
//   - key_buffer_size          = ~10% ของ RAM
//   - max_heap_table_size      = ~5% ของ RAM
//
// ตัวเลขเปอร์เซ็นต์เหล่านี้ตรงกับสัดส่วนที่คำนวณได้จาก log ต้นฉบับ
// (ตัวอย่าง: RAM ~125 GiB -> buffer pool 64146M, key_buffer 12829M, heap 6415M)
package tuning

import (
	"fmt"
	"strconv"
	"strings"

	"mariadb-installer/internal/runner"
)

const (
	bufferPoolPercent = 0.75
	shmmaxPercent     = 0.75
	keyBufferPercent  = 0.10
	heapTablePercent  = 0.05
	defaultPageSize   = 4096 // ไบต์ ใช้คำนวณ shmall เมื่อหา PAGESIZE จริงไม่ได้
	defaultHugepageKB = 2048 // KB (2MB) ค่าปกติของ hugepage บน x86_64 เมื่อหาจาก remote ไม่ได้
)

type Values struct {
	MemTotalKB     uint64 // จาก /proc/meminfo MemTotal (KB) ของ remote host
	MemTotalBytes  uint64
	HugepageSizeKB uint64 // จาก /proc/meminfo Hugepagesize (KB)
	PageSizeBytes  uint64 // จาก getconf PAGESIZE บน remote host

	InnodbBufferPoolMB uint64
	KeyBufferMB        uint64
	MaxHeapTableMB     uint64
	ShmMax             uint64 // ไบต์
	ShmAll             uint64 // หน่วยเป็นจำนวน page
	NrHugepages        uint64
}

// Detect สั่ง remote host ให้รายงาน MemTotal, Hugepagesize, page size แล้วคำนวณค่า tuning ทั้งหมด
// ในโหมด dry-run ที่ยังไม่เชื่อมต่อ SSH จริง จะคืนค่าตัวอย่างสมมติ (placeholder) เพื่อให้ดู plan ได้
func Detect(r *runner.Runner) (*Values, error) {
	if r.DryRun {
		return detectDryRunPlaceholder(), nil
	}

	memTotalKB, err := readMemTotalKB(r)
	if err != nil {
		return nil, fmt.Errorf("อ่าน MemTotal จาก %s ไม่สำเร็จ: %w", r.HostLabel, err)
	}
	hugepageSizeKB, err := readHugepageSizeKB(r)
	if err != nil {
		// ไม่ fatal ถ้าหา hugepage size ไม่ได้ ใช้ค่า default
		hugepageSizeKB = defaultHugepageKB
	}
	pageSize, err := readPageSize(r)
	if err != nil {
		pageSize = defaultPageSize
	}

	memTotalBytes := memTotalKB * 1024

	v := &Values{
		MemTotalKB:     memTotalKB,
		MemTotalBytes:  memTotalBytes,
		HugepageSizeKB: hugepageSizeKB,
		PageSizeBytes:  pageSize,
	}

	v.InnodbBufferPoolMB = uint64(float64(memTotalKB) * bufferPoolPercent / 1024)
	v.KeyBufferMB = uint64(float64(memTotalKB) * keyBufferPercent / 1024)
	v.MaxHeapTableMB = uint64(float64(memTotalKB) * heapTablePercent / 1024)

	v.ShmMax = uint64(float64(memTotalBytes) * shmmaxPercent)
	v.ShmAll = v.ShmMax / pageSize

	hugepageBytes := hugepageSizeKB * 1024
	v.NrHugepages = uint64(float64(memTotalBytes) * shmmaxPercent / float64(hugepageBytes))

	return v, nil
}

// detectDryRunPlaceholder คืนค่าตัวอย่างสมมติ (RAM 16GB) สำหรับ dry-run ที่ยังไม่เชื่อมต่อ SSH จริง
// ตัวเลขจริงจะคำนวณใหม่ตอน --apply เมื่อเชื่อมต่อ remote host ได้แล้ว
func detectDryRunPlaceholder() *Values {
	var placeholderMemKB uint64 = 16 * 1024 * 1024 // 16 GiB (ตัวแปร runtime ไม่ใช่ const
	// เพื่อให้การคูณ/หารด้วยเปอร์เซ็นต์ที่ลงตัวไม่พอดีไม่ถูกตรวจเป็น constant overflow ตอน build)
	v := &Values{
		MemTotalKB:     placeholderMemKB,
		MemTotalBytes:  placeholderMemKB * 1024,
		HugepageSizeKB: defaultHugepageKB,
		PageSizeBytes:  defaultPageSize,
	}
	v.InnodbBufferPoolMB = uint64(float64(placeholderMemKB) * bufferPoolPercent / 1024)
	v.KeyBufferMB = uint64(float64(placeholderMemKB) * keyBufferPercent / 1024)
	v.MaxHeapTableMB = uint64(float64(placeholderMemKB) * heapTablePercent / 1024)
	v.ShmMax = uint64(float64(v.MemTotalBytes) * shmmaxPercent)
	v.ShmAll = v.ShmMax / defaultPageSize
	v.NrHugepages = uint64(float64(v.MemTotalBytes) * shmmaxPercent / float64(defaultHugepageKB*1024))
	return v
}

func readMemTotalKB(r *runner.Runner) (uint64, error) {
	out, err := r.Run(`grep '^MemTotal:' /proc/meminfo | awk '{print $2}'`)
	if err != nil {
		return 0, err
	}
	return parseUintField(out, "MemTotal")
}

func readHugepageSizeKB(r *runner.Runner) (uint64, error) {
	out, err := r.Run(`grep '^Hugepagesize:' /proc/meminfo | awk '{print $2}'`)
	if err != nil {
		return 0, err
	}
	return parseUintField(out, "Hugepagesize")
}

func readPageSize(r *runner.Runner) (uint64, error) {
	out, err := r.Run("getconf PAGESIZE")
	if err != nil {
		return 0, err
	}
	return parseUintField(out, "PAGESIZE")
}

func parseUintField(raw, fieldName string) (uint64, error) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return 0, fmt.Errorf("ไม่พบค่า %s", fieldName)
	}
	n, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("แปลงค่า %s (%q) เป็นตัวเลขไม่สำเร็จ: %w", fieldName, val, err)
	}
	return n, nil
}

func (v *Values) String() string {
	return fmt.Sprintf(
		"RAM=%dMB innodb_buffer_pool=%dM key_buffer=%dM max_heap_table=%dM shmmax=%d shmall=%d nr_hugepages=%d",
		v.MemTotalKB/1024, v.InnodbBufferPoolMB, v.KeyBufferMB, v.MaxHeapTableMB, v.ShmMax, v.ShmAll, v.NrHugepages,
	)
}
