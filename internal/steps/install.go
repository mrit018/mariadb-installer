package steps

import (
	"fmt"

	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
)

// InstallPackages ติดตั้ง MariaDB server/client/backup + galera ผ่าน package manager ของ OS family นั้น
// บน RHEL ต้อง --disablerepo=AppStream เพราะ RHEL/Rocky มี mariadb เวอร์ชันเก่ากว่าอยู่ใน AppStream
// ซึ่งจะชนกับ mariadb.org repo ที่เพิ่งเพิ่มเข้าไป (เหตุผลเดียวกับใน log ต้นฉบับ)
func InstallPackages(r *runner.Runner, info *osdetect.Info) error {
	r.SetStep("ติดตั้ง MariaDB Server/Client/Backup และ Galera")

	switch info.Family {
	case osdetect.FamilyRHEL:
		_, err := r.Run("dnf -y install galera-4")
		if err != nil {
			return fmt.Errorf("ติดตั้ง galera-4 ไม่สำเร็จ: %w", err)
		}
		_, err = r.Run("dnf --disablerepo=AppStream --disablerepo=appstream -y install MariaDB-server MariaDB-client MariaDB-backup")
		if err != nil {
			return fmt.Errorf("ติดตั้ง MariaDB-server ไม่สำเร็จ: %w", err)
		}
	case osdetect.FamilyDebian:
		_, err := r.Run("apt-get -y install galera-4 || apt-get -y install galera-3")
		if err != nil {
			return fmt.Errorf("ติดตั้ง galera ไม่สำเร็จ: %w", err)
		}
		_, err = r.Run("apt-get -y install mariadb-server mariadb-client mariadb-backup")
		if err != nil {
			return fmt.Errorf("ติดตั้ง mariadb-server ไม่สำเร็จ: %w", err)
		}
	default:
		return fmt.Errorf("ไม่รองรับ OS family: %s", info.Family)
	}
	return nil
}

// StartService สั่ง systemd ให้เปิดและ enable mariadb service พร้อมตรวจสถานะ
func StartService(r *runner.Runner) error {
	r.SetStep("เปิดใช้งาน MariaDB service")
	r.RunIgnoreErr("systemctl daemon-reload")
	if _, err := r.Run("systemctl enable --now mariadb"); err != nil {
		return fmt.Errorf("เปิด mariadb service ไม่สำเร็จ: %w", err)
	}
	status, _ := r.Run("systemctl is-active mariadb")
	fmt.Printf("    สถานะ mariadb service: %s\n", status)
	return nil
}
