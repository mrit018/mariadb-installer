// mariadb-installer ติดตั้งและปรับแต่ง MariaDB + Galera Cluster บนเครื่อง Linux
// ระยะไกลผ่าน SSH โดยตัวโปรแกรมนี้รันได้บน Windows, Linux, หรือ macOS ก็ได้
// (ใช้ SSH สั่งงาน Linux ปลายทางทั้งหมด ไม่มีการรันคำสั่งบนเครื่อง local เลย)
//
// รองรับทั้ง RHEL-family (dnf/firewalld) และ Debian-family (apt) บนเครื่องปลายทาง
// ค่า tuning (innodb_buffer_pool_size, hugepages, shmmax/shmall ฯลฯ) คำนวณจาก
// RAM จริงของเครื่องปลายทางโดยอัตโนมัติ
//
// วิธีใช้ (เครื่องเดียว):
//
//	# ดูแผนการทำงานก่อน โดยยังไม่เชื่อมต่อ SSH จริง
//	mariadb-installer.exe --dry-run --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"
//
//	# ติดตั้งจริงด้วย private key
//	mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"
//
//	# ติดตั้งจริงด้วย password
//	mariadb-installer.exe --apply --host=10.0.0.10 --user=root --password=secret
//
//	# ล็อกอินด้วย user อื่นแล้วใช้ sudo ต่อ (ถ้า sudo ต้องใช้ password)
//	mariadb-installer.exe --apply --host=10.0.0.10 --user=mrit --password=ssh-pass --sudo-password=sudo-pass
//
// วิธีใช้ (หลายเครื่อง / Galera cluster) ผ่านไฟล์ config:
//
//	mariadb-installer.exe --apply --config=hosts.json
package main

import (
	"flag"
	"fmt"
	"mariadb-installer/internal/hostsconfig"
	"mariadb-installer/internal/sshclient"
	"os"
)

func main() {
	var (
		dryRun      = flag.Bool("dry-run", false, "แสดงแผนการทำงานทั้งหมดโดยไม่เชื่อมต่อ/แก้ไขเครื่องปลายทางจริง")
		apply       = flag.Bool("apply", false, "ลงมือติดตั้งจริงผ่าน SSH")
		verbose     = flag.Bool("verbose", false, "แสดง output ของทุกคำสั่งแบบละเอียด")
		charset     = flag.String("charset", "utf8mb4", "character set ของ MariaDB (เช่น utf8mb4, tis620)")
		skipCleanup = flag.Bool("skip-cleanup", false, "ข้ามขั้นตอนลบ MySQL/MariaDB เดิมบนเครื่องปลายทาง")

		// โหมดเครื่องเดียว
		host     = flag.String("host", "", "IP/hostname ของเครื่อง Linux ปลายทาง (โหมดเครื่องเดียว)")
		sshPort  = flag.Int("ssh-port", 22, "SSH port ของเครื่องปลายทาง")
		user     = flag.String("user", "root", "ผู้ใช้ที่ใช้ SSH login")
		keyPath  = flag.String("key", "", "path ไปยัง private key file (.pem) สำหรับ SSH auth")
		password = flag.String("password", "", "password สำหรับ SSH auth (ใช้แทน --key ก็ได้)")
		sudoPass = flag.String("sudo-password", "", "password สำหรับ sudo เมื่อ login ด้วย user ที่ไม่ใช่ root")

		// โหมดหลายเครื่อง
		configPath = flag.String("config", "", "path ไปยังไฟล์ config JSON สำหรับติดตั้งหลายเครื่อง/Galera cluster")

		// โหมด GUI
		gui        = flag.Bool("gui", false, "เปิดหน้า GUI local ใน browser สำหรับกรอกค่าได้ง่ายขึ้น")
		guiPort    = flag.Int("gui-port", 8079, "port สำหรับ GUI local")
		desktopGUI = flag.Bool("desktop-gui", false, "เปิด native Windows desktop GUI")

		// โหมดแสดงตัวอย่าง (ไม่เชื่อมต่อ SSH ใด ๆ แค่พิมพ์ตัวอย่างคำสั่งออกมาดู)
		showExamples = flag.Bool("examples", false,
			"แสดงตัวอย่างการสั่งงานทุกสถานการณ์ (เครื่องเดียว, Galera cluster, troubleshooting) แล้วออกจากโปรแกรม")
		examplesCategory = flag.String("examples-category", "",
			fmt.Sprintf("กรองตัวอย่างเฉพาะหมวด (ใช้คู่กับ --examples)"))
	)
	flag.Parse()

	if *desktopGUI {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
			os.Exit(1)
		}
		if err := startDesktopGUI(exe); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *gui {
		if err := startGUI(*guiPort); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
			os.Exit(1)
		}
		return
	}

	err := runApp(appOptions{
		DryRun:           *dryRun,
		Apply:            *apply,
		Verbose:          *verbose,
		Charset:          *charset,
		SkipCleanup:      *skipCleanup,
		Host:             *host,
		SSHPort:          *sshPort,
		User:             *user,
		KeyPath:          *keyPath,
		Password:         *password,
		SudoPassword:     *sudoPass,
		ConfigPath:       *configPath,
		ShowExamples:     *showExamples,
		ExamplesCategory: *examplesCategory,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
		os.Exit(1)
	}
}

