// Command resetpassword mengganti kata sandi sebuah akun dan mencabut semua
// sesinya. Ini jalur pemulihan lokal FISCA: tidak ada reset lewat email karena
// aplikasi dipakai untuk pemakaian pribadi tanpa SMTP (keputusan ISSUE-15).
//
// Jalankan dari mesin yang punya akses database:
//
//	npm run auth:reset -- anda@example.com kata-sandi-baru
//	echo "kata-sandi-baru" | npm run auth:reset -- anda@example.com
//
// Argumen kata sandi opsional; bila tidak diberikan, dibaca dari stdin.
// Catatan: bila ditulis sebagai argumen, kata sandi ikut tersimpan di riwayat
// shell — pakai bentuk pipa bila itu menjadi masalah.
//
// Berbeda dari cmd/seed, perintah ini BOLEH jalan di produksi: tool inilah
// yang dipakai saat akun terkunci.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/config"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/envfile"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

func main() {
	envfile.Load(".env", "../../.env")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "pemakaian: resetpassword <email> [kata-sandi]")
		os.Exit(2)
	}
	email := strings.TrimSpace(os.Args[1])
	if email == "" {
		log.Fatal("email tidak boleh kosong")
	}

	password, err := readPassword(os.Args[2:])
	if err != nil {
		log.Fatal(err)
	}

	// Hanya DATABASE_URL yang dibaca. config.Load sengaja tidak dipakai: saat
	// pengguna terkunci, tool ini tidak boleh gagal karena masalah konfigurasi
	// lain seperti AUTH_MODE yang tidak dikenal.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Config kosong: ResetPassword hanya memakai pool dan query, tidak ada
	// konfigurasi runtime yang relevan di sini.
	svc := service.New(pool, &config.Config{})

	result, err := svc.ResetPassword(ctx, email, password)
	if err != nil {
		log.Fatalf("reset gagal: %v", err)
	}

	fmt.Printf("kata sandi diperbarui untuk %s\n", result.User.Email)
	fmt.Printf("sesi dicabut: %d\n", result.RevokedSessions)
	fmt.Println("masuk kembali dengan kata sandi yang baru.")
}

// readPassword mengambil kata sandi dari argumen bila ada, atau satu baris
// dari stdin. Stdin memungkinkan bentuk pipa sehingga kata sandi tidak masuk
// riwayat shell.
//
// Aturannya sama dengan endpoint daftar (auth.ValidatePassword), supaya pesan
// galat yang dilihat pengguna konsisten dengan halaman login.
func readPassword(args []string) (string, error) {
	if len(args) > 0 {
		password := args[0]
		if err := auth.ValidatePassword(password); err != nil {
			return "", err
		}
		return password, nil
	}

	fmt.Fprint(os.Stderr, "Kata sandi baru: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("membaca kata sandi dari stdin: %w", err)
	}
	password := strings.TrimRight(line, "\r\n")
	if err := auth.ValidatePassword(password); err != nil {
		return "", err
	}
	return password, nil
}
