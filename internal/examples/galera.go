package examples

// galeraExamples รวมตัวอย่างการตั้ง Galera cluster หลายขนาด ผ่านไฟล์ config JSON
// แต่ละตัวอย่างแสดงทั้งขั้นตอนสร้างไฟล์ config และคำสั่งรันจริง
func galeraExamples() []Example {
	return []Example{
		{
			Category: CategoryGalera,
			Title:    "Galera cluster 2 node (ขั้นต่ำสำหรับทดสอบ ไม่แนะนำ production)",
			Description: "2 node ใช้ได้สำหรับทดสอบ แต่ production ควรมีอย่างน้อย 3 node " +
				"เพื่อให้ cluster โหวต quorum ได้เมื่อ node หนึ่งล่ม (2 node ถ้า node หนึ่งหลุด อีก node จะ non-primary)",
			Commands: []string{
				`# 1) สร้างไฟล์ hosts-2node.json`,
				`{
  "cluster_name": "dev-galera",
  "user": "root",
  "auth": "key",
  "key_path": "C:\\keys\\id_rsa",
  "hosts": [
    { "name": "db1", "address": "10.0.0.11" },
    { "name": "db2", "address": "10.0.0.12" }
  ]
}`,
				`# 2) ดูแผนงานก่อน`,
				`mariadb-installer.exe --dry-run --config=hosts-2node.json`,
				`# 3) ติดตั้งจริง`,
				`mariadb-installer.exe --apply --config=hosts-2node.json`,
			},
		},
		{
			Category: CategoryGalera,
			Title:    "Galera cluster 3 node (ค่ามาตรฐานสำหรับ production)",
			Description: "3 node คือขนาดที่แนะนำสำหรับ production ทั่วไป เพราะ quorum ยังโหวตได้ " +
				"แม้ node หนึ่งล่ม (2 จาก 3 ยัง majority)",
			Commands: []string{
				`# 1) สร้างไฟล์ hosts-3node.json`,
				`{
  "cluster_name": "prod-galera",
  "user": "root",
  "auth": "key",
  "key_path": "C:\\keys\\id_rsa",
  "hosts": [
    { "name": "db1", "address": "10.0.0.11" },
    { "name": "db2", "address": "10.0.0.12" },
    { "name": "db3", "address": "10.0.0.13" }
  ]
}`,
				`# 2) ติดตั้งจริง (โปรแกรมจะ bootstrap db1 ก่อน แล้วให้ db2/db3 เข้าร่วมตามลำดับ)`,
				`mariadb-installer.exe --apply --config=hosts-3node.json`,
			},
		},
		{
			Category: CategoryGalera,
			Title:    "Galera cluster 5 node (รองรับ workload สูง/ทนล่มได้มากขึ้น)",
			Description: "5 node ใช้เมื่อต้องการกระจาย read load มากขึ้นหรือทนต่อการล่มพร้อมกันได้สูงสุด 2 node " +
				"โดย quorum ยังทำงานได้ (3 จาก 5 ยัง majority) ใช้ auth แบบ password ในตัวอย่างนี้",
			Commands: []string{
				`# 1) สร้างไฟล์ hosts-5node.json`,
				`{
  "cluster_name": "prod-galera-ha",
  "user": "root",
  "auth": "password",
  "password": "ตัวอย่างรหัสผ่าน",
  "hosts": [
    { "name": "db1", "address": "10.0.0.11" },
    { "name": "db2", "address": "10.0.0.12" },
    { "name": "db3", "address": "10.0.0.13" },
    { "name": "db4", "address": "10.0.0.14" },
    { "name": "db5", "address": "10.0.0.15" }
  ]
}`,
				`# 2) ติดตั้งจริง (ติดตั้งทีละเครื่องตามลำดับในลิสต์ ไม่พร้อมกัน)`,
				`mariadb-installer.exe --apply --config=hosts-5node.json`,
			},
		},
		{
			Category:    CategoryGalera,
			Title:       "เครื่องในคลัสเตอร์ใช้ SSH port ไม่เท่ากัน",
			Description: "ใส่ \"port\" แยกต่อ host ในไฟล์ config ได้ ถ้าไม่ระบุจะใช้ port 22 เป็นค่า default",
			Commands: []string{
				`{
  "cluster_name": "prod-galera",
  "user": "root",
  "auth": "key",
  "key_path": "C:\\keys\\id_rsa",
  "hosts": [
    { "name": "db1", "address": "10.0.0.11", "port": 2201 },
    { "name": "db2", "address": "10.0.0.12", "port": 2202 },
    { "name": "db3", "address": "10.0.0.13", "port": 2203 }
  ]
}`,
				`mariadb-installer.exe --apply --config=hosts-3node.json`,
			},
		},
		{
			Category: CategoryGalera,
			Title:    "ติดตั้งหลายเครื่องแบบ \"แยกอิสระ\" (ไม่ตั้ง Galera cluster)",
			Description: "ถ้าไม่ระบุ \"cluster_name\" (หรือลบบรรทัดนี้ทิ้ง) โปรแกรมจะติดตั้ง MariaDB " +
				"แยกอิสระแต่ละเครื่อง ไม่ตั้งค่า wsrep_cluster_address ใด ๆ ให้ ใช้เมื่อแค่ต้องการติดตั้ง " +
				"หลายเครื่องพร้อมกันโดยไม่เกี่ยวข้องกันเป็นคลัสเตอร์",
			Commands: []string{
				`{
  "user": "root",
  "auth": "key",
  "key_path": "C:\\keys\\id_rsa",
  "hosts": [
    { "name": "db-report-1", "address": "10.0.0.21" },
    { "name": "db-report-2", "address": "10.0.0.22" }
  ]
}`,
				`mariadb-installer.exe --apply --config=hosts-standalone.json`,
			},
		},
	}
}
