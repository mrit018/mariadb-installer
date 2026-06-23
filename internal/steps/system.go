package steps

import (
	"fmt"

	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
	"mariadb-installer/internal/tuning"
)

// galeraPorts คือพอร์ตที่ Galera cluster + xtrabackup SST ต้องเปิด
// (3306=mysql, 4567=galera replication, 4444=SST, 4568=IST, 3334/33128=ตามที่เห็นใน log ต้นฉบับ)
var galeraPorts = []int{33128, 3306, 4567, 4444, 3334}

// ConfigureFirewall เปิดพอร์ตที่จำเป็นสำหรับ MariaDB + Galera
// RHEL: ใช้ firewalld (ตามด้วย disable firewalld แล้วใช้ iptables -F ปล่อยผ่านหมด ตาม pattern ใน log)
// Debian: ใช้ ufw ถ้ามี ไม่งั้น fallback เป็น iptables ตรง ๆ
func ConfigureFirewall(r *runner.Runner, info *osdetect.Info) error {
	r.SetStep("ตั้งค่า Firewall")

	switch info.Family {
	case osdetect.FamilyRHEL:
		for _, port := range galeraPorts {
			r.RunIgnoreErr(fmt.Sprintf("firewall-cmd --zone=trusted --add-port=%d/tcp --permanent", port))
			r.RunIgnoreErr(fmt.Sprintf("firewall-cmd --zone=public --add-port=%d/tcp --permanent", port))
		}
		r.RunIgnoreErr("firewall-cmd --reload")
		// log ต้นฉบับสุดท้ายแล้ว stop+disable firewalld และเคลียร์ iptables เอง
		// (ทีมที่เขียน log นี้เลือกคุมด้วย iptables/security group ภายนอกแทน)
		r.RunIgnoreErr("iptables -F")
		r.RunIgnoreErr("iptables -t nat -F")
		r.RunIgnoreErr("systemctl stop firewalld")
		r.RunIgnoreErr("systemctl disable firewalld")
	case osdetect.FamilyDebian:
		if _, err := r.Run("command -v ufw"); err == nil {
			for _, port := range galeraPorts {
				r.RunIgnoreErr(fmt.Sprintf("ufw allow %d/tcp", port))
			}
		} else {
			for _, port := range galeraPorts {
				r.RunIgnoreErr(fmt.Sprintf("iptables -A INPUT -p tcp --dport %d -j ACCEPT", port))
			}
		}
	}
	return nil
}

// DisableSELinux ปิด SELinux แบบเดียวกับ log ต้นฉบับ (setenforce 0 + เขียน config permanent)
// บน Debian ส่วนใหญ่ไม่มี SELinux อยู่แล้วจึง no-op
func DisableSELinux(r *runner.Runner, info *osdetect.Info) error {
	if info.Family != osdetect.FamilyRHEL {
		return nil
	}
	r.SetStep("ปิด SELinux")
	r.RunIgnoreErr("setenforce 0")

	content := "SELINUX=disabled\nSELINUXTYPE=targeted\n"
	if err := r.WriteFile("/etc/sysconfig/selinux", content, 0644); err != nil {
		return err
	}
	fmt.Println("    หมายเหตุ: ต้อง reboot เครื่องเพื่อให้ SELINUX=disabled มีผลสมบูรณ์ " +
		"(เช่นเดียวกับ log ต้นฉบับที่มีคำสั่ง `sudo reboot` ก่อนติดตั้งจริง)")
	return nil
}

// TuneKernel เขียนค่า sysctl ที่คำนวณจาก RAM จริง (vm.nr_hugepages, kernel.shmmax, kernel.shmall ฯลฯ)
// และค่า fixed อื่น ๆ ตามที่เห็นใน log ต้นฉบับ แล้วรัน sysctl -p
func TuneKernel(r *runner.Runner, info *osdetect.Info, t *tuning.Values) error {
	r.SetStep("ปรับค่า Kernel (sysctl) ตาม RAM เครื่อง")

	r.RunIgnoreErr("echo never > /sys/kernel/mm/transparent_hugepage/enabled")

	sysctlPath := "/etc/sysctl.d/99-mariadb-galera.conf"
	content := fmt.Sprintf(`# สร้างโดย mariadb-installer ตาม RAM จริงของเครื่อง (%d MB)
fs.suid_dumpable = 0
fs.aio-max-nr = 1048576
fs.file-max = 6815744
kernel.shmmni = 4096
kernel.sem = 250 32000 100 128
net.ipv4.ip_local_port_range = 9000 65500
net.core.rmem_default = 262144
net.core.rmem_max = 4194304
net.core.wmem_default = 262144
net.core.wmem_max = 1048576
net.ipv4.tcp_fin_timeout = 5
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_keepalive_intvl = 30
vm.nr_hugepages = %d
kernel.shmmax = %d
kernel.shmall = %d
`, t.MemTotalKB/1024, t.NrHugepages, t.ShmMax, t.ShmAll)

	if err := r.WriteFile(sysctlPath, content, 0644); err != nil {
		return err
	}

	r.RunIgnoreErr("sysctl -p " + sysctlPath)
	r.RunIgnoreErr("sysctl -w net.ipv4.route.flush=1")

	fmt.Printf("    [คำนวณจาก RAM] %s\n", t.String())
	return nil
}

// EnsureRoot ตรวจว่า login เข้า remote host ด้วย user root จริงหรือไม่ (เฉพาะ apply mode)
// เช็คผ่าน `whoami` บน remote host เพราะตัวโปรแกรมนี้อาจรันอยู่บนเครื่องอื่น (เช่น Windows)
// ที่ไม่มีแนวคิด root/uid 0 แบบ Linux
func EnsureRoot(r *runner.Runner) error {
	if r.DryRun {
		return nil
	}
	whoami, err := r.Run("whoami")
	if err != nil {
		return fmt.Errorf("ตรวจสิทธิ์ผู้ใช้บน %s ไม่สำเร็จ: %w", r.HostLabel, err)
	}
	if whoami != "root" {
		return fmt.Errorf("ต้อง SSH login เป็น root บน %s (ตรวจพบว่า login เป็น %q) "+
			"เนื่องจากต้องแก้ไฟล์ระบบและติดตั้งแพ็กเกจ", r.HostLabel, whoami)
	}
	return nil
}
