package jail

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SSHManager SFTP/SSH jail yonetimi
type SSHManager struct{}

// NewSSHManager yeni SSHManager
func NewSSHManager() *SSHManager { return &SSHManager{} }

// SetupChrootJail kullaniciyi SFTP-only chroot jail'e alir
func (m *SSHManager) SetupChrootJail(username, homeDir string) error {
	// 1. sftponly grubuna ekle
	exec.Command("groupadd", "-f", "sftponly").Run()
	exec.Command("usermod", "-aG", "sftponly", username).Run()
	exec.Command("usermod", "-s", "/usr/sbin/nologin", username).Run()

	// 2. Home dizin izinlerini duzelt (chroot icin root:root gerekir)
	exec.Command("chown", "root:root", homeDir).Run()
	exec.Command("chmod", "755", homeDir).Run()

	// 3. SSH konfigurasyonunu guncelle
	if err := m.ensureSSHConfig(); err != nil {
		return err
	}

	// 4. SSHD reload
	exec.Command("systemctl", "reload", "sshd").Run()
	return nil
}

// RemoveChrootJail kullaniciyi jail'den cikarir
func (m *SSHManager) RemoveChrootJail(username, homeDir string) error {
	// sftponly grubundan cikar
	exec.Command("gpasswd", "-d", username, "sftponly").Run()
	exec.Command("usermod", "-s", "/bin/bash", username).Run()

	// Home dizin sahipligini geri ver
	exec.Command("chown", username+":"+username, homeDir).Run()
	return nil
}

// IsJailed kullanicinin jail'de olup olmadigini kontrol eder
func (m *SSHManager) IsJailed(username string) bool {
	out, err := exec.Command("groups", username).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "sftponly")
}

// ensureSSHConfig sshd_config'e sftponly grubu icin chroot ekler
func (m *SSHManager) ensureSSHConfig() error {
	sshConfig := "/etc/ssh/sshd_config"
	data, err := os.ReadFile(sshConfig)
	if err != nil {
		return fmt.Errorf("sshd_config okunamadi: %w", err)
	}

	// Zaten ekli mi kontrol et
	if strings.Contains(string(data), "Match Group sftponly") {
		return nil
	}

	// Jail blogunu ekle
	jailBlock := `

# OpenSpeed Panel - SFTP Chroot Jail
Match Group sftponly
    ChrootDirectory /home/%u
    ForceCommand internal-sftp
    PasswordAuthentication yes
    PermitTunnel no
    AllowAgentForwarding no
    AllowTcpForwarding no
    X11Forwarding no
`

	f, err := os.OpenFile(sshConfig, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("sshd_config yazilamadi: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(jailBlock); err != nil {
		return err
	}

	// Konfigurasyonu test et
	if out, err := exec.Command("sshd", "-t").CombinedOutput(); err != nil {
		return fmt.Errorf("sshd_config hatasi: %s - %w", string(out), err)
	}

	return nil
}
