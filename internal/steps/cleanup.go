// Package steps รวม step การติดตั้งทั้งหมด เรียกตามลำดับจาก main.go
// แต่ละ step รับ *runner.Runner และ *osdetect.Info เพื่อปรับคำสั่งให้ตรงกับ OS
package steps

import (
	"fmt"

	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
)

// CleanupOld หยุดและลบ MySQL/MariaDB/Percona/Galera เดิมทั้งหมด รวมถึง data directory
// เทียบเท่าบล็อก killall + rpm/apt --erase + rm -fr /var/lib/mysql ใน log ต้นฉบับ
func CleanupOld(r *runner.Runner, info *osdetect.Info) error {
	r.SetStep("ล้างค่าการติดตั้ง MySQL/MariaDB เดิม")

	// ฆ่า process ที่อาจค้างอยู่ (ignore error เพราะอาจไม่มี process นั้นรันอยู่เลย)
	procs := []string{
		"mysqld", "mysqld_safe", "rsync", "nc", "socat",
		"xtrabackup", "wsrep_sst_xtrabackup", "wsrep_sst_xtrabackup-v2", "xbstream",
	}
	for _, p := range procs {
		r.RunIgnoreErr(fmt.Sprintf("killall -9 %s", p))
	}

	// คำเตือนก่อนลบข้อมูลจริง (เฉพาะ apply mode เท่านั้นที่จะถามจริง)
	if !r.Confirm("จะลบ /var/lib/mysql และไฟล์ติดตั้งเดิมทั้งหมด ดำเนินการต่อหรือไม่?") {
		return fmt.Errorf("ผู้ใช้ยกเลิกการลบข้อมูลเดิม")
	}

	r.RunIgnoreErr("rm -f /var/lock/subsys/mysql")
	r.RunIgnoreErr("rm -fr /var/lib/mysql/.sst")

	switch info.Family {
	case osdetect.FamilyRHEL:
		r.RunIgnoreErr("rm -f /etc/yum.repos.d/mariadb.repo")
		// ลบ rpm package ทุกตัวที่เกี่ยวกับ mysql/mariadb/percona/galera ยกเว้น perl-* และ mysql-libs
		patterns := []string{"Percona", "percona", "MySQL", "mysql", "galera", "Galera", "mariadb", "MariaDB", "mariadb-libs"}
		for _, pat := range patterns {
			cmd := fmt.Sprintf(
				`rpm -q -a | grep %s | grep -v '^perl' | grep -v '^mysql-libs' | xargs --no-run-if-empty rpm --erase --nodeps`,
				pat,
			)
			r.RunIgnoreErr(cmd)
		}
		r.RunIgnoreErr("rpm -q -a | grep Percona-Server | xargs --no-run-if-empty rpm --erase --nodeps")
	case osdetect.FamilyDebian:
		r.RunIgnoreErr("rm -f /etc/apt/sources.list.d/mariadb.list")
		packages := []string{"mariadb-server", "mariadb-client", "mariadb-backup", "mariadb-common",
			"galera-4", "galera-3", "percona-server-server", "percona-xtrabackup*", "mysql-server", "mysql-client"}
		for _, p := range packages {
			r.RunIgnoreErr(fmt.Sprintf("apt-get -y purge %s", p))
		}
		r.RunIgnoreErr("apt-get -y autoremove")
	}

	// ลบ user mysql เก่า แล้วสร้างใหม่ (ให้ uid/gid สะอาด เหมือนใน log)
	r.RunIgnoreErr("userdel mysql")
	r.RunIgnoreErr("useradd -r -M -s /sbin/nologin mysql")

	r.RunIgnoreErr("rm -f /etc/init.d/mysql*")
	r.RunIgnoreErr("rm -f /var/log/mysqld.log")
	r.RunIgnoreErr("rm -fr /var/lib/mysql")

	return nil
}
