package sshclient

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// knownHostsCallback โหลดไฟล์ known_hosts (ค่า default ใช้ตำแหน่งมาตรฐานของ OpenSSH)
// เพื่อตรวจสอบ host key ของเครื่องปลายทางตอนเชื่อมต่อ ป้องกัน MITM attack
func knownHostsCallback(path string) (ssh.HostKeyCallback, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("หา home directory ไม่สำเร็จ เพื่อหาไฟล์ known_hosts default: %w", err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf(
			"ไม่พบไฟล์ known_hosts ที่ %s (เปิด HostKeyCheck=true ต้องมีไฟล์นี้ "+
				"หรือ SSH เข้าเครื่องปลายทางด้วยมือหนึ่งครั้งก่อนเพื่อให้ OpenSSH สร้างไฟล์นี้ให้): %w",
			path, err,
		)
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("อ่านไฟล์ known_hosts ที่ %s ไม่สำเร็จ: %w", path, err)
	}
	return callback, nil
}
