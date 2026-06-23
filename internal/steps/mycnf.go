package steps

import (
	"fmt"

	"mariadb-installer/internal/runner"
	"mariadb-installer/internal/tuning"
)

// MyCnfOptions คือค่าที่ผู้ใช้กำหนดเองได้ (แทน hardcode เช่น charset, datadir)
type MyCnfOptions struct {
	Datadir        string
	Port           int
	CharacterSet   string // เช่น "utf8mb4" หรือ "tis620" ตามที่เห็นใน log ต้นฉบับ
	BindAddress    string
	ServerID       int
	MaxConnections int
}

// DefaultMyCnfOptions ค่าเริ่มต้นที่สมเหตุสมผลสำหรับ production (charset utf8mb4 แทน tis620
// ซึ่งเป็น legacy Thai charset ที่ไม่รองรับ Unicode/emoji เต็มรูปแบบ ผู้ใช้ปรับเป็น tis620 ได้ถ้าจำเป็นจริง ๆ)
func DefaultMyCnfOptions() MyCnfOptions {
	return MyCnfOptions{
		Datadir:        "/var/lib/mysql",
		Port:           3306,
		CharacterSet:   "utf8mb4",
		BindAddress:    "0.0.0.0",
		ServerID:       1,
		MaxConnections: 1000,
	}
}

// WriteMyCnf สร้าง /etc/my.cnf ทั้งไฟล์ในครั้งเดียว โดยแทนค่า buffer pool / key_buffer / max_heap_table
// ด้วยค่าที่คำนวณจาก RAM จริงของเครื่อง (tuning.Values) แทนการ sed แก้บรรทัดเฉพาะแบบ log ต้นฉบับ
// ซึ่งเสี่ยง sed ผิดบรรทัดถ้าโครงสร้างไฟล์เปลี่ยน
func WriteMyCnf(r *runner.Runner, t *tuning.Values, opt MyCnfOptions) error {
	r.SetStep("สร้าง /etc/my.cnf ตามค่า tuning ที่คำนวณได้")

	content := fmt.Sprintf(`[xtrabackup]
datadir=%[1]s

[client]
port = %[2]d
socket = %[1]s/mysql.sock
default-character-set = %[3]s

[mysqld]
large-pages
table_open_cache = 4000
table_definition_cache = 4000
port = %[2]d
datadir = %[1]s
tmpdir = /tmp
socket = %[1]s/mysql.sock
skip-external-locking
default_storage_engine = InnoDB

# --- ค่าที่คำนวณจาก RAM เครื่องจริง (%[4]d MB) ---
key_buffer_size = %[5]dM
max_heap_table_size = %[6]dM
innodb_buffer_pool_size = %[7]dM
# ------------------------------------------------

max_allowed_packet = 512M
sort_buffer_size = 1M
read_buffer_size = 1M
read_rnd_buffer_size = 1M
myisam_sort_buffer_size = 256M
thread_cache_size = 8
query_cache_size = 0
character-set-server = %[3]s
skip-name-resolve
innodb_file_per_table
skip-character-set-client-handshake
init_connect = 'SET NAMES %[3]s'
innodb_data_home_dir = %[1]s/
innodb_data_file_path = ibdata1:100M:autoextend
innodb_log_files_in_group = 2
innodb_log_group_home_dir = %[1]s/
innodb_read_io_threads = 16
innodb_write_io_threads = 16
innodb_stats_on_metadata = 0
innodb_thread_concurrency = 64
innodb_log_file_size = 4G
innodb_log_buffer_size = 32M
innodb_flush_log_at_trx_commit = 2
innodb_flush_log_at_timeout = 1800
innodb_lock_wait_timeout = 50
innodb_doublewrite = 1
innodb_open_files = 30000
slave_compressed_protocol = 1
net_write_timeout = 600
net_read_timeout = 600
connect_timeout = 60
wait_timeout = 300
sql-mode = "NO_ENGINE_SUBSTITUTION"
innodb_flush_method = O_DIRECT
join_buffer_size = 2M
concurrent_insert = 2
max_connections = %[8]d
open_files_limit = 100000
table-open-cache-instances = 32
server-id = %[9]d
log-bin = %[1]s/mysql-bin.log
#disable-log-bin
log_error = /var/log/mysqld.log
bind-address = %[10]s
log_bin_trust_function_creators = 1
expire_logs_days = 14
innodb_max_dirty_pages_pct = 90
innodb_max_dirty_pages_pct_lwm = 10
innodb_lru_scan_depth = 10K
innodb_page_cleaners = 24
innodb_use_native_aio = 1
innodb_stats_persistent = 1
innodb_adaptive_flushing = 1
innodb_flush_neighbors = 0
innodb_purge_threads = 4
innodb_adaptive_hash_index = 0
max_prepared_stmt_count = 1000000
innodb_max_purge_lag_delay = 300000
innodb_max_purge_lag = 10M

[mysqldump]
quick
max_allowed_packet = 512M

[mysql]
no-auto-rehash
default-character-set = %[3]s

[mysqlhotcopy]
interactive-timeout
`,
		opt.Datadir, opt.Port, opt.CharacterSet,
		t.MemTotalKB/1024,
		t.KeyBufferMB, t.MaxHeapTableMB, t.InnodbBufferPoolMB,
		opt.MaxConnections, opt.ServerID, opt.BindAddress,
	)

	return r.WriteFile("/etc/my.cnf", content, 0644)
}
