# Repository Guide

ไฟล์นี้มีไว้ให้ AI/agent ตัวถัดไปอ่านก่อนแก้ repo นี้

## ภาพรวมโปรเจกต์

- โปรเจกต์นี้คือ `mariadb-installer` เขียนด้วย Go
- หน้าที่หลักคือเชื่อมต่อ Linux ปลายทางผ่าน SSH แล้วติดตั้ง/ปรับแต่ง MariaDB หรือ Galera Cluster
- ตัวโปรแกรมรันบนเครื่อง local ได้ แต่คำสั่งจริงทั้งหมดไปรันบน remote host เท่านั้น
- รองรับ 2 กลุ่ม OS หลัก: RHEL-family (`dnf`, `firewalld`) และ Debian-family (`apt`)

## โครงสร้างที่ควรรู้

- `main.go` เป็น entrypoint และจัด flow ของงานทั้งหมด
- `internal/steps/` เก็บขั้นตอนย่อย เช่น cleanup, repo, install, sysctl, my.cnf, galera
- `internal/osdetect/` ตรวจชนิด OS และสรุป family/arch สำหรับใช้เลือก repo
- `internal/sshclient/` จัดการ SSH connection และการเขียนไฟล์ไปยัง remote host
- `internal/runner/` เป็น abstraction สำหรับรันคำสั่ง, log, dry-run, และเขียนไฟล์
- `internal/tuning/` คำนวณค่าปรับแต่งจาก RAM จริงของเครื่องปลายทาง
- `internal/examples/` สร้างตัวอย่างคำสั่ง `--examples`

## Build และ Run

- ใช้ Go `1.22`
- dependency ถูก vendor ไว้แล้ว จึงควร build ด้วย `-mod=vendor`
- คำสั่ง build หลักบน Windows คือ `go build -mod=vendor -o mariadb-installer.exe .`
- ถ้าเจอปัญหา cache บน Windows ให้ตั้ง `GOCACHE` ไปที่โฟลเดอร์ใน workspace ก่อน build
- อย่าแก้ dependency โดยไม่จำเป็น และอย่าดึงแพ็กเกจจาก network ถ้าไม่ต้องใช้

## ลำดับงานหลัก

1. ตรวจว่าผู้ใช้เป็น `root`
2. ตรวจ OS และ architecture ของ remote host
3. คำนวณ tuning จาก RAM จริง
4. cleanup ของ MariaDB/MySQL เก่า ถ้าไม่ได้ข้ามด้วย `--skip-cleanup`
5. ตั้ง firewall และ sysctl
6. เพิ่ม MariaDB repository
7. เขียน `my.cnf`
8. ติดตั้งแพ็กเกจ MariaDB/Galera
9. ถ้าเป็น cluster ให้เขียน `galera.cnf`
10. start service หรือ bootstrap node แรก

## จุดเสี่ยง

- งานนี้แก้ remote host จริงผ่าน SSH ดังนั้นห้ามสมมติว่าคำสั่ง local เป็น target
- หลีกเลี่ยงการเปลี่ยน behavior ที่กระทบการลบข้อมูลใน `/var/lib/mysql`
- ถ้าแก้ repo logic ต้องตรวจทั้ง Debian และ RHEL path
- ถ้าแก้ host config ต้องรักษา compatibility กับ single-host และ multi-host/Galera mode
- ถ้าแก้ tuning ควรตรวจผลกระทบต่อ `my.cnf`, sysctl, และ Galera config ร่วมกัน

## แนวทางแก้โค้ด

- แก้ root cause มากกว่าปะแก้เฉพาะจุด
- ใช้ `apply_patch` สำหรับการแก้ไฟล์
- อย่าแก้ไฟล์ build artifact เช่น `.exe` หรือ `.gocache`
- ถ้าเพิ่มพฤติกรรมใหม่ ให้ดูว่า `--dry-run` และ `--apply` ต้องแสดงผลสอดคล้องกันหรือไม่
- ถ้าเพิ่ม flag ใหม่ ให้แก้ทั้ง `main.go` และ README ที่เกี่ยวข้อง

## สิ่งที่ควรตรวจก่อนปิดงาน

- `go build -mod=vendor -o mariadb-installer.exe .`
- ถ้าแก้ flow หลัก ให้ตรวจ output ของ `--examples` หรือ `--dry-run`
- ถ้าแก้ repo/install ให้ดูว่า command ที่สร้างขึ้นตรงกับ distro family ที่รองรับ

