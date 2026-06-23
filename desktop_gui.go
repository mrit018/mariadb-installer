package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func startDesktopGUI(installerPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("native desktop GUI รองรับเฉพาะ Windows")
	}
	if installerPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("หา path ของ executable ไม่สำเร็จ: %w", err)
		}
		installerPath = exe
	}
	scriptPath, err := writeDesktopGUIScript(installerPath)
	if err != nil {
		return err
	}
	defer os.Remove(scriptPath)

	ps := firstExisting("powershell.exe", "pwsh.exe")
	if ps == "" {
		return fmt.Errorf("ไม่พบ PowerShell บนเครื่องนี้")
	}
	cmd := exec.Command(ps, "-NoProfile", "-ExecutionPolicy", "Bypass", "-STA", "-File", scriptPath, "-InstallerPath", installerPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func firstExisting(names ...string) string {
	for _, n := range names {
		if _, err := exec.LookPath(n); err == nil {
			return n
		}
	}
	return ""
}

func writeDesktopGUIScript(installerPath string) (string, error) {
	script := strings.ReplaceAll(desktopGUIScript, "__INSTALLER_PATH__", escapePSLiteral(installerPath))
	f, err := os.CreateTemp("", "mariadb-installer-desktop-gui-*.ps1")
	if err != nil {
		return "", fmt.Errorf("สร้าง temp script GUI ไม่สำเร็จ: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("เขียน script GUI ไม่สำเร็จ: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("ปิด script GUI ไม่สำเร็จ: %w", err)
	}
	return path, nil
}

func escapePSLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

const desktopGUIScript = `
param(
  [string]$InstallerPath = '__INSTALLER_PATH__'
)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$profilesPath = Join-Path $env:APPDATA 'mariadb-installer\profiles.json'

function Get-Store {
  if (Test-Path $profilesPath) {
    try { return (Get-Content $profilesPath -Raw | ConvertFrom-Json) } catch { return [pscustomobject]@{} }
  }
  return [pscustomobject]@{}
}

function Save-Store($store) {
  $dir = Split-Path $profilesPath
  if (!(Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
  $store | ConvertTo-Json -Depth 10 | Set-Content -Encoding UTF8 $profilesPath
}

function Get-ProfileNames {
  $store = Get-Store
  return @($store.PSObject.Properties.Name | Sort-Object)
}

function FormToProfile {
  return [ordered]@{
    ProfileName = $txtProfileName.Text
    Mode = $cmbMode.Text
    Action = $cmbAction.Text
    Host = $txtHost.Text
    SSHPort = [int]$numPort.Value
    User = $txtUser.Text
    Auth = $cmbAuth.Text
    KeyPath = $txtKeyPath.Text
    Charset = $txtCharset.Text
    ConfigJSON = $txtConfig.Text
    Verbose = $chkVerbose.Checked
    SkipCleanup = $chkSkipCleanup.Checked
    ConfirmCleanup = $chkConfirmCleanup.Checked
    Password = ''
    SudoPassword = ''
  }
}

function ProfileToForm($p) {
  if ($null -eq $p) { return }
  $txtProfileName.Text = $p.ProfileName
  $cmbMode.Text = $p.Mode
  $cmbAction.Text = $p.Action
  $txtHost.Text = $p.Host
  if ($p.SSHPort) { $numPort.Value = [int]$p.SSHPort }
  $txtUser.Text = $p.User
  $cmbAuth.Text = $p.Auth
  $txtKeyPath.Text = $p.KeyPath
  $txtCharset.Text = $p.Charset
  $txtConfig.Text = $p.ConfigJSON
  $chkVerbose.Checked = [bool]$p.Verbose
  $chkSkipCleanup.Checked = [bool]$p.SkipCleanup
  $chkConfirmCleanup.Checked = [bool]$p.ConfirmCleanup
}

$form = New-Object System.Windows.Forms.Form
$form.Text = 'mariadb-installer'
$form.Size = New-Object System.Drawing.Size(1240, 920)
$form.StartPosition = 'CenterScreen'
$form.Font = New-Object System.Drawing.Font('Segoe UI', 9)

$layout = New-Object System.Windows.Forms.TableLayoutPanel
$layout.Dock = 'Fill'
$layout.ColumnCount = 2
$layout.RowCount = 1
$layout.ColumnStyles.Add((New-Object System.Windows.Forms.ColumnStyle([System.Windows.Forms.SizeType]::Percent, 48)))
$layout.ColumnStyles.Add((New-Object System.Windows.Forms.ColumnStyle([System.Windows.Forms.SizeType]::Percent, 52)))
$form.Controls.Add($layout)

$left = New-Object System.Windows.Forms.Panel
$left.Dock = 'Fill'
$left.AutoScroll = $true
$layout.Controls.Add($left, 0, 0)

$right = New-Object System.Windows.Forms.Panel
$right.Dock = 'Fill'
$layout.Controls.Add($right, 1, 0)

$y = 12
function Add-Label($text, $x, $y) {
  $lbl = New-Object System.Windows.Forms.Label
  $lbl.Text = $text
  $lbl.Location = New-Object System.Drawing.Point($x, $y)
  $lbl.AutoSize = $true
  $left.Controls.Add($lbl)
  return $lbl
}
function Add-TextBox($x, $y, $w, $h = 28, $multiline = $false) {
  $tb = New-Object System.Windows.Forms.TextBox
  $tb.Location = New-Object System.Drawing.Point($x, $y)
  $tb.Size = New-Object System.Drawing.Size($w, $h)
  $tb.Multiline = $multiline
  if ($multiline) { $tb.ScrollBars = 'Both'; $tb.AcceptsReturn = $true; $tb.AcceptsTab = $true }
  $left.Controls.Add($tb)
  return $tb
}
function Add-Combo($x, $y, $w, $items) {
  $cb = New-Object System.Windows.Forms.ComboBox
  $cb.Location = New-Object System.Drawing.Point($x, $y)
  $cb.Size = New-Object System.Drawing.Size($w, 28)
  $cb.DropDownStyle = 'DropDownList'
  [void]$cb.Items.AddRange($items)
  $left.Controls.Add($cb)
  return $cb
}
function Add-Check($text, $x, $y) {
  $c = New-Object System.Windows.Forms.CheckBox
  $c.Text = $text
  $c.Location = New-Object System.Drawing.Point($x, $y)
  $c.AutoSize = $true
  $left.Controls.Add($c)
  return $c
}
function Add-Button($text, $x, $y, $w = 110) {
  $b = New-Object System.Windows.Forms.Button
  $b.Text = $text
  $b.Location = New-Object System.Drawing.Point($x, $y)
  $b.Size = New-Object System.Drawing.Size($w, 30)
  $left.Controls.Add($b)
  return $b
}

Add-Label 'Profile name' 12 $y
$txtProfileName = Add-TextBox 12 ($y+22) 260
$cmbProfiles = Add-Combo 286 ($y+22) 250 @()
$btnLoadProfile = Add-Button 'Load' 548 ($y+20) 80
$btnSaveProfile = Add-Button 'Save' 634 ($y+20) 80
$btnDeleteProfile = Add-Button 'Delete' 720 ($y+20) 80
$y += 68

Add-Label 'Mode' 12 $y
$cmbMode = Add-Combo 12 ($y+22) 160 @('single','config')
$cmbMode.SelectedIndex = 0
Add-Label 'Action' 190 $y
$cmbAction = Add-Combo 190 ($y+22) 160 @('dry-run','apply')
$cmbAction.SelectedIndex = 0
Add-Label 'Charset' 368 $y
$txtCharset = Add-TextBox 368 ($y+22) 120
$txtCharset.Text = 'utf8mb4'
$y += 68

Add-Label 'Host' 12 $y
$txtHost = Add-TextBox 12 ($y+22) 260
Add-Label 'SSH Port' 286 $y
$numPort = New-Object System.Windows.Forms.NumericUpDown
$numPort.Location = New-Object System.Drawing.Point(286, ($y+22))
$numPort.Width = 100
$numPort.Minimum = 1
$numPort.Maximum = 65535
$numPort.Value = 22
$left.Controls.Add($numPort)
Add-Label 'User' 404 $y
$txtUser = Add-TextBox 404 ($y+22) 180
$txtUser.Text = 'root'
$y += 68

Add-Label 'Auth' 12 $y
$cmbAuth = Add-Combo 12 ($y+22) 160 @('password','key')
$cmbAuth.SelectedIndex = 0
Add-Label 'SSH Password' 190 $y
$txtPassword = Add-TextBox 190 ($y+22) 210
$txtPassword.UseSystemPasswordChar = $true
Add-Label 'Key Path' 412 $y
$txtKeyPath = Add-TextBox 412 ($y+22) 220
$y += 68

Add-Label 'Sudo Password' 12 $y
$txtSudo = Add-TextBox 12 ($y+22) 210
$txtSudo.UseSystemPasswordChar = $true
Add-Label 'Config Path' 236 $y
$txtConfigPath = Add-TextBox 236 ($y+22) 190
Add-Label 'Config JSON' 440 $y
$y += 22
$txtConfig = Add-TextBox 440 ($y+22) 240 250 $true
$txtConfig.Anchor = 'Top,Bottom,Left,Right'
$y += 280

$chkVerbose = Add-Check 'Verbose' 12 $y
$chkSkipCleanup = Add-Check 'Skip cleanup' 112 $y
$chkConfirmCleanup = Add-Check 'Confirm cleanup' 240 $y
$chkConfirmCleanup.Checked = $true
$y += 38

$btnRunDry = Add-Button 'Run Dry-Run' 12 $y 110
$btnRunApply = Add-Button 'Run Apply' 130 $y 110
$btnRefresh = Add-Button 'Refresh Profiles' 248 $y 130
$y += 44

$output = New-Object System.Windows.Forms.TextBox
$output.Multiline = $true
$output.ScrollBars = 'Both'
$output.ReadOnly = $true
$output.Font = New-Object System.Drawing.Font('Consolas', 10)
$output.Dock = 'Fill'
$right.Controls.Add($output)

function Append-Output($text) {
  $output.AppendText($text + [Environment]::NewLine)
  $output.SelectionStart = $output.Text.Length
  $output.ScrollToCaret()
}

function Build-Args([string]$modeAction) {
  $args = New-Object System.Collections.Generic.List[string]
  if ($modeAction -eq 'dry-run') { [void]$args.Add('--dry-run') } else { [void]$args.Add('--apply') }
  if ($chkVerbose.Checked) { [void]$args.Add('--verbose') }
  if ($chkSkipCleanup.Checked) { [void]$args.Add('--skip-cleanup') }
  [void]$args.Add('--charset=' + $txtCharset.Text)
  if ($cmbMode.Text -eq 'config') {
    if ($txtConfig.Text.Trim() -eq '' -and $txtConfigPath.Text.Trim() -eq '') { throw 'ต้องใส่ Config JSON หรือ Config Path' }
    if ($txtConfig.Text.Trim() -ne '') {
      $tmp = [System.IO.Path]::GetTempFileName() + '.json'
      [System.IO.File]::WriteAllText($tmp, $txtConfig.Text, [System.Text.Encoding]::UTF8)
      $script:tmpConfig = $tmp
      [void]$args.Add('--config=' + $tmp)
    } else {
      [void]$args.Add('--config=' + $txtConfigPath.Text.Trim())
    }
  } else {
    if ($txtHost.Text.Trim() -eq '') { throw 'ต้องใส่ Host' }
    [void]$args.Add('--host=' + $txtHost.Text.Trim())
    [void]$args.Add('--ssh-port=' + $numPort.Value)
    [void]$args.Add('--user=' + $txtUser.Text.Trim())
    if ($cmbAuth.Text -eq 'key') {
      if ($txtKeyPath.Text.Trim() -eq '') { throw 'ต้องใส่ Key Path' }
      [void]$args.Add('--key=' + $txtKeyPath.Text.Trim())
    } else {
      if ($txtPassword.Text -eq '') { throw 'ต้องใส่ SSH Password' }
      [void]$args.Add('--password=' + $txtPassword.Text)
    }
    if ($txtSudo.Text -ne '') { [void]$args.Add('--sudo-password=' + $txtSudo.Text) }
  }
  return ,$args.ToArray()
}

function Run-Installer([string]$modeAction) {
  $script:tmpConfig = $null
  try {
    $args = Build-Args $modeAction
    Append-Output ('> ' + $InstallerPath + ' ' + ($args -join ' '))
    $result = & $InstallerPath @args 2>&1 | Out-String
    Append-Output $result.TrimEnd()
  } catch {
    Append-Output ('ERROR: ' + $_.Exception.Message)
  } finally {
    if ($script:tmpConfig -and (Test-Path $script:tmpConfig)) { Remove-Item $script:tmpConfig -Force -ErrorAction SilentlyContinue }
  }
}

function Update-ProfileList {
  $cmbProfiles.Items.Clear()
  foreach ($n in (Get-ProfileNames)) { [void]$cmbProfiles.Items.Add($n) }
}

$btnRefresh.Add_Click({ Update-ProfileList })
$btnLoadProfile.Add_Click({
  $name = $txtProfileName.Text.Trim()
  if ($cmbProfiles.Text.Trim() -ne '') { $name = $cmbProfiles.Text.Trim() }
  $store = Get-Store
  if (-not $store.$name) { [System.Windows.Forms.MessageBox]::Show("ไม่พบ profile $name"); return }
  ProfileToForm $store.$name
  $txtProfileName.Text = $name
})
$btnSaveProfile.Add_Click({
  $name = $txtProfileName.Text.Trim()
  if ($name -eq '') { [System.Windows.Forms.MessageBox]::Show('กรุณาใส่ชื่อ profile'); return }
  $store = Get-Store
  $store | Add-Member -NotePropertyName $name -NotePropertyValue (FormToProfile) -Force
  Save-Store $store
  Update-ProfileList
})
$btnDeleteProfile.Add_Click({
  $name = $txtProfileName.Text.Trim()
  if ($name -eq '') { $name = $cmbProfiles.Text.Trim() }
  if ($name -eq '') { [System.Windows.Forms.MessageBox]::Show('กรุณาเลือกหรือใส่ชื่อ profile'); return }
  $store = Get-Store
  $store.PSObject.Properties.Remove($name)
  Save-Store $store
  Update-ProfileList
})
$btnRunDry.Add_Click({ Run-Installer 'dry-run' })
$btnRunApply.Add_Click({ Run-Installer 'apply' })

Update-ProfileList
$form.Add_Shown({ $form.Activate() })
[void]$form.ShowDialog()
`
