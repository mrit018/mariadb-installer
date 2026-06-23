package examples

// troubleshootExamples รวมตัวอย่างคำสั่ง/วิธีแก้สำหรับสถานการณ์ error ที่พบบ่อย
// เน้นบอกอาการ (error message ที่จะเห็น) คู่กับวิธีแก้ที่ตรงจุด
func troubleshootExamples() []Example {
	return []Example{
		{
			Category: CategoryTroubleshoot,
			Title:    `Error: "ต้องให้คำสั่งรันเป็น root บน ... (ตรวจพบว่า login เป็น ...)"`,
			Description: "เกิดเมื่อ SSH login ด้วย user อื่นที่ยังไม่ได้ยกระดับสิทธิ์เป็น root " +
				"ให้เพิ่ม --sudo-password ถ้า sudo ของเครื่องนั้นต้องใช้รหัสผ่าน หรือเตรียม sudoers ให้เป็น NOPASSWD",
			Commands: []string{
				`# ผิด: login เป็น user ธรรมดา แต่ยังไม่ให้ sudo password`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=ubuntu --key="C:\keys\id_rsa"`,
				``,
				`# ถูก: login เป็น user ธรรมดาแล้วส่ง sudo password`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=ubuntu --key="C:\keys\id_rsa" --sudo-password="sudo-pass"`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    `Error: "เชื่อมต่อ SSH ไปยัง ... ไม่สำเร็จ" (connection refused / timeout)`,
			Description: "ตรวจ 3 จุดตามลำดับ: (1) เครื่องปลายทางเปิด SSH service จริงไหม " +
				"(2) firewall/security group เปิดพอร์ต SSH ให้เครื่อง Windows นี้เข้าถึงไหม " +
				"(3) --ssh-port ตรงกับที่เครื่องปลายทางใช้จริงไหม (ถ้าไม่ใช่ 22)",
			Commands: []string{
				`# ทดสอบเชื่อมต่อ SSH ตรง ๆ ก่อนด้วย ssh client ทั่วไป (ถ้ามี OpenSSH บน Windows)`,
				`ssh -p 2222 root@10.0.0.10 "echo ok"`,
				``,
				`# ถ้าเชื่อมต่อ ssh ตรงได้ แต่ mariadb-installer ยัง error ให้เช็คว่าใส่ --ssh-port ครบไหม`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --ssh-port=2222 --user=root --key="C:\keys\id_rsa"`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    `Error: "parse private key ไม่สำเร็จ (key ผิด format หรือ passphrase ไม่ถูก)"`,
			Description: "มักเกิดจาก private key เป็นไฟล์ .ppk ของ PuTTY (ไม่ใช่ OpenSSH format) " +
				"หรือ key มี passphrase ป้องกันอยู่ ต้องแปลงเป็น OpenSSH format ก่อน " +
				"(เช่นใช้ PuTTYgen แปลง .ppk -> OpenSSH .pem) แล้วใส่ path ที่แปลงแล้ว",
			Commands: []string{
				`# ผิด: ใช้ไฟล์ .ppk ของ PuTTY ตรง ๆ`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa.ppk"`,
				``,
				`# ถูก: แปลงเป็น OpenSSH format ก่อนด้วย PuTTYgen (Conversions > Export OpenSSH key) แล้วใช้ไฟล์ใหม่`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa_openssh.pem"`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    `Error: "ไม่รองรับ OS นี้ (รองรับเฉพาะ RHEL-family และ Debian-family)"`,
			Description: "เกิดเมื่อเครื่องปลายทางเป็น distro ที่ไม่ใช่ RHEL/CentOS/Rocky/AlmaLinux " +
				"หรือ Debian/Ubuntu (เช่น Alpine, openSUSE) ปัจจุบันโปรแกรมยังไม่รองรับ distro กลุ่มอื่น",
			Commands: []string{
				`# ตรวจสอบ distro เครื่องปลายทางก่อนด้วยมือผ่าน SSH`,
				`ssh root@10.0.0.10 "cat /etc/os-release"`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    "ติดตั้งล้มเหลวกลางคันแล้วอยากรันใหม่ตั้งแต่ต้น",
			Description: "ขั้นตอน cleanup จะลบของเก่าให้อัตโนมัติเมื่อรันใหม่ (จะถามยืนยันก่อนลบ " +
				"/var/lib/mysql อีกครั้ง) แค่รันคำสั่ง --apply เดิมซ้ำได้เลย ไม่ต้องแก้ไขอะไรเพิ่ม",
			Commands: []string{
				`# รันคำสั่งเดิมซ้ำได้เลย ขั้นตอน cleanup จะเคลียร์ของเก่าที่ค้างไว้ก่อนติดตั้งใหม่`,
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    "Node ในคลัสเตอร์หลุดออกไปแล้วต้องการเข้าร่วมคลัสเตอร์ใหม่ (ไม่ใช่ bootstrap ใหม่)",
			Description: "ห้ามรัน mariadb-installer ซ้ำกับ node ที่ออกจากคลัสเตอร์ไปแล้วในขณะที่ node อื่น " +
				"ยังรันอยู่ เพราะ pipeline cleanup จะลบข้อมูลเดิมและ bootstrap ใหม่ทับ ให้ SSH เข้าไปสั่ง " +
				"systemctl start mariadb ตรง ๆ บน node นั้นแทน (Galera จะ SST ข้อมูลจาก node ที่ยังรันอยู่ให้อัตโนมัติ)",
			Commands: []string{
				`# SSH ตรงไปยัง node ที่หลุด แล้ว start ใหม่ตรง ๆ (ไม่ใช่รัน mariadb-installer ซ้ำ)`,
				`ssh root@10.0.0.12 "systemctl start mariadb"`,
				``,
				`# ตรวจสถานะการเข้าร่วมคลัสเตอร์`,
				`ssh root@10.0.0.12 "mysql -e \"SHOW STATUS LIKE 'wsrep_cluster_size'\""`,
			},
		},
		{
			Category: CategoryTroubleshoot,
			Title:    "ต้องการดูเนื้อหาไฟล์ที่จะเขียนก่อนตัดสินใจ --apply",
			Description: "ใช้ --dry-run คู่กับ --verbose จะเห็นเนื้อหาไฟล์ /etc/my.cnf, sysctl.conf, " +
				"galera.cnf ที่จะเขียนจริงครบทุกไฟล์ โดยยังไม่เชื่อมต่อ SSH เลย",
			Commands: []string{
				`mariadb-installer.exe --dry-run --verbose --config=hosts-3node.json`,
			},
		},
	}
}
