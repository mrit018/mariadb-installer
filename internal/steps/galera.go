package steps

import (
	"fmt"
	"strings"

	"mariadb-installer/internal/runner"
)

// GaleraOptions คือค่าที่ต้องใช้ตั้งค่า Galera cluster บน node หนึ่งตัว
type GaleraOptions struct {
	ClusterName  string   // ชื่อ cluster ต้องเหมือนกันทุก node
	NodeName     string   // ชื่อ node นี้ (ไม่ซ้ำกันระหว่าง node)
	NodeAddress  string   // IP ของ node นี้เอง
	AllAddresses []string // IP ของทุก node ในคลัสเตอร์ (รวมตัวเอง) ใช้สร้าง wsrep_cluster_address
}

// WriteGaleraConfig สร้าง /etc/my.cnf.d/galera.cnf บน remote host หนึ่งเครื่อง
// ไฟล์นี้แยกจาก /etc/my.cnf หลัก เพื่อให้แก้ค่าคลัสเตอร์ทีหลังได้ง่ายโดยไม่ต้องแตะ tuning หลัก
func WriteGaleraConfig(r *runner.Runner, opt GaleraOptions) error {
	r.SetStep("สร้าง Galera cluster config")

	clusterAddr := "gcomm://" + strings.Join(opt.AllAddresses, ",")

	content := fmt.Sprintf(`# สร้างโดย mariadb-installer สำหรับ Galera cluster: %[1]s
[mysqld]
wsrep_on = ON
wsrep_provider = /usr/lib64/galera-4/libgalera_smm.so
wsrep_cluster_name = "%[1]s"
wsrep_cluster_address = "%[2]s"
wsrep_node_name = "%[3]s"
wsrep_node_address = "%[4]s"
binlog_format = ROW
default_storage_engine = InnoDB
innodb_autoinc_lock_mode = 2
wsrep_sst_method = mariabackup
wsrep_sst_auth = root:
`, opt.ClusterName, clusterAddr, opt.NodeName, opt.NodeAddress)

	confDir := "/etc/my.cnf.d"
	r.RunIgnoreErr("mkdir -p " + confDir)
	return r.WriteFile(confDir+"/galera.cnf", content, 0644)
}

// BootstrapFirstNode เริ่ม cluster ครั้งแรกบน node แรก (ต้องรันก่อน node อื่นเข้าร่วม)
// node อื่น ๆ ใช้ systemctl start mariadb ปกติ (ไม่ต้อง --wsrep-new-cluster)
func BootstrapFirstNode(r *runner.Runner) error {
	r.SetStep("Bootstrap Galera cluster (node แรก)")
	_, err := r.Run("galera_new_cluster")
	if err != nil {
		return fmt.Errorf("bootstrap cluster บน %s ไม่สำเร็จ: %w", r.HostLabel, err)
	}
	return nil
}
