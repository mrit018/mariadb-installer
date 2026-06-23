# mariadb-installer

โปรแกรม Go สำหรับติดตั้งและปรับแต่ง MariaDB + Galera Cluster บนเครื่อง **Linux ปลายทาง**
โดยตัวโปรแกรมเองรันได้บน **Windows** (หรือ Linux/macOS ก็ได้) แล้วสั่งงานเครื่อง Linux
ปลายทางทั้งหมดผ่าน **SSH** — ไม่มีการรันคำสั่งใด ๆ บนเครื่องที่ใช้สั่งงานเลย

รองรับทั้ง RHEL-family (RHEL/CentOS/Rocky/AlmaLinux ใช้ `dnf` + `firewalld`)
และ Debian-family (Debian/Ubuntu ใช้ `apt` + `ufw`/`iptables`) บนเครื่องปลายทาง

ค่า tuning ทั้งหมด (innodb_buffer_pool_size, key_buffer_size, max_heap_table_size,
vm.nr_hugepages, kernel.shmmax, kernel.shmall) **คำนวณจาก RAM จริงของเครื่องปลายทาง
โดยอัตโนมัติ** (อ่านผ่าน SSH) ตามสัดส่วนเดียวกับสคริปต์ต้นฉบับ

## Build

ต้องมี Go >= 1.22 ตัวโปรเจกต์ **vendor dependency ไว้ในตัวแล้ว** (โฟลเดอร์ `vendor/`)
จึง build ได้โดยไม่ต้องต่อเน็ตดึง package ใด ๆ เพิ่ม

**บน Windows** (PowerShell หรือ cmd):

```powershell
go build -mod=vendor -o mariadb-installer.exe .
```

**บน Linux/macOS:**

```bash
go build -mod=vendor -o mariadb-installer .
```

## ดูตัวอย่างการสั่งงานทั้งหมด

ก่อนเริ่มใช้งานจริง ลองดูตัวอย่างคำสั่งที่ครอบคลุมสถานการณ์ต่าง ๆ ได้ทันทีโดยไม่ต้องเปิดเอกสารแยก:

```powershell
# ดูตัวอย่างทั้งหมด (เครื่องเดียว, Galera cluster หลายขนาด, troubleshooting)
.\mariadb-installer.exe --examples

# ดูเฉพาะหมวดเครื่องเดียว
.\mariadb-installer.exe --examples-category=single-host

# ดูเฉพาะหมวด Galera cluster (ตัวอย่าง 2/3/5 node)
.\mariadb-installer.exe --examples-category=galera

# ดูเฉพาะหมวด troubleshooting (error ที่พบบ่อยพร้อมวิธีแก้)
.\mariadb-installer.exe --examples-category=troubleshoot
```

โหมดนี้ไม่เชื่อมต่อ SSH หรือแก้ไขอะไรเลย แค่พิมพ์ตัวอย่างออกมาให้อ่าน

## วิธีใช้

### โหมดเครื่องเดียว (ผ่าน flag)

```powershell
# ดูแผนการทำงานก่อน โดยยังไม่เชื่อมต่อ SSH จริง
.\mariadb-installer.exe --dry-run --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"

# ติดตั้งจริงด้วย private key
.\mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"

# ติดตั้งจริงด้วย password แทน key
.\mariadb-installer.exe --apply --host=10.0.0.10 --user=root --password=ตัวอย่างรหัสผ่าน
```

### โหมดหลายเครื่อง / Galera Cluster (ผ่านไฟล์ config)

สร้างไฟล์ `hosts.json`:

```json
{
  "cluster_name": "prod-galera",
  "user": "root",
  "auth": "key",
  "key_path": "C:\\keys\\id_rsa",
  "hosts": [
    { "name": "db1", "address": "10.0.0.11" },
    { "name": "db2", "address": "10.0.0.12" },
    { "name": "db3", "address": "10.0.0.13" }
  ]
}
```

แล้วรัน:

```powershell
.\mariadb-installer.exe --dry-run --config=hosts.json
.\mariadb-installer.exe --apply --config=hosts.json
```

- ถ้าระบุ `cluster_name` และมีมากกว่า 1 host จะถูกตั้งเป็น **Galera cluster** อัตโนมัติ
  (node แรกในลิสต์จะถูก bootstrap ด้วย `galera_new_cluster` ก่อน, node ที่เหลือ
  ใช้ `systemctl start mariadb` ปกติเข้าร่วมคลัสเตอร์)
- ถ้าไม่ระบุ `cluster_name` (หรือมีแค่ 1 host) จะติดตั้งแยกอิสระแต่ละเครื่อง
  ไม่ตั้งค่า Galera ใด ๆ ให้
- ใช้ `"auth": "password"` และ `"password": "..."` แทน `key`/`key_path` ได้ถ้าต้องการ
  authenticate ด้วย password

### Flags

