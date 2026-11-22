package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// SessionTTL adalah masa berlaku token sesi.
//
// Dipilih panjang: pemakaian pribadi tidak membutuhkan rotasi refresh yang
// rumit, dan panjangnya justru mengurangi jumlah panggilan peremajaan. Sesi
// tetap bisa dicabut kapan saja lewat logout karena validasinya membaca tabel
// sessions, bukan sekadar klaim kedaluwarsa.
const SessionTTL = 30 * 24 * time.Hour

// sessionTokenBytes menghasilkan 256 bit entropi. Angka ini membuat token
// tidak bisa ditebak meski hash-nya bocor.
const sessionTokenBytes = 32

// NewSessionToken membuat token sesi baru beserta hash penyimpanannya.
//
// Hash memakai SHA-256, bukan Argon2id: token acak 256-bit sudah berentropi
// penuh sehingga tidak ada ruang brute-force, sedangkan verifikasi berjalan pada
// setiap request dan Argon2id memang dibuat lambat. Lihat 0003_auth.sql.
func NewSessionToken() (raw string, hash string, err error) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("membuat token sesi: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken menghitung digest yang disimpan untuk sebuah token mentah.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
