// Package runner ให้ตัวช่วยรันคำสั่งและเขียนไฟล์บนเครื่อง Linux ปลายทางผ่าน SSH
// โดยรองรับ dry-run mode (แค่พิมพ์ว่าจะทำอะไร ไม่ลงมือจริง) และ apply mode (ทำจริงผ่าน SSH)
//
// ตัวโปรแกรมหลักรันอยู่บน Windows (หรือ OS ใดก็ได้) แล้วสั่งงาน Linux เป้าหมาย
// ผ่านการเชื่อมต่อ SSH ทั้งหมด ไม่มีการรันคำสั่งบนเครื่อง local เลย
package runner

import (
	"fmt"
	"os"
	"strings"

	"mariadb-installer/internal/sshclient"
)

// Runner คือตัวกลางที่ทุก step เรียกใช้แทนการเรียก ssh client ตรง ๆ
// เพื่อให้ dry-run ครอบคลุมทุกการกระทำในโปรแกรมแบบเดียวกันหมด
type Runner struct {
	DryRun  bool
	Verbose bool

	// ssh เป็น nil ได้เมื่อ DryRun == true และผู้ใช้ยังไม่ได้เชื่อมต่อจริง
	ssh *sshclient.Client

	// HostLabel ใช้แสดงใน log ว่ากำลังทำงานกับ host ไหน (เผื่อกรณีรันหลายเครื่อง)
	HostLabel string

	currentStep string
}

// New สร้าง Runner สำหรับ host หนึ่งเครื่อง
// ssh เป็น nil ได้ถ้า dryRun == true และยังไม่ได้เชื่อมต่อจริง (เช่นดู plan ก่อนโดยไม่ใส่ credential)
func New(dryRun, verbose bool, ssh *sshclient.Client, hostLabel string) *Runner {
	return &Runner{DryRun: dryRun, Verbose: verbose, ssh: ssh, HostLabel: hostLabel}
}

// SetStep ตั้งชื่อ step ปัจจุบันสำหรับแสดงผลใน log
func (r *Runner) SetStep(name string) {
	r.currentStep = name
	fmt.Printf("\n==> [%s] [%s]\n", r.HostLabel, name)
}

func (r *Runner) logf(format string, args ...interface{}) {
	fmt.Printf("    "+format+"\n", args...)
}

// Run รันคำสั่ง shell หนึ่งบรรทัดบน remote host ผ่าน SSH คืนค่า stdout (trim แล้ว) และ error
// ใน dry-run mode จะไม่เชื่อมต่อ SSH จริง แค่ print แล้วคืนค่า "" เสมอ
func (r *Runner) Run(cmd string) (string, error) {
	if r.DryRun {
		r.logf("[DRY-RUN] $ %s", cmd)
		return "", nil
	}
	if r.ssh == nil {
		return "", fmt.Errorf("ยังไม่ได้เชื่อมต่อ SSH กับ %s", r.HostLabel)
	}
	if r.Verbose {
		r.logf("$ %s", cmd)
	}
	result, err := r.ssh.Run(cmd)
	if err != nil {
		return result.Stdout, fmt.Errorf("[%s] %w", r.HostLabel, err)
	}
	if r.Verbose && result.Stdout != "" {
		r.logf("%s", result.Stdout)
	}
	return result.Stdout, nil
}

// RunIgnoreErr รันคำสั่งแบบไม่สนใจ error (ใช้กับ killall/rpm --erase ที่ไม่เจอ process/package ก็ไม่เป็นไร)
func (r *Runner) RunIgnoreErr(cmd string) string {
	out, _ := r.Run(cmd)
	return out
}

// RunMust รันคำสั่งและ exit โปรแกรมทันทีถ้าล้มเหลว (ใช้กับ step ที่พลาดแล้วทำต่อไม่ได้)
func (r *Runner) RunMust(cmd string) string {
	out, err := r.Run(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[FATAL] %v\n", err)
		os.Exit(1)
	}
	return out
}

// WriteFile เขียนไฟล์ทั้งก้อนบน remote host (เทียบเท่า cat > file ผ่าน SSH stdin)
func (r *Runner) WriteFile(path, content string, mode os.FileMode) error {
	if r.DryRun {
		r.logf("[DRY-RUN] เขียนไฟล์ %s (%d บรรทัด)", path, strings.Count(content, "\n")+1)
		if r.Verbose {
			fmt.Println("------ เนื้อหาไฟล์ ------")
			fmt.Println(content)
			fmt.Println("------------------------")
		}
		return nil
	}
	if r.ssh == nil {
		return fmt.Errorf("ยังไม่ได้เชื่อมต่อ SSH กับ %s", r.HostLabel)
	}
	if err := r.ssh.WriteFile(path, content, mode); err != nil {
		return fmt.Errorf("เขียนไฟล์ %s บน %s ไม่สำเร็จ: %w", path, r.HostLabel, err)
	}
	r.logf("เขียนไฟล์ %s แล้ว", path)
	return nil
}

// Confirm ถาม y/N จากผู้ใช้ (บนเครื่องที่รันโปรแกรมนี้ ไม่ใช่บน remote host)
// ก่อนทำ step ที่ทำลายข้อมูล (เช่นลบ /var/lib/mysql เดิมบน remote host)
// ใน dry-run จะข้ามแล้วถือว่า "ตอบ yes" เพื่อให้ดู plan ต่อได้ครบ
func (r *Runner) Confirm(prompt string) bool {
	if r.DryRun {
		r.logf("[DRY-RUN] (จะถาม) %s -> สมมติว่าตอบ yes", prompt)
		return true
	}
	fmt.Printf("    [%s] %s [y/N]: ", r.HostLabel, prompt)
	var line string
	fmt.Scanln(&line)
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