| Flag            | คำอธิบาย                                                          |
|------------------|--------------------------------------------------------------------|
| `--examples`     | แสดงตัวอย่างการสั่งงานทุกสถานการณ์ทั้งหมด แล้วออกจากโปรแกรม          |
| `--examples-category` | กรองตัวอย่างเฉพาะหมวด: `single-host`, `galera`, `troubleshoot`  |
| `--dry-run`      | แสดงแผนงานทั้งหมด **ไม่เชื่อมต่อ SSH จริง** ไม่แก้ไขเครื่องปลายทาง  |
| `--apply`        | ติดตั้งจริงผ่าน SSH                                                |
| `--host`         | IP/hostname ของเครื่องปลายทาง (โหมดเครื่องเดียว)                   |
| `--ssh-port`     | SSH port (default 22)                                              |
| `--user`         | user สำหรับ SSH login (default `root`)                             |
| `--key`          | path ไปยัง private key (.pem) สำหรับ SSH auth                      |
| `--password`     | password สำหรับ SSH auth (ใช้แทน `--key`)                          |
| `--config`       | path ไปยังไฟล์ config JSON (โหมดหลายเครื่อง/Galera cluster)        |
| `--verbose`      | แสดง output ของทุกคำสั่ง + เนื้อหาไฟล์ที่จะเขียน                    |
| `--skip-cleanup` | ข้ามขั้นตอนลบ MySQL/MariaDB เดิมบนเครื่องปลายทาง                   |
| `--charset`      | character set ของ MariaDB (default: `utf8mb4`)                     |

## ลำดับขั้นตอนการทำงาน (รันบนเครื่องปลายทางผ่าน SSH ทั้งหมด)

1. **เชื่อมต่อ SSH** ไปยังเครื่องปลายทาง แล้วตรวจว่า login เป็น **root** จริง
   (จำเป็นเพราะต้องแก้ไฟล์ระบบและติดตั้งแพ็กเกจ)
2. **ตรวจ OS** จาก `/etc/os-release` + `uname -m` บนเครื่องปลายทาง
3. **คำนวณ tuning** จาก `/proc/meminfo`, `getconf PAGESIZE` บนเครื่องปลายทาง
4. **precheck service** — ตรวจว่า `mysql` / `mariadb` / `percona` ยัง active อยู่หรือไม่
   ถ้ายังทำงานอยู่จะหยุดทันทีและแจ้งให้ stop service ก่อน
5. **cleanup** — kill process ค้าง, ลบ package เก่า, ลบ `/var/lib/mysql`
   (ถามยืนยันก่อนลบจริงในโหมด apply)
6. **firewall** — เปิดพอร์ต 3306/4567/4444/33128/3334
7. **selinux** — ปิด SELinux (เฉพาะ RHEL-family)
8. **sysctl** — เขียนค่า kernel tuning ที่คำนวณจาก RAM จริง
9. **repo** — เพิ่ม MariaDB official repo ตาม OS + arch ที่ตรวจพบ
10. **my.cnf** — สร้าง `/etc/my.cnf` พร้อมค่า buffer pool ที่คำนวณจาก RAM จริง
11. **install** — ติดตั้ง `galera-4`, `MariaDB-server/client/backup`
12. **galera config** (เฉพาะโหมดคลัสเตอร์) — เขียน `/etc/my.cnf.d/galera.cnf`
    พร้อม `wsrep_cluster_address` ของทุก node
13. **start service** — bootstrap node แรกด้วย `galera_new_cluster`,
    node อื่นใช้ `systemctl enable --now mariadb`

ทุก node ในโหมดหลายเครื่องจะถูกติดตั้ง **ทีละเครื่องตามลำดับ** (ไม่พร้อมกัน)
เพื่อให้ node แรก bootstrap คลัสเตอร์เสร็จก่อน node อื่นจะเข้าร่วมและ SST ข้อมูล

## หมายเหตุสำคัญ

- โปรแกรมนี้ **ทำลายข้อมูลเดิมบนเครื่องปลายทาง** (ลบ `/var/lib/mysql`)
  ถ้าไม่ใส่ `--skip-cleanup` ตรวจสอบให้แน่ใจว่า backup ข้อมูลแล้วก่อนรันด้วย `--apply`
- หลังติดตั้งเสร็จควร SSH เข้าไปรัน `mysql_secure_installation` บนแต่ละเครื่อง
  เพื่อตั้งรหัสผ่าน root
- ถ้ามี service เดิมของ `mysql` / `mariadb` / `percona` ยัง active อยู่ ระบบจะหยุดก่อน
  เพื่อหลีกเลี่ยงการทับ service เดิม ให้ stop service เหล่านั้นแล้วรันใหม่
- ค่า `character-set` เริ่มต้นคือ `utf8mb4` (รองรับ Unicode/อีโมจิเต็มรูปแบบ)
  แทน `tis620` ที่เห็นใน log ต้นฉบับ ซึ่งเป็น legacy Thai charset ที่ไม่รองรับ Unicode —
  ถ้าจำเป็นต้องใช้ `tis620` จริง ๆ ใส่ `--charset=tis620`
- `HostKeyCheck` ปิดอยู่โดย default (`InsecureIgnoreHostKey`) เพื่อความสะดวกกับ VM ที่
  เพิ่งสร้าง — ถ้าต้องการความปลอดภัยสูงขึ้นใน production ควรแก้ไขโค้ดส่วน
  `internal/sshclient` ให้เปิด `HostKeyCheck: true` พร้อมระบุ `KnownHostsPath`
- `wsrep_sst_auth` ในไฟล์ galera.cnf ตั้งเป็น `root:` (ไม่มีรหัสผ่าน) เพราะยังไม่ได้รัน
  `mysql_secure_installation` ตอนติดตั้ง — ควรอัปเดตค่านี้ให้ตรงกับรหัสผ่าน root จริง
  หลังตั้งรหัสผ่านเสร็จ เพื่อความปลอดภัยของการ sync ข้อมูลระหว่าง node
