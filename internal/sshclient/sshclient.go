// Package sshclient เปิดการเชื่อมต่อ SSH ไปยังเครื่อง Linux ปลายทาง
// เพื่อให้ runner.Runner สั่งคำสั่ง/เขียนไฟล์ระยะไกลได้ โดยไม่ต้องรันโปรแกรมนี้บน Linux เอง
//
// รองรับการ authenticate สองแบบ: private key (.pem ในรูปแบบ OpenSSH) และ password
package sshclient

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// AuthMethod ระบุว่าจะ authenticate ด้วยอะไร
type AuthMethod string

const (
	AuthKey      AuthMethod = "key"
	AuthPassword AuthMethod = "password"
)

// Config คือข้อมูลการเชื่อมต่อไปยัง Linux target หนึ่งเครื่อง
type Config struct {
	Host          string // เช่น "192.168.1.10" หรือ "db1.example.com"
	Port          int    // ปกติ 22
	User          string // ปกติ "root" ตามที่ระบุไว้
	AuthMethod    AuthMethod
	KeyPath       string // path ไปยัง private key file (.pem) ใช้เมื่อ AuthMethod == AuthKey
	KeyPassphrase string // ถ้า private key ถูกเข้ารหัสด้วย passphrase
	Password      string // ใช้เมื่อ AuthMethod == AuthPassword
	// HostKeyCheck ถ้า true จะตรวจ host key กับ known_hosts (ปลอดภัยกว่า)
	// ถ้า false จะข้ามการตรวจ (สะดวกสำหรับ VM ที่เพิ่งสร้าง แต่เสี่ยง MITM)
	HostKeyCheck   bool
	KnownHostsPath string // ใช้เมื่อ HostKeyCheck == true
	Timeout        time.Duration
}

// Client คือ session SSH ที่เปิดอยู่กับเครื่องปลายทางหนึ่งเครื่อง
type Client struct {
	cfg    Config
	client *ssh.Client
}

// Dial เปิดการเชื่อมต่อ SSH ไปยัง target ตาม Config ที่ให้มา
func Dial(cfg Config) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}

	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := buildHostKeyCallback(cfg)
	if err != nil {
		return nil, err
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.Timeout,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("เชื่อมต่อ SSH ไปยัง %s ไม่สำเร็จ: %w", addr, err)
	}

	return &Client{cfg: cfg, client: client}, nil
}

// Ping ทดสอบว่า session ที่เชื่อมไว้ยังใช้รันคำสั่งได้จริง (รัน `echo ok` แบบง่าย ๆ)
func (c *Client) Ping() error {
	result, err := c.Run("echo ok")
	if err != nil {
		return err
	}
	if result.Stdout != "ok" {
		return fmt.Errorf("ผลลัพธ์ทดสอบไม่ตรงตามคาด: %q", result.Stdout)
	}
	return nil
}

func buildAuthMethods(cfg Config) ([]ssh.AuthMethod, error) {
	switch cfg.AuthMethod {
	case AuthKey:
		keyData, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("อ่าน private key ที่ %s ไม่สำเร็จ: %w", cfg.KeyPath, err)
		}
		var signer ssh.Signer
		if cfg.KeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(cfg.KeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key ไม่สำเร็จ (key ผิด format หรือ passphrase ไม่ถูก): %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil

	case AuthPassword:
		if cfg.Password == "" {
			return nil, fmt.Errorf("ระบุ AuthMethod เป็น password แต่ไม่ได้ใส่รหัสผ่าน")
		}
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil

	default:
		return nil, fmt.Errorf("ไม่รู้จัก AuthMethod: %q (รองรับ %q หรือ %q)", cfg.AuthMethod, AuthKey, AuthPassword)
	}
}

// buildHostKeyCallback สร้างตัวตรวจ host key ตาม config
// ค่า default คือ InsecureIgnoreHostKey เพื่อความสะดวกในการใช้งานครั้งแรกกับ VM ใหม่
// แนะนำให้เปิด HostKeyCheck=true ใน production จริงเพื่อป้องกัน MITM
func buildHostKeyCallback(cfg Config) (ssh.HostKeyCallback, error) {
	if !cfg.HostKeyCheck {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	return knownHostsCallback(cfg.KnownHostsPath)
}

// Close ปิดการเชื่อมต่อ SSH
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// CommandResult คือผลลัพธ์ของคำสั่งที่รันบน remote host
// แยก Stdout/Stderr ออกจากกันเพื่อให้ runner เลือกแสดงผลหรือ parse ได้ง่ายขึ้น
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run รันคำสั่งหนึ่งคำสั่งบนเครื่องปลายทางผ่าน SSH exec channel
// คืนค่า CommandResult (stdout/stderr แยกกัน, trim แล้ว) และ error ถ้า exit code != 0
func (c *Client) Run(cmd string) (CommandResult, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return CommandResult{}, fmt.Errorf("เปิด SSH session ไม่สำเร็จ: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	runErr := session.Run(cmd)

	result := CommandResult{
		Stdout: trimOutput(stdoutBuf.Bytes()),
		Stderr: trimOutput(stderrBuf.Bytes()),
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			result.ExitCode = -1
		}
		return result, fmt.Errorf(
			"คำสั่งล้มเหลวบน %s: %q (exit=%d): %w\nstderr: %s",
			c.cfg.Host, cmd, result.ExitCode, runErr, result.Stderr,
		)
	}

	return result, nil
}

func trimOutput(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