// target คือเครื่องปลายทางหนึ่งเครื่องพร้อมข้อมูลที่ต้องใช้เชื่อมต่อ
type target struct {
	name string // ชื่ออ้างอิง (ใช้เป็น wsrep_node_name เมื่อทำคลัสเตอร์)
	cfg  sshclient.Config
}

// resolveTargets อ่านค่าจาก flag เครื่องเดียว หรือไฟล์ config หลายเครื่อง (--config มาก่อนถ้าระบุทั้งคู่)
func resolveTargets(configPath, host string, sshPort int, user, keyPath, password, sudoPassword string) ([]target, string, error) {
	if configPath != "" {
		cfg, err := hostsconfig.Load(configPath)
		if err != nil {
			return nil, "", err
		}
		targets := make([]target, len(cfg.Hosts))
		for i, h := range cfg.Hosts {
			port := h.Port
			if port == 0 {
				port = 22
			}
			sc := sshclient.Config{
				Host: h.Address,
				Port: port,
				User: cfg.User,
			}
			if cfg.Auth == "key" {
				sc.AuthMethod = sshclient.AuthKey
				sc.KeyPath = cfg.KeyPath
			} else {
				sc.AuthMethod = sshclient.AuthPassword
				sc.Password = cfg.Password
			}
			sc.SudoPassword = cfg.SudoPassword
			if sc.User != "root" && sc.SudoPassword == "" {
				sc.SudoPassword = sc.Password
			}
			targets[i] = target{name: h.Name, cfg: sc}
		}
		return targets, cfg.ClusterName, nil
	}

	// โหมดเครื่องเดียวผ่าน flag
	if host == "" {
		return nil, "", fmt.Errorf("ต้องระบุ --host (เครื่องเดียว) หรือ --config (หลายเครื่อง)")
	}
	sc := sshclient.Config{Host: host, Port: sshPort, User: user}
	switch {
	case keyPath != "":
		sc.AuthMethod = sshclient.AuthKey
		sc.KeyPath = keyPath
	case password != "":
		sc.AuthMethod = sshclient.AuthPassword
		sc.Password = password
	default:
		return nil, "", fmt.Errorf("ต้องระบุ --key หรือ --password อย่างใดอย่างหนึ่งสำหรับ SSH auth")
	}
	sc.SudoPassword = sudoPassword
	if sc.User != "root" && sc.SudoPassword == "" {
		sc.SudoPassword = sc.Password
	}
	return []target{{name: host, cfg: sc}}, "", nil
}
