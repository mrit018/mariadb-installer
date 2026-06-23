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
// วิธีใช้ (หลายเครื่อง / Galera cluster) ผ่านไฟล์ config:
//
//	mariadb-installer.exe --apply --config=hosts.json
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"mariadb-installer/internal/examples"
	"mariadb-installer/internal/hostsconfig"
	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
	"mariadb-installer/internal/sshclient"
	"mariadb-installer/internal/steps"
	"mariadb-installer/internal/tuning"
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

		// โหมดหลายเครื่อง
		configPath = flag.String("config", "", "path ไปยังไฟล์ config JSON สำหรับติดตั้งหลายเครื่อง/Galera cluster")

		// โหมดแสดงตัวอย่าง (ไม่เชื่อมต่อ SSH ใด ๆ แค่พิมพ์ตัวอย่างคำสั่งออกมาดู)
		showExamples = flag.Bool("examples", false,
			"แสดงตัวอย่างการสั่งงานทุกสถานการณ์ (เครื่องเดียว, Galera cluster, troubleshooting) แล้วออกจากโปรแกรม")
		examplesCategory = flag.String("examples-category", "",
			fmt.Sprintf("กรองตัวอย่างเฉพาะหมวด (ใช้คู่กับ --examples) เลือกได้: %s", strings.Join(examples.AvailableCategories(), ", ")))
	)
	flag.Parse()

	if *showExamples || *examplesCategory != "" {
		exList, ok := examples.Filter(*examplesCategory)
		if !ok {
			fmt.Fprintln(os.Stderr, examples.ErrUnknownCategoryHint(*examplesCategory))
			os.Exit(1)
		}
		examples.Print(exList)
		return
	}

	if !*dryRun && !*apply {
		fmt.Fprintln(os.Stderr, "ต้องระบุ --dry-run (ดูแผนงานก่อน) หรือ --apply (ติดตั้งจริง) อย่างใดอย่างหนึ่ง")
		fmt.Fprintln(os.Stderr, "หรือดูตัวอย่างการสั่งงานทั้งหมดด้วย --examples")
		flag.Usage()
		os.Exit(1)
	}
	if *dryRun && *apply {
		fmt.Fprintln(os.Stderr, "ระบุได้แค่ --dry-run หรือ --apply อย่างเดียว ไม่ใช่ทั้งสองอย่าง")
		os.Exit(1)
	}

	targets, clusterName, err := resolveTargets(*configPath, *host, *sshPort, *user, *keyPath, *password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("=== DRY-RUN MODE: จะไม่มีการเชื่อมต่อ/แก้ไขเครื่องปลายทางจริง มีแต่การแสดงแผนงาน ===")
	} else {
		fmt.Println("=== APPLY MODE: กำลังติดตั้งจริงผ่าน SSH ===")
	}

	isCluster := clusterName != "" && len(targets) > 1
	if isCluster {
		fmt.Printf("โหมด Galera cluster: %q จำนวน %d node\n", clusterName, len(targets))
	}

	allAddresses := make([]string, len(targets))
	for i, t := range targets {
		allAddresses[i] = t.cfg.Host
	}

	// ติดตั้งทุกเครื่องตามลำดับ (ไม่ทำพร้อมกัน เพื่อให้ log อ่านง่ายและไม่ชนกันตอน clone SST จาก node แรก)
	for i, t := range targets {
		if err := runOnHost(t, *dryRun, *verbose, *charset, *skipCleanup, isCluster, clusterName, allAddresses, i == 0); err != nil {
			fmt.Fprintf(os.Stderr, "\n[FATAL] เครื่อง %s ล้มเหลว: %v\n", t.cfg.Host, err)
			os.Exit(1)
		}
	}

	fmt.Println("\n=== เสร็จสมบูรณ์ทุกเครื่อง ===")
	if *dryRun {
		fmt.Println("นี่คือ dry-run เท่านั้น รันด้วย --apply เพื่อติดตั้งจริง")
	} else {
		fmt.Println("ตรวจสอบสถานะแต่ละเครื่องด้วย: ssh <host> systemctl status mariadb")
		fmt.Println("รัน mysql_secure_installation บนแต่ละเครื่องเพื่อตั้งรหัสผ่าน root")
	}
}

// target คือเครื่องปลายทางหนึ่งเครื่องพร้อมข้อมูลที่ต้องใช้เชื่อมต่อ
type target struct {
	name string // ชื่ออ้างอิง (ใช้เป็น wsrep_node_name เมื่อทำคลัสเตอร์)
	cfg  sshclient.Config
}

// resolveTargets อ่านค่าจาก flag เครื่องเดียว หรือไฟล์ config หลายเครื่อง (--config มาก่อนถ้าระบุทั้งคู่)
func resolveTargets(configPath, host string, sshPort int, user, keyPath, password string) ([]target, string, error) {
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
	return []target{{name: host, cfg: sc}}, "", nil
}

