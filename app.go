package main

import (
	"fmt"

	"mariadb-installer/internal/examples"
	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
	"mariadb-installer/internal/sshclient"
	"mariadb-installer/internal/steps"
	"mariadb-installer/internal/tuning"
)

type appOptions struct {
	DryRun             bool
	Apply              bool
	Verbose            bool
	Charset            string
	SkipCleanup        bool
	AutoConfirmCleanup bool

	Host         string
	SSHPort      int
	User         string
	KeyPath      string
	Password     string
	SudoPassword string

	ConfigPath       string
	ShowExamples     bool
	ExamplesCategory string
}

func runApp(opts appOptions) error {
	if opts.ShowExamples || opts.ExamplesCategory != "" {
		exList, ok := examples.Filter(opts.ExamplesCategory)
		if !ok {
			return fmt.Errorf(examples.ErrUnknownCategoryHint(opts.ExamplesCategory))
		}
		examples.Print(exList)
		return nil
	}

	if !opts.DryRun && !opts.Apply {
		return fmt.Errorf("ต้องระบุ --dry-run (ดูแผนงานก่อน) หรือ --apply (ติดตั้งจริง) อย่างใดอย่างหนึ่ง")
	}
	if opts.DryRun && opts.Apply {
		return fmt.Errorf("ระบุได้แค่ --dry-run หรือ --apply อย่างเดียว ไม่ใช่ทั้งสองอย่าง")
	}

	targets, clusterName, err := resolveTargets(opts.ConfigPath, opts.Host, opts.SSHPort, opts.User, opts.KeyPath, opts.Password, opts.SudoPassword)
	if err != nil {
		return err
	}

	if opts.DryRun {
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

	for i, t := range targets {
		if err := runOnHost(t, opts.DryRun, opts.Verbose, opts.Charset, opts.SkipCleanup, isCluster, clusterName, allAddresses, i == 0, opts.AutoConfirmCleanup); err != nil {
			return fmt.Errorf("เครื่อง %s ล้มเหลว: %w", t.cfg.Host, err)
		}
	}

	fmt.Println("\n=== เสร็จสมบูรณ์ทุกเครื่อง ===")
	if opts.DryRun {
		fmt.Println("นี่คือ dry-run เท่านั้น รันด้วย --apply เพื่อติดตั้งจริง")
	} else {
		fmt.Println("ตรวจสอบสถานะแต่ละเครื่องด้วย: ssh <host> systemctl status mariadb")
		fmt.Println("รัน mysql_secure_installation บนแต่ละเครื่องเพื่อตั้งรหัสผ่าน root")
	}
	return nil
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
	autoConfirmCleanup bool,
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
	if autoConfirmCleanup {
		r.ConfirmFunc = func(string) bool { return true }
	}

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
