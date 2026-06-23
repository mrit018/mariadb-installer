package steps

import (
	"fmt"

	"mariadb-installer/internal/osdetect"
	"mariadb-installer/internal/runner"
)

// mariadbVersion คือเวอร์ชัน MariaDB ที่จะติดตั้ง (ตรงกับ "12.1" ที่เห็นใน log ต้นฉบับ)
const mariadbVersion = "12.1"

// AddMariaDBRepo เขียน repo file ของ mariadb.org ให้ตรงกับ OS family และ arch ของเครื่อง
// แทนการ hardcode baseurl ตายตัวแบบใน log ต้นฉบับ (ซึ่งพิมพ์ผิดระหว่าง rhel10 กับ rhel9 ในรอบต่าง ๆ)
func AddMariaDBRepo(r *runner.Runner, info *osdetect.Info) error {
	r.SetStep(fmt.Sprintf("เพิ่ม MariaDB %s repository (%s)", mariadbVersion, info.MariaDBRepoArch))

	switch info.Family {
	case osdetect.FamilyRHEL:
		content := fmt.Sprintf(`[mariadb]
name = MariaDB
baseurl = https://yum.mariadb.org/%s/%s
module_hotfixes=1
gpgkey=https://yum.mariadb.org/RPM-GPG-KEY-MariaDB
gpgcheck=1
`, mariadbVersion, info.MariaDBRepoArch)

		if err := r.WriteFile("/etc/yum.repos.d/mariadb.repo", content, 0644); err != nil {
			return err
		}
		r.RunIgnoreErr("dnf clean all")
		r.RunIgnoreErr("rm -fr /var/cache/dnf/*")
		_, err := r.Run("dnf makecache")
		return err

	case osdetect.FamilyDebian:
		// mariadb.org ให้ใช้ apt repo แบบ deb822 หรือ classic list ก็ได้ ที่นี่ใช้ classic list
		// เพราะรองรับ Ubuntu/Debian รุ่นเก่าได้กว้างกว่า
		codename, err := r.Run("source /etc/os-release && echo $VERSION_CODENAME")
		if err != nil || codename == "" {
			codename = "stable"
		}
		distro := "ubuntu"
		if info.Name == "Debian GNU/Linux" {
			distro = "debian"
		}

		r.RunIgnoreErr("apt-get -y install software-properties-common dirmngr apt-transport-https")
		_, err = r.Run("curl -fsSL https://mariadb.org/mariadb_release_signing_key.pgp | gpg --dearmor -o /etc/apt/keyrings/mariadb-keyring.gpg")
		if err != nil {
			return fmt.Errorf("ดาวน์โหลด GPG key ของ MariaDB ไม่สำเร็จ: %w", err)
		}

		content := fmt.Sprintf(
			"deb [signed-by=/etc/apt/keyrings/mariadb-keyring.gpg] https://deb.mariadb.org/%s/%s %s main\n",
			mariadbVersion, distro, codename,
		)
		if err := r.WriteFile("/etc/apt/sources.list.d/mariadb.list", content, 0644); err != nil {
			return err
		}
		_, err = r.Run("apt-get update")
		return err
	}
	return fmt.Errorf("ไม่รองรับ OS family: %s", info.Family)
}
