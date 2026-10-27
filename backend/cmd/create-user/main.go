package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/auth"
	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/database"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

func main() {
	name := flag.String("name", "", "Nama lengkap pengguna")
	email := flag.String("email", "", "Email pengguna")
	role := flag.String("role", "staff", "Role: admin, staff, reviewer")
	flag.Parse()

	*name = strings.TrimSpace(*name)
	*email = strings.TrimSpace(*email)
	*role = strings.ToLower(strings.TrimSpace(*role))

	if *name == "" || *email == "" {
		log.Fatal("Gunakan -name dan -email.")
	}

	if *role != "admin" && *role != "staff" && *role != "reviewer" {
		log.Fatal("Role harus admin, staff, atau reviewer.")
	}

	if err := godotenv.Load(); err != nil {
		log.Printf("File .env tidak dimuat: %v; menggunakan environment sistem", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Password (minimal 12 karakter): ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatalf("Gagal membaca password: %v", err)
	}

	fmt.Print("Ulangi password: ")
	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatalf("Gagal membaca konfirmasi password: %v", err)
	}

	// Pastikan input berikutnya tidak tertinggal jika terminal
	// mengirim newline setelah password.
	_ = reader

	if string(password) != string(confirmation) {
		log.Fatal("Password tidak cocok.")
	}

	passwordHash, err := auth.HashPassword(string(password))
	for i := range password {
		password[i] = 0
	}
	for i := range confirmation {
		confirmation[i] = 0
	}

	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatalf("Koneksi database gagal: %v", err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		INSERT INTO users (full_name, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
	`, *name, strings.ToLower(*email), passwordHash, *role)

	if err != nil {
		log.Fatalf("Gagal membuat pengguna. Pastikan email belum digunakan dan role valid: %v", err)
	}

	fmt.Printf("Pengguna berhasil dibuat: %s (%s)\n", *email, *role)
}
