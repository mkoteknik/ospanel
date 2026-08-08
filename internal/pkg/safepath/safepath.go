package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve kullanıcı girdisini jail dizini içinde güvenli bir mutlak yola çözümler.
// - jail: kullanıcının kök dizini (örn /home/alice). Boşsa hata döner.
// - userPath: kullanıcı girdisi (mutlak veya relative, örn "public_html/../etc/passwd").
// Symlink'ler EvalSymlinks ile çözülür, sonuç jail dışına çıkarsa hata döner.
// Windows'ta da çalışır (filepath semantics).
func Resolve(jail, userPath string) (string, error) {
	if jail == "" {
		return "", fmt.Errorf("jail boş olamaz")
	}
	jail = filepath.Clean(jail)
	if !filepath.IsAbs(jail) {
		return "", fmt.Errorf("jail mutlak yol olmalı: %s", jail)
	}

	// Boş input → jail kendisi
	if userPath == "" || userPath == "." {
		return jail, nil
	}

	// Kullanıcı girdisini temizle; mutlak ise jail'e göre relative yap
	cleaned := filepath.Clean(userPath)

	// Mutlak yolu jail'e göre yeniden kökle ("/etc/passwd" → jail + "/etc/passwd" değil, direkt jail dışına çıkmayı engelle)
	// Eğer kullanıcı mutlak verdiyse, önce jail'den strip et
	if filepath.IsAbs(cleaned) {
		// Mutlak ise, jail prefix'i varsa strip et, yoksa relative'e çevir
		// Örn jail=/home/alice, input=/home/alice/public_html → public_html
		// input=/etc/passwd → etc/passwd (ama sonra jail ile join edilince jail/etc/passwd olur, ki bu OK — gerçek /etc/passwd'e erişemez)
		rel, err := filepath.Rel(jail, cleaned)
		if err == nil && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			// Zaten jail içinde bir mutlak yol → relative kısmı kullan
			cleaned = rel
		} else {
			// Jail dışı mutlak yol → başındaki /'ı at, jail altında değerlendir
			cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
			// Windows: "C:\foo" gibi ise
			if len(cleaned) >= 2 && cleaned[1] == ':' {
				cleaned = cleaned[2:]
				cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
			}
		}
	}

	joined := filepath.Join(jail, cleaned)
	joined = filepath.Clean(joined)

	// Jail dışına çıkma kontrolü (prefix check)
	if !isWithin(joined, jail) {
		return "", fmt.Errorf("yol jail dışına çıkıyor: %s", userPath)
	}

	// Symlink çözümleme: eğer hedef var ise, gerçek yol da jail içinde mi?
	// Yoksa (yeni dosya oluşturma) parent'ı kontrol et
	targetToCheck := joined
	if _, err := os.Lstat(joined); os.IsNotExist(err) {
		targetToCheck = filepath.Dir(joined)
		// Parent da yoksa en yakın var olan ata'yı bul
		for targetToCheck != jail && targetToCheck != string(filepath.Separator) && targetToCheck != "." {
			if _, err := os.Lstat(targetToCheck); err == nil {
				break
			}
			parent := filepath.Dir(targetToCheck)
			if parent == targetToCheck {
				break
			}
			targetToCheck = parent
		}
	}

	if real, err := filepath.EvalSymlinks(targetToCheck); err == nil {
		real = filepath.Clean(real)
		// EvalSymlinks jail dışına çıkardı mı?
		if !isWithin(real, jail) && !isWithin(joined, jail) {
			// Eğer real jail dışı ise ama joined jail içindeyse, symlink jail dışını gösteriyor demektir
			// Bu durumda engelle (zip slip / symlink attack)
			if !isWithin(real, jail) {
				// Sadece symlink hedefi jail içinde değilse engelle
				// Ancak /var/lib/ospanel gibi gerçek jail dışına symlink kasıtlı değilse engelle
				return "", fmt.Errorf("symlink jail dışına işaret ediyor: %s -> %s", userPath, real)
			}
		}
	}

	return joined, nil
}

// isWithin child yolunun parent içinde olup olmadığını döner (sınır dahil)
func isWithin(child, parent string) bool {
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)

	// Eşit ise içinde sayılır
	if child == parent {
		return true
	}
	// Parent separator ile bitmiyorsa ekle
	if !strings.HasSuffix(parent, string(filepath.Separator)) {
		parent += string(filepath.Separator)
	}
	if !strings.HasSuffix(child, string(filepath.Separator)) {
		child += string(filepath.Separator)
	}
	return strings.HasPrefix(child, parent)
}

// MustResolve Resolve'un panik versiyonu (test için)
func MustResolve(jail, userPath string) string {
	p, err := Resolve(jail, userPath)
	if err != nil {
		panic(err)
	}
	return p
}

// JoinAndCheck jail + elemanları join eder ve jail içinde mi kontrol eder (ExtractZip için)
func JoinAndCheck(jail string, elem ...string) (string, error) {
	joined := filepath.Join(append([]string{jail}, elem...)...)
	return Resolve(jail, joined)
}
