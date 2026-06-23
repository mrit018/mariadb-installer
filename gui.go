package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var guiMu sync.Mutex

type guiForm struct {
	ProfileName    string
	Mode           string
	Action         string
	Host           string
	SSHPort        int
	User           string
	Auth           string
	KeyPath        string
	Password       string
	SudoPassword   string
	Charset        string
	Verbose        bool
	SkipCleanup    bool
	ConfirmCleanup bool
	ConfigJSON     string
}

type guiPageData struct {
	Form     guiForm
	Profiles []string
	Output   string
	Error    string
}

var guiTemplate = template.Must(template.New("gui").Parse(`<!doctype html>
<html lang="th">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>mariadb-installer GUI</title>
<style>
body { font-family: Segoe UI, Arial, sans-serif; margin: 0; background: #0f172a; color: #e2e8f0; }
.wrap { max-width: 1180px; margin: 0 auto; padding: 24px; }
.hero { padding: 20px 24px; background: linear-gradient(135deg, #111827, #1e293b); border: 1px solid #334155; border-radius: 18px; box-shadow: 0 20px 50px rgba(0,0,0,.25); }
.hero h1 { margin: 0 0 8px; font-size: 28px; }
.hero p { margin: 0; color: #94a3b8; }
.grid { display: grid; grid-template-columns: 1.1fr .9fr; gap: 18px; margin-top: 18px; }
.card { background: #111827; border: 1px solid #334155; border-radius: 18px; padding: 18px; }
label { display: block; font-size: 13px; color: #cbd5e1; margin: 10px 0 6px; }
input[type=text], input[type=password], input[type=number], textarea, select {
  width: 100%; box-sizing: border-box; border-radius: 12px; border: 1px solid #334155;
  background: #0b1220; color: #e2e8f0; padding: 10px 12px; outline: none;
}
textarea { min-height: 260px; resize: vertical; font-family: Consolas, monospace; font-size: 13px; }
.row { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.checks { display: flex; flex-wrap: wrap; gap: 16px; margin-top: 12px; }
.checks label { display: inline-flex; align-items: center; gap: 8px; margin: 0; }
.buttons { display: flex; gap: 10px; margin-top: 16px; }
button { border: 0; border-radius: 999px; padding: 11px 16px; font-weight: 600; cursor: pointer; }
.primary { background: #38bdf8; color: #00111f; }
.secondary { background: #1f2937; color: #e2e8f0; border: 1px solid #334155; }
.result, .error { white-space: pre-wrap; font-family: Consolas, monospace; font-size: 13px; line-height: 1.5; }
.result { background: #020617; border: 1px solid #334155; padding: 14px; border-radius: 14px; }
.error { background: #3f1d1d; border: 1px solid #7f1d1d; padding: 14px; border-radius: 14px; color: #fecaca; }
.hint { color: #94a3b8; font-size: 13px; margin-top: 8px; }
.full { grid-column: 1 / -1; }
@media (max-width: 980px) { .grid { grid-template-columns: 1fr; } .row { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div class="wrap">
  <div class="hero">
    <h1>mariadb-installer GUI</h1>
    <p>กรอก host, user, password, key และ sudo ได้จากหน้าเดียว ใช้ pipeline เดิมจาก CLI</p>
  </div>

  <div class="grid">
    <div class="card">
      <form method="post" action="/action">
        <label>Profile</label>
        <div class="row">
          <div>
            <input type="text" name="profile_name" value="{{.Form.ProfileName}}" placeholder="ชื่อ profile เช่น prod-db1">
          </div>
          <div>
            <select name="profile_select">
              <option value="">เลือก profile ที่บันทึกไว้</option>
              {{range .Profiles}}
              <option value="{{.}}" {{if eq . $.Form.ProfileName}}selected{{end}}>{{.}}</option>
              {{end}}
            </select>
          </div>
        </div>

        <div class="buttons">
          <button class="secondary" type="submit" name="submit" value="load">Load Profile</button>
          <button class="secondary" type="submit" name="submit" value="save">Save Profile</button>
          <button class="secondary" type="submit" name="submit" value="delete">Delete Profile</button>
        </div>

        <label>โหมด</label>
        <select name="mode">
          <option value="single" {{if eq .Form.Mode "single"}}selected{{end}}>เครื่องเดียว</option>
          <option value="config" {{if eq .Form.Mode "config"}}selected{{end}}>Config JSON / หลายเครื่อง</option>
        </select>

        <div class="row">
          <div>
            <label>คำสั่ง</label>
            <select name="action">
              <option value="dry-run" {{if eq .Form.Action "dry-run"}}selected{{end}}>Dry-run</option>
              <option value="apply" {{if eq .Form.Action "apply"}}selected{{end}}>Apply</option>
            </select>
          </div>
          <div>
            <label>Charset</label>
            <input type="text" name="charset" value="{{.Form.Charset}}" placeholder="utf8mb4">
          </div>
        </div>

        <div class="row">
          <div>
            <label>Host</label>
            <input type="text" name="host" value="{{.Form.Host}}" placeholder="209.15.116.130">
          </div>
          <div>
            <label>SSH Port</label>
            <input type="number" name="ssh_port" value="{{.Form.SSHPort}}">
          </div>
        </div>

        <div class="row">
          <div>
            <label>User</label>
            <input type="text" name="user" value="{{.Form.User}}" placeholder="root หรือ user อื่น">
          </div>
          <div>
            <label>Auth</label>
            <select name="auth">
              <option value="password" {{if eq .Form.Auth "password"}}selected{{end}}>Password</option>
              <option value="key" {{if eq .Form.Auth "key"}}selected{{end}}>Private key</option>
            </select>
          </div>
        </div>

        <div class="row">
          <div>
            <label>SSH Password</label>
            <input type="password" name="password" value="" placeholder="รหัสผ่าน SSH">
          </div>
          <div>
            <label>Key Path</label>
            <input type="text" name="key_path" value="{{.Form.KeyPath}}" placeholder="C:\Users\you\.ssh\id_rsa">
          </div>
        </div>

        <div class="row">
          <div>
            <label>Sudo Password</label>
            <input type="password" name="sudo_password" value="" placeholder="รหัสผ่าน sudo">
          </div>
          <div>
            <label>Config JSON File Path</label>
            <input type="text" name="config_path" value="" placeholder="หรือปล่อยว่างแล้วใช้ textarea ข้างล่าง">
          </div>
        </div>

        <div class="checks">
          <label><input type="checkbox" name="verbose" {{if .Form.Verbose}}checked{{end}}> verbose</label>
          <label><input type="checkbox" name="skip_cleanup" {{if .Form.SkipCleanup}}checked{{end}}> skip cleanup</label>
          <label><input type="checkbox" name="confirm_cleanup" {{if .Form.ConfirmCleanup}}checked{{end}}> ยืนยันให้ลบข้อมูลเดิม</label>
        </div>

        <label>Config JSON</label>
        <textarea name="config_json" placeholder="{ ... }">{{.Form.ConfigJSON}}</textarea>
        <div class="hint">ถ้าใส่ Config JSON ระบบจะใช้โหมดหลายเครื่องแทนฟอร์ม single host</div>

        <div class="buttons">
          <button class="primary" type="submit" name="submit" value="run">Run</button>
          <button class="secondary" type="reset">Reset</button>
        </div>
      </form>
    </div>

    <div class="card">
      <label>ผลลัพธ์</label>
      {{if .Error}}
      <div class="error">{{.Error}}</div>
      {{end}}
      {{if .Output}}
      <div class="result">{{.Output}}</div>
      {{else}}
      <div class="hint">ผลลัพธ์จะขึ้นที่นี่หลังจากกด Run</div>
      {{end}}
      <div class="hint" style="margin-top:14px;">
        คำสั่งจะรันบนโปรแกรมนี้เหมือน CLI ปกติ แต่ผ่านฟอร์มแทนการพิมพ์ flag
      </div>
    </div>
  </div>
</div>
</body>
</html>`))

