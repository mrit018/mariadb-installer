package examples

// singleHostExamples รวมตัวอย่างการสั่งงานโหมดเครื่องเดียว (ผ่าน flag --host)
// ครอบคลุม auth ทั้งสองแบบ, การปรับ port/charset, และตัวเลือกข้ามขั้นตอน
func singleHostExamples() []Example {
	return []Example{
		{
			Category: CategorySingleHost,
			Title:    "ดูแผนงานก่อน (dry-run) ด้วย private key",
			Description: "ใช้เช็คก่อนทุกครั้งว่า pipeline จะรันอะไรบ้าง โดยยังไม่เชื่อมต่อ SSH จริง " +
				"และไม่แก้ไขเครื่องปลายทาง",
			Commands: []string{
				`mariadb-installer.exe --dry-run --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"`,
			},
		},
		{
			Category:    CategorySingleHost,
			Title:       "ติดตั้งจริงด้วย private key",
			Description: "กรณีปกติที่สุด เครื่องปลายทางตั้ง SSH key auth ไว้แล้ว",
			Commands: []string{
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"`,
			},
		},
		{
			Category:    CategorySingleHost,
			Title:       "ติดตั้งจริงด้วย password แทน key",
			Description: "ใช้เมื่อเครื่องปลายทางยังไม่ได้ตั้ง SSH key หรือเป็น VM ใหม่ที่ยังไม่ copy key ขึ้นไป",
			Commands: []string{
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --password=ตัวอย่างรหัสผ่าน`,
			},
		},
		{
			Category: CategorySingleHost,
			Title:    "เครื่องปลายทางใช้ SSH port ไม่ใช่ 22",
			Description: "บางเครื่องเปลี่ยน SSH port เพื่อความปลอดภัย (เช่น cloud VM ที่ตั้ง custom port) " +
				"ใส่ --ssh-port ให้ตรงกับที่เครื่องปลายทางเปิดไว้จริง",
			Commands: []string{
				`mariadb-installer.exe --apply --host=10.0.0.10 --ssh-port=2222 --user=root --key="C:\keys\id_rsa"`,
			},
		},
		{
			Category: CategorySingleHost,
			Title:    "ติดตั้งบนเครื่องที่ยังไม่เคยมี MySQL/MariaDB มาก่อน (ข้าม cleanup)",
			Description: "ใส่ --skip-cleanup เพื่อข้ามขั้นตอนลบของเก่า/ถามยืนยันลบข้อมูล " +
				"ใช้กับเครื่องใหม่ที่มั่นใจว่าไม่มี MySQL/MariaDB เดิมอยู่แล้วเท่านั้น",
			Commands: []string{
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa" --skip-cleanup`,
			},
		},
		{
			Category: CategorySingleHost,
			Title:    "กำหนด character set เอง (เช่นต้องใช้ tis620 กับระบบเก่า)",
			Description: "ค่า default คือ utf8mb4 (รองรับ Unicode เต็มรูปแบบ) เปลี่ยนเป็น tis620 ได้ " +
				"ถ้าระบบเดิมผูกกับ legacy Thai charset นี้อยู่",
			Commands: []string{
				`mariadb-installer.exe --apply --host=10.0.0.10 --user=root --key="C:\keys\id_rsa" --charset=tis620`,
			},
		},
		{
			Category:    CategorySingleHost,
			Title:       "ดู log ทุกคำสั่งแบบละเอียด (verbose) รวมเนื้อหาไฟล์ที่จะเขียน",
			Description: "มีประโยชน์เวลา dry-run แล้วต้องการตรวจเนื้อหา /etc/my.cnf หรือ sysctl ก่อนรันจริง",
			Commands: []string{
				`mariadb-installer.exe --dry-run --verbose --host=10.0.0.10 --user=root --key="C:\keys\id_rsa"`,
			},
		},
	}
}
