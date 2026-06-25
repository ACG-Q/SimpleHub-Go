package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"simplehub-go/internal/model"
)

type SetupService struct {
	db *gorm.DB
}

type InitResult struct {
	Port     int
	Entry    string
	Username string
	Password string
	IsFirst  bool
}

func NewSetupService(db *gorm.DB) *SetupService {
	return &SetupService{db: db}
}

func (s *SetupService) InitApp() (*InitResult, error) {
	existing := model.GetConfig(s.db, "jwt_secret")
	if existing != "" {
		port := model.GetConfig(s.db, "port")
		entry := model.GetConfig(s.db, "admin_entry")
		p := 0
		fmt.Sscanf(port, "%d", &p)
		return &InitResult{Port: p, Entry: entry, IsFirst: false}, nil
	}

	password := generatePassword(16)
	username := generateHex(8)
	entry := generateHex(8)
	port := generatePort()
	jwtSecret := generateHex(64)
	encKey := generateHex(32)

	if err := model.SetConfig(s.db, "port", fmt.Sprintf("%d", port)); err != nil {
		return nil, err
	}
	if err := model.SetConfig(s.db, "admin_entry", entry); err != nil {
		return nil, err
	}
	if err := model.SetConfig(s.db, "jwt_secret", jwtSecret); err != nil {
		return nil, err
	}
	if err := model.SetConfig(s.db, "encryption_key", encKey); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := model.User{
		ID:           newID(),
		Email:        username,
		PasswordHash: string(hash),
		IsAdmin:      true,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &InitResult{Port: port, Entry: entry, Username: username, Password: password, IsFirst: true}, nil
}

func (s *SetupService) Reset() (*InitResult, error) {
	password := generatePassword(16)
	username := generateHex(8)
	entry := generateHex(8)
	port := generatePort()
	jwtSecret := generateHex(64)
	encKey := generateHex(32)

	model.SetConfig(s.db, "port", fmt.Sprintf("%d", port))
	model.SetConfig(s.db, "admin_entry", entry)
	model.SetConfig(s.db, "jwt_secret", jwtSecret)
	model.SetConfig(s.db, "encryption_key", encKey)

	s.db.Where("is_admin = ?", true).Delete(&model.User{})

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := model.User{
		ID:           newID(),
		Email:        username,
		PasswordHash: string(hash),
		IsAdmin:      true,
	}
	s.db.Create(&user)

	return &InitResult{Port: port, Entry: entry, Username: username, Password: password, IsFirst: true}, nil
}

func (s *SetupService) GetJWTSecret() string {
	return model.GetConfig(s.db, "jwt_secret")
}

func (s *SetupService) GetEncryptionKey() string {
	return model.GetConfig(s.db, "encryption_key")
}

func (s *SetupService) GetPort() int {
	p := model.GetConfig(s.db, "port")
	var port int
	fmt.Sscanf(p, "%d", &port)
	if port == 0 {
		return 10000
	}
	return port
}

func (s *SetupService) GetEntry() string {
	return model.GetConfig(s.db, "admin_entry")
}

var commonPorts = map[int]bool{
	80: true, 443: true, 3000: true, 8000: true, 8080: true,
	8443: true, 9090: true, 9000: true, 5000: true, 4000: true,
	3306: true, 5432: true, 6379: true, 27017: true, 22: true,
	21: true, 25: true, 53: true, 110: true, 143: true,
}

func generatePort() int {
	for {
		n, _ := rand.Int(rand.Reader, big.NewInt(55535))
		p := int(n.Int64()) + 10000
		if !commonPorts[p] {
			return p
		}
	}
}

func generateHex(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generatePassword(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteByte(chars[n.Int64()])
	}
	return b.String()
}