func startGUI(port int) error {
	if port == 0 {
		port = 8079
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("เปิด GUI port %s ไม่สำเร็จ: %w", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderGUI(w, guiPageData{Form: defaultGUIForm(), Profiles: mustProfileNames()})
	})
	mux.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		data, err := handleGUIAction(r)
		if err != nil {
			data.Error = err.Error()
		}
		renderGUI(w, data)
	})

	server := &http.Server{Handler: mux}
	url := "http://" + addr
	fmt.Printf("เปิด GUI ที่ %s\n", url)
	go openBrowser(url)
	return server.Serve(ln)
}

func mustProfileNames() []string {
	names, err := profileNames()
	if err != nil {
		return nil
	}
	return names
}

func defaultGUIForm() guiForm {
	return guiForm{
		Mode:        "single",
		Action:      "dry-run",
		SSHPort:     22,
		User:        "root",
		Auth:        "password",
		Charset:     "utf8mb4",
		SkipCleanup: true,
	}
}

func handleGUIAction(r *http.Request) (guiPageData, error) {
	if err := r.ParseForm(); err != nil {
		return guiPageData{}, err
	}

	form := guiForm{
		ProfileName:    strings.TrimSpace(r.FormValue("profile_name")),
		Mode:           strings.TrimSpace(r.FormValue("mode")),
		Action:         strings.TrimSpace(r.FormValue("action")),
		Host:           strings.TrimSpace(r.FormValue("host")),
		User:           strings.TrimSpace(r.FormValue("user")),
		Auth:           strings.TrimSpace(r.FormValue("auth")),
		KeyPath:        strings.TrimSpace(r.FormValue("key_path")),
		Charset:        strings.TrimSpace(r.FormValue("charset")),
		ConfigJSON:     r.FormValue("config_json"),
		Verbose:        r.FormValue("verbose") != "",
		SkipCleanup:    r.FormValue("skip_cleanup") != "",
		ConfirmCleanup: r.FormValue("confirm_cleanup") != "",
	}

	if v := strings.TrimSpace(r.FormValue("ssh_port")); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return guiPageData{Form: form}, fmt.Errorf("SSH Port ไม่ถูกต้อง: %w", err)
		}
		form.SSHPort = port
	} else {
		form.SSHPort = 22
	}

	if form.Action == "" {
		form.Action = "dry-run"
	}
	if form.Mode == "" {
		form.Mode = "single"
	}
	if form.Auth == "" {
		form.Auth = "password"
	}
	if form.Charset == "" {
		form.Charset = "utf8mb4"
	}
	if form.ProfileName == "" {
		form.ProfileName = strings.TrimSpace(r.FormValue("profile_select"))
	}
	submit := strings.TrimSpace(r.FormValue("submit"))
	switch submit {
	case "load":
		return loadGUIProfileAction(form, r)
	case "save":
		if err := saveProfile(form.ProfileName, form); err != nil {
			return guiPageData{Form: form, Profiles: mustProfileNames()}, err
		}
		return guiPageData{Form: form, Profiles: mustProfileNames(), Output: fmt.Sprintf("saved profile %q", form.ProfileName)}, nil
	case "delete":
		if err := deleteProfile(form.ProfileName); err != nil {
			return guiPageData{Form: form, Profiles: mustProfileNames()}, err
		}
		return guiPageData{Form: defaultGUIForm(), Profiles: mustProfileNames(), Output: fmt.Sprintf("deleted profile %q", form.ProfileName)}, nil
	case "run":
		fallthrough
	default:
		if form.Mode == "config" && strings.TrimSpace(form.ConfigJSON) == "" && strings.TrimSpace(r.FormValue("config_path")) == "" {
			return guiPageData{Form: form, Profiles: mustProfileNames()}, fmt.Errorf("โหมด config ต้องใส่ Config JSON หรือ config_path อย่างใดอย่างหนึ่ง")
		}

		if !form.SkipCleanup && !form.ConfirmCleanup {
			return guiPageData{Form: form, Profiles: mustProfileNames()}, fmt.Errorf("ถ้าไม่เลือก skip cleanup ต้องติ๊ก 'ยืนยันให้ลบข้อมูลเดิม' ด้วย")
		}

		opts := appOptions{
			DryRun:             form.Action != "apply",
			Apply:              form.Action == "apply",
			Verbose:            form.Verbose,
			Charset:            form.Charset,
			SkipCleanup:        form.SkipCleanup,
			AutoConfirmCleanup: form.ConfirmCleanup,
			SSHPort:            form.SSHPort,
			User:               form.User,
			KeyPath:            form.KeyPath,
			Password:           r.FormValue("password"),
			SudoPassword:       r.FormValue("sudo_password"),
		}

		configPath := strings.TrimSpace(r.FormValue("config_path"))
		tempFile := ""
		if strings.TrimSpace(form.ConfigJSON) != "" {
			f, err := os.CreateTemp("", "mariadb-installer-gui-*.json")
			if err != nil {
				return guiPageData{Form: form, Profiles: mustProfileNames()}, err
			}
			tempFile = f.Name()
			if _, err := io.WriteString(f, form.ConfigJSON); err != nil {
				f.Close()
				os.Remove(tempFile)
				return guiPageData{Form: form, Profiles: mustProfileNames()}, err
			}
			if err := f.Close(); err != nil {
				os.Remove(tempFile)
				return guiPageData{Form: form, Profiles: mustProfileNames()}, err
			}
			configPath = tempFile
		}
		defer func() {
			if tempFile != "" {
				_ = os.Remove(tempFile)
			}
		}()

		if configPath != "" {
			opts.ConfigPath = configPath
		} else {
			opts.Host = form.Host
		}

		output, err := captureOutput(func() error {
			return runApp(opts)
		})
		result := guiPageData{Form: form, Profiles: mustProfileNames(), Output: output}
		if err != nil {
			result.Error = err.Error()
		}
		return result, nil
	}
}

