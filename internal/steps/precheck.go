package steps

import (
	"fmt"
	"strings"

	"mariadb-installer/internal/runner"
)

// CheckExistingDatabaseServices ตรวจว่า service ที่เกี่ยวกับ MySQL/MariaDB/Percona
// ยัง active อยู่หรือไม่ ถ้ายัง active จะหยุด pipeline ทันทีและบอกให้ stop service ก่อน
func CheckExistingDatabaseServices(r *runner.Runner) error {
	r.SetStep("ตรวจสอบ service เดิมก่อนติดตั้ง")

	if r.DryRun {
		r.RunIgnoreErr("systemctl is-active mariadb mysql mysqld percona-server percona-xtradb-cluster")
		return nil
	}

	services := []string{
		"mariadb",
		"mysql",
		"mysqld",
		"percona-server",
		"percona-xtradb-cluster",
	}

	var active []string
	for _, svc := range services {
		out, err := r.Run(fmt.Sprintf("systemctl is-active %s", svc))
		if err == nil && strings.TrimSpace(out) == "active" {
			active = append(active, svc)
		}
	}

	if len(active) > 0 {
		return fmt.Errorf("พบ service ที่ยัง active อยู่: %s. "+
			"กรุณา stop service เหล่านี้ก่อน เช่น `systemctl stop %s` แล้วรัน installer ใหม่",
			strings.Join(active, ", "), strings.Join(active, " "))
	}

	return nil
}
