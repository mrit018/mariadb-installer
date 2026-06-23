package sshclient

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// WriteFile เขียนไฟล์บนเครื่องปลายทาง โดยส่งเนื้อหาผ่าน stdin ของคำสั่ง `cat > path`
// ถ้า login ด้วย user ที่ไม่ใช่ root จะ prefix ด้วย sudo ให้อัตโนมัติ
// แทนการใช้ SFTP (ลด dependency ลงหนึ่งตัว) วิธีนี้ใช้ได้กับทุก path ที่ user ปัจจุบันเขียนถึงได้
func (c *Client) WriteFile(path, content string, mode os.FileMode) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("เปิด SSH session ไม่สำเร็จ: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(path), mode.Perm(), shellQuote(path))
	if c.cfg.User != "root" {
		cmd = sudoShellCommand(cmd, c.cfg.SudoPassword != "")
		if c.cfg.SudoPassword != "" {
			payload := append([]byte(c.cfg.SudoPassword+"\n"), []byte(content)...)
			session.Stdin = bytes.NewReader(payload)
		} else {
			session.Stdin = bytes.NewReader([]byte(content))
		}
	} else {
		session.Stdin = bytes.NewReader([]byte(content))
	}

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("เขียนไฟล์ %s บน %s ไม่สำเร็จ: %w\nผลลัพธ์: %s", path, c.cfg.Host, err, trimOutput(out))
	}
	return nil
}

// shellQuote ครอบ path ด้วย single quote เพื่อป้องกัน path ที่มีช่องว่างหรือ special char
// ทำให้คำสั่ง shell ที่ส่งไปไม่แตก (escape single quote ภายใน path ด้วย '\” ตามมาตรฐาน POSIX shell)
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