func loadGUIProfileAction(form guiForm, r *http.Request) (guiPageData, error) {
	name := strings.TrimSpace(r.FormValue("profile_select"))
	if name == "" {
		name = form.ProfileName
	}
	loaded, err := loadProfile(name)
	if err != nil {
		return guiPageData{Form: form, Profiles: mustProfileNames()}, err
	}
	loaded.ProfileName = name
	return guiPageData{Form: loaded, Profiles: mustProfileNames(), Output: fmt.Sprintf("loaded profile %q", name)}, nil
}

func renderGUI(w http.ResponseWriter, data guiPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = guiTemplate.Execute(w, data)
}

func captureOutput(fn func() error) (string, error) {
	guiMu.Lock()
	defer guiMu.Unlock()

	oldOut := os.Stdout
	oldErr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		return "", err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return "", err
	}

	var buf bytes.Buffer
	done := make(chan struct{}, 2)
	copyPipe := func(r *os.File) {
		_, _ = io.Copy(&buf, r)
		done <- struct{}{}
	}
	go copyPipe(outR)
	go copyPipe(errR)

	os.Stdout = outW
	os.Stderr = errW

	runErr := fn()

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	<-done
	<-done
	_ = outR.Close()
	_ = errR.Close()

	return buf.String(), runErr
}

func openBrowser(url string) {
	// small delay helps on slower machines so the browser doesn't race the listener start
	time.Sleep(500 * time.Millisecond)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
