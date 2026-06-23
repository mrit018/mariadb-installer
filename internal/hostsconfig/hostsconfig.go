// Package hostsconfig อ่านไฟล์ config (รูปแบบ JSON) ที่ระบุเครื่อง Linux ปลายทาง
// หลายเครื่องในไฟล์เดียว ใช้ตอนต้องการติดตั้งพร้อมกันหลายเครื่อง หรือตั้ง Galera cluster
//
// ตัวอย่างไฟล์ hosts.json:
//
//	{
//	  "cluster_name": "prod-galera",
//	  "user": "root",
//	  "auth": "key",
//	  "key_path": "C:\\Users\\me\\.ssh\\id_rsa",
//	  "hosts": [
//	    { "name": "db1", "address": "10.0.0.11" },
//	    { "name": "db2", "address": "10.0.0.12" }
//	  ]
//	}
//
// ใช้ JSON (ไม่ใช่ YAML) เพราะเป็น standard library ของ Go โดยตรง ไม่ต้องดึง
// dependency เพิ่ม และผู้ใช้ทั่วไปก็แก้ไฟล์ JSON ง่ายด้วย editor ทั่วไปอยู่แล้ว
package hostsconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// HostEntry คือเครื่อง Linux ปลายทางหนึ่งเครื่องในไฟล์ config
type HostEntry struct {
	Name    string `json:"name"`    // ชื่ออ้างอิงสั้น ๆ (ใช้เป็น wsrep_node_name ถ้าทำคลัสเตอร์)
	Address string `json:"address"` // IP หรือ hostname
	Port    int    `json:"port,omitempty"`
}

// Config คือเนื้อหาทั้งไฟล์ config สำหรับติดตั้งหลายเครื่อง
type Config struct {
	ClusterName string      `json:"cluster_name,omitempty"` // ใส่เมื่อต้องการตั้งเป็น Galera cluster
	User        string      `json:"user"`
	Auth        string      `json:"auth"` // "key" หรือ "password"
	KeyPath     string      `json:"key_path,omitempty"`
	Password    string      `json:"password,omitempty"`
	Hosts       []HostEntry `json:"hosts"`
}

// Load อ่านและ parse ไฟล์ config จาก path ที่ระบุ
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("อ่านไฟล์ config %q ไม่สำเร็จ: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ไฟล์ config %q ไม่ใช่ JSON ที่ถูกต้อง: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("ไฟล์ config %q ไม่ถูกต้อง: %w", path, err)
	}
	return &cfg, nil
}

// Validate ตรวจความถูกต้องพื้นฐานของ config ก่อนนำไปใช้
func (c *Config) Validate() error {
	if len(c.Hosts) == 0 {
		return fmt.Errorf("ต้องระบุ host อย่างน้อย 1 เครื่องใน \"hosts\"")
	}
	if c.User == "" {
		return fmt.Errorf("ต้องระบุ \"user\" (เช่น \"root\")")
	}
	switch c.Auth {
	case "key":
		if c.KeyPath == "" {
			return fmt.Errorf(`auth = "key" ต้องระบุ "key_path" ด้วย`)
		}
	case "password":
		if c.Password == "" {
			return fmt.Errorf(`auth = "password" ต้องระบุ "password" ด้วย`)
		}
	default:
		return fmt.Errorf(`"auth" ต้องเป็น "key" หรือ "password" เท่านั้น (พบ %q)`, c.Auth)
	}
	for i, h := range c.Hosts {
		if h.Address == "" {
			return fmt.Errorf("hosts[%d] ไม่มี \"address\"", i)
		}
		if h.Name == "" {
			return fmt.Errorf("hosts[%d] ไม่มี \"name\"", i)
		}
	}
	return nil
}

// IsCluster คืนค่าจริงถ้า config นี้ตั้งใจให้เป็น Galera cluster (ระบุ cluster_name
// และมีมากกว่า 1 host)
func (c *Config) IsCluster() bool {
	return c.ClusterName != "" && len(c.Hosts) > 1
}

// Addresses คืน address ของทุก host (ใช้สร้าง wsrep_cluster_address)
func (c *Config) Addresses() []string {
	addrs := make([]string, len(c.Hosts))
	for i, h := range c.Hosts {
		addrs[i] = h.Address
	}
	return addrs
}
