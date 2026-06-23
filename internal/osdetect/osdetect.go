// Package osdetect ตรวจจับว่าเครื่อง Linux ปลายทาง (ที่เชื่อมผ่าน SSH) เป็น
// RHEL-family (dnf/yum + firewalld) หรือ Debian-family (apt + ufw/iptables)
// เพื่อให้ steps อื่นเลือกคำสั่งที่ถูกต้อง ข้อมูลทั้งหมดอ่านจาก remote host
// ไม่ใช่จากเครื่องที่รันโปรแกรมนี้ (ซึ่งอาจเป็น Windows)
package osdetect

import (
	"fmt"
	"strings"

	"mariadb-installer/internal/runner"
)

type Family string

const (
	FamilyRHEL    Family = "rhel"
	FamilyDebian  Family = "debian"
	FamilyUnknown Family = "unknown"
)

type Info struct {
	Family          Family
	Name            string // เช่น "Rocky Linux", "Ubuntu"
	VersionID       string // เช่น "9", "22.04"
	Arch            string // จาก uname -m ของ remote host
	MariaDBRepoArch string // ใช้สร้าง baseurl ของ mariadb.org เช่น "rhel9-amd64", "ubuntu-amd64"
}

// Detect สั่ง remote host (ผ่าน r) ให้ cat /etc/os-release และ uname -m
// แล้วสรุปออกมาเป็น Info ในโหมด dry-run ที่ยังไม่เชื่อมต่อจริง จะคืนค่า "ไม่ทราบ" แทน
// (ผู้ใช้ต้องเชื่อมต่อจริงด้วย --apply หรือระบุ --assume-os เพื่อดู plan ที่แม่นยำ)
func Detect(r *runner.Runner) (*Info, error) {
	if r.DryRun {
		return detectDryRunPlaceholder(), nil
	}

	osReleaseRaw, err := r.Run("cat /etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("อ่าน /etc/os-release บน %s ไม่สำเร็จ: %w", r.HostLabel, err)
	}
	archRaw, err := r.Run("uname -m")
	if err != nil {
		return nil, fmt.Errorf("รัน uname -m บน %s ไม่สำเร็จ: %w", r.HostLabel, err)
	}

	osRelease := parseOSRelease(osReleaseRaw)

	idLike := strings.ToLower(osRelease["ID_LIKE"])
	id := strings.ToLower(osRelease["ID"])
	versionID := osRelease["VERSION_ID"]
	name := osRelease["NAME"]

	info := &Info{
		Name:      name,
		VersionID: versionID,
		Arch:      mapArch(strings.TrimSpace(archRaw)),
	}

	switch {
	case strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") ||
		id == "rhel" || id == "centos" || id == "rocky" || id == "almalinux" || id == "fedora":
		info.Family = FamilyRHEL
		info.MariaDBRepoArch = fmt.Sprintf("rhel%s-%s", majorVersion(versionID), info.Arch)
	case strings.Contains(idLike, "debian") || id == "debian" || id == "ubuntu":
		info.Family = FamilyDebian
		// mariadb.org ใช้ ubuntu/debian ตามชื่อ id จริง ไม่ใช่ "debianX"
		info.MariaDBRepoArch = fmt.Sprintf("%s-%s", id, info.Arch)
	default:
		info.Family = FamilyUnknown
	}

	return info, nil
}

// detectDryRunPlaceholder คืนค่า Info แบบ "ไม่ทราบแน่ชัด" สำหรับ dry-run ที่ไม่ได้เชื่อมต่อ SSH จริง
// เพื่อให้ pipeline ยังเดินต่อแสดง plan คำสั่งได้ (คำสั่งที่ขึ้นกับ OS family จะแสดงทั้งสองแบบ
// หรือระบุไว้ชัดว่าเป็นการสมมติ RHEL เป็นค่า default)
func detectDryRunPlaceholder() *Info {
	return &Info{
		Family:          FamilyRHEL,
		Name:            "(ไม่ทราบ - dry-run โดยไม่ได้เชื่อมต่อ SSH จริง สมมติเป็น RHEL-family)",
		VersionID:       "?",
		Arch:            "amd64",
		MariaDBRepoArch: "rhel9-amd64",
	}
}

func mapArch(uname string) string {
	switch uname {
	case "x86_64":
		return "amd64"
	case "aarch64", "arm64":
		return "aarch64"
	default:
		return uname
	}
}

// majorVersion ตัดเอาเฉพาะเลขเวอร์ชันหลัก เช่น "9.4" -> "9"
func majorVersion(v string) string {
	if i := strings.Index(v, "."); i != -1 {
		return v[:i]
	}
	return v
}

// parseOSRelease parse เนื้อหาของ /etc/os-release ที่ได้มาจาก remote host (เป็น string ธรรมดา)
func parseOSRelease(content string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.Trim(parts[1], `"`)
		result[key] = val
	}
	return result
}

// PackageManager คืนชื่อ binary ของ package manager หลักที่ใช้ติดตั้ง
func (i *Info) PackageManager() string {
	switch i.Family {
	case FamilyRHEL:
		return "dnf"
	case FamilyDebian:
		return "apt-get"
	default:
		return ""
	}
}

func (i *Info) String() string {
	return fmt.Sprintf("%s %s (%s, family=%s, arch=%s)", i.Name, i.VersionID, i.MariaDBRepoArch, i.Family, i.Arch)
}
