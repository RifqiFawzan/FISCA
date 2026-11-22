package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Parameter Argon2id mengikuti rekomendasi OWASP Password Storage Cheat Sheet:
// m=64 MiB, t=3, p=4. Nilai ini memang dibuat lambat untuk kata sandi — lihat
// catatan di migration 0003_auth.sql mengenai mengapa sesi tidak memakainya.
const (
	argonTime      uint32 = 3
	argonMemoryKiB uint32 = 64 * 1024
	argonThreads   uint8  = 4
	argonKeyLen    uint32 = 32
	argonSaltLen           = 16

	// Batas parameter yang dibaca dari hash tersimpan. Tanpa ini, nilai rusak di
	// database bisa membuat Argon2id mengalokasikan memori tak terbatas.
	maxMemoryKiB = 512 * 1024
	maxTime      = 10
	maxThreads   = 8
)

// Panjang kata sandi dihitung per rune agar teks non-ASCII (Indonesia) tidak
// dihitung per byte. Hint "Minimal 8 karakter" di /login memakai angka ini.
const (
	MinPasswordLength = 8
	MaxPasswordLength = 256
)

// ErrPasswordFormat menandakan hash tersimpan tidak bisa dibaca.
var ErrPasswordFormat = errors.New("format hash kata sandi tidak valid")

// ValidatePassword menolak kata sandi yang terlalu pendek atau terlalu panjang.
func ValidatePassword(plain string) error {
	n := len([]rune(plain))
	if n < MinPasswordLength {
		return fmt.Errorf("kata sandi minimal %d karakter", MinPasswordLength)
	}
	if n > MaxPasswordLength {
		return fmt.Errorf("kata sandi maksimal %d karakter", MaxPasswordLength)
	}
	return nil
}

// HashPassword menghasilkan digest Argon2id dalam format PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
//
// Salt dibuat acak tiap pemanggilan sehingga dua kata sandi identik tetap
// menghasilkan hash berbeda.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("membuat salt: %w", err)
	}
	return encodeArgon(salt, deriveArgon(plain, salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)), nil
}

// VerifyPassword mencocokkan plain dengan hash tersimpan. Constant-time,
// dan parameter diambil dari hash itu sendiri sehingga hash lama tetap sah
// bila parameter diubah di masa depan.
func VerifyPassword(encoded, plain string) (bool, error) {
	salt, hash, params, err := decodeArgon(encoded)
	if err != nil {
		return false, err
	}
	got := deriveArgon(plain, salt, params.time, params.memoryKiB, params.threads, uint32(len(hash)))
	return subtle.ConstantTimeCompare(got, hash) == 1, nil
}

// VerifyPasswordOrDummy memverifikasi kata sandi, tapi bila hash kosong (akun
// tidak ada, atau akun tanpa kata sandi) ia tetap menjalankan satu derivasi
// Argon2id penuh terhadap hash dummy lalu mengembalikan false.
//
// Tujuannya waktu respons identik, sehingga pesan gagal-masuk tidak bisa
// dipakai untuk menebak apakah sebuah email terdaftar (AC ISSUE-15 #2).
func VerifyPasswordOrDummy(encoded, plain string) bool {
	if encoded == "" {
		_, _ = VerifyPassword(dummyHash(), plain)
		return false
	}
	ok, err := VerifyPassword(encoded, plain)
	return err == nil && ok
}

// dummyHash adalah digest Argon2id valid yang dibuat sekali. Argumen pemakaian
// disimpan agar prosesnya hanya mahal satu kali per proses.
var dummyHash = sync.OnceValue(func() string {
	h, err := HashPassword("fisca-timing-equaliser-not-a-real-password")
	if err != nil {
		return ""
	}
	return h
})

type argonParams struct {
	memoryKiB uint32
	time      uint32
	threads   uint8
}

func deriveArgon(plain string, salt []byte, time, memoryKiB uint32, threads uint8, keyLen uint32) []byte {
	return argon2.IDKey([]byte(plain), salt, time, memoryKiB, threads, keyLen)
}

func encodeArgon(salt, hash []byte) string {
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		enc.EncodeToString(salt), enc.EncodeToString(hash),
	)
}

func decodeArgon(encoded string) (salt, hash []byte, params argonParams, err error) {
	// Format PHC: "" , algo, v=19, m=..,t=..,p=.., salt, hash → 6 bagian.
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		return nil, nil, params, ErrPasswordFormat
	}
	if parts[1] != "argon2id" {
		return nil, nil, params, fmt.Errorf("%w: algoritma %q tidak didukung", ErrPasswordFormat, parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, params, fmt.Errorf("%w: versi", ErrPasswordFormat)
	}
	if version != argon2.Version {
		return nil, nil, params, fmt.Errorf("%w: versi argon2 %d tidak didukung", ErrPasswordFormat, version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memoryKiB, &params.time, &params.threads); err != nil {
		return nil, nil, params, fmt.Errorf("%w: parameter", ErrPasswordFormat)
	}
	if params.memoryKiB == 0 || params.memoryKiB > maxMemoryKiB ||
		params.time == 0 || params.time > maxTime ||
		params.threads == 0 || params.threads > maxThreads {
		return nil, nil, params, fmt.Errorf("%w: parameter di luar batas aman", ErrPasswordFormat)
	}

	enc := base64.RawStdEncoding
	salt, err = enc.DecodeString(parts[4])
	if err != nil {
		return nil, nil, params, fmt.Errorf("%w: salt", ErrPasswordFormat)
	}
	hash, err = enc.DecodeString(parts[5])
	if err != nil {
		return nil, nil, params, fmt.Errorf("%w: hash", ErrPasswordFormat)
	}
	if len(salt) == 0 || len(hash) == 0 {
		return nil, nil, params, ErrPasswordFormat
	}
	return salt, hash, params, nil
}