// runOnHost รัน pipeline การติดตั้งทั้งหมดบนเครื่องปลายทางหนึ่งเครื่อง
func runOnHost(
	t target,
	dryRun, verbose bool,
	charset string,
	skipCleanup bool,
	isCluster bool,
	clusterName string,
	allAddresses []string,
	isFirstNode bool,
) error {
	hostLabel := fmt.Sprintf("%s@%s", t.cfg.User, t.cfg.Host)

	var sshConn *sshclient.Client
	if !dryRun {
		conn, err := sshclient.Dial(t.cfg)
		if err != nil {
			return err
		}
		defer conn.Close()
		if err := conn.Ping(); err != nil {
			return fmt.Errorf("เชื่อมต่อ SSH สำเร็จแต่รันคำสั่งทดสอบไม่ผ่าน: %w", err)
		}
		sshConn = conn
		fmt.Printf("เชื่อมต่อ %s สำเร็จ\n", hostLabel)
	}

	r := runner.New(dryRun, verbose, sshConn, hostLabel)

	if err := steps.EnsureRoot(r); err != nil {
		return err
	}

	info, err := osdetect.Detect(r)
	if err != nil {
		return fmt.Errorf("ตรวจ OS ไม่สำเร็จ: %w", err)
	}
	fmt.Printf("[%s] ตรวจพบ OS: %s\n", hostLabel, info)

	if info.Family == osdetect.FamilyUnknown {
		return fmt.Errorf("ไม่รองรับ OS นี้ (รองรับเฉพาะ RHEL-family และ Debian-family)")
	}

	tval, err := tuning.Detect(r)
	if err != nil {
		return fmt.Errorf("คำนวณค่า tuning จาก RAM ไม่สำเร็จ: %w", err)
	}
	fmt.Printf("[%s] ค่า tuning ที่คำนวณได้: %s\n", hostLabel, tval)

	pipeline := buildPipeline(r, info, tval, charset, skipCleanup)

	if isCluster {
		pipeline = append(pipeline, pipelineStep{"galera-config", func() error {
			return steps.WriteGaleraConfig(r, steps.GaleraOptions{
				ClusterName:  clusterName,
				NodeName:     t.name,
				NodeAddress:  t.cfg.Host,
				AllAddresses: allAddresses,
			})
		}})
	}

	for _, step := range pipeline {
		if err := step.fn(); err != nil {
			return fmt.Errorf("ขั้นตอน %q ล้มเหลว: %w", step.name, err)
		}
	}

	if isCluster && isFirstNode {
		fmt.Printf("[%s] เป็น node แรกของคลัสเตอร์ ต้อง bootstrap ก่อน node อื่น\n", hostLabel)
		if err := steps.BootstrapFirstNode(r); err != nil {
			return err
		}
	} else {
		if err := steps.StartService(r); err != nil {
			return err
		}
	}

	return nil
}

// pipelineStep คือหนึ่งขั้นตอนในแผนการติดตั้ง ตั้งชื่อไว้สำหรับ error message ที่อ่านง่าย
type pipelineStep struct {
	name string
	fn   func() error
}

// buildPipeline ประกอบลำดับขั้นตอนการติดตั้งทั้งหมด
// precheck service -> cleanup -> firewall -> selinux -> sysctl tuning -> repo -> my.cnf -> install
// (galera config และ start service ถูกเพิ่มต่อจาก pipeline นี้แยกใน runOnHost
// เพราะ node แรกของคลัสเตอร์ต้อง bootstrap แทนการ start แบบปกติ)
func buildPipeline(
	r *runner.Runner,
	info *osdetect.Info,
	tval *tuning.Values,
	charset string,
	skipCleanup bool,
) []pipelineStep {
	var pipeline []pipelineStep

	pipeline = append(pipeline, pipelineStep{"precheck-services", func() error {
		return steps.CheckExistingDatabaseServices(r)
	}})

	if !skipCleanup {
		pipeline = append(pipeline, pipelineStep{"cleanup", func() error { return steps.CleanupOld(r, info) }})
	}

	pipeline = append(pipeline,
		pipelineStep{"firewall", func() error { return steps.ConfigureFirewall(r, info) }},
		pipelineStep{"selinux", func() error { return steps.DisableSELinux(r, info) }},
		pipelineStep{"sysctl", func() error { return steps.TuneKernel(r, info, tval) }},
		pipelineStep{"repo", func() error { return steps.AddMariaDBRepo(r, info) }},
		pipelineStep{"my.cnf", func() error {
			opt := steps.DefaultMyCnfOptions()
			opt.CharacterSet = charset
			return steps.WriteMyCnf(r, tval, opt)
		}},
		pipelineStep{"install", func() error { return steps.InstallPackages(r, info) }},
	)

	return pipeline
}
