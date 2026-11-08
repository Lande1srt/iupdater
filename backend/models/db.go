package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	dbDir := filepath.Dir(getConfigPath())
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(getConfigPath()), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(
		&App{}, &AppVersion{}, &Component{},
		&StorageConfig{}, &StorageProfile{},
		&Agent{},
		&AdminAPIKey{}, // NEW
	)
	if err != nil {
		return err
	}

	ensureDefaultAPIKey()
	return nil
}

// init 专用的配置路径 — 避免循环 import config 包
func getConfigPath() string {
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	return "data/iupdater.db"
}

// ensureDefaultAPIKey：如果数据库里没有任何 AdminAPIKey，自动生成一个，
// 并打印到 stdout 让运维拷贝保存。这样部署即有 API Key，不用手动写 admin user。
func ensureDefaultAPIKey() {
	var count int64
	DB.Model(&AdminAPIKey{}).Count(&count)
	if count > 0 {
		return
	}

	plain, err := GenerateAdminAPIKeyPlain()
	if err != nil {
		log.Printf("[WARN] 生成默认 AdminAPIKey 失败: %v", err)
		return
	}
	hash := Sha256Hex(plain)

	key := AdminAPIKey{
		Name:        "Default (auto-generated on first boot)",
		KeyHash:     hash,
		Permissions: "*",
		Enabled:     true,
	}
	if err := DB.Create(&key).Error; err != nil {
		log.Printf("[WARN] 写入默认 AdminAPIKey 失败: %v", err)
		return
	}
	fmt.Println()
	fmt.Println("================================================================")
	fmt.Println("  iUpdater Admin APIKey created — 请立即保存到安全位置！")
	fmt.Println("  Key       :", plain)
	fmt.Println("  Header curl: curl -H \"X-API-Key: " + plain + "\" http://host:port/api/admin/...")
	fmt.Println("  或          : Authorization: Bearer APIKey-" + plain)
	fmt.Println()
	fmt.Println("  —— 接入 SeaStack 基座 ——")
	fmt.Println("  1) 浏览器打开 SeaStack → Dashboard → 🚀 iUpdater 控制台")
	fmt.Println("     (未配置时会自动弹出配置模态框)")
	fmt.Println("  2) 服务地址填: http://localhost:"+fmt.Sprintf("%d", configPortSafe())+"  (或你的公网域名)")
	fmt.Println("  3) API Key 填:  " + plain)
	fmt.Println("  4) 点「保存并测试」即可")
	fmt.Println()
	fmt.Println("  数据库已只保留 SHA256 哈希，明文丢失无法恢复，")
	fmt.Println("  可通过 POST /api/admin/api-keys/:id/regenerate 重新生成")
	fmt.Println("================================================================")
	fmt.Println()
}

// configPortSafe — 读取 PORT 环境变量（默认 8081），
// 用于打印基座接入指引时展示监听端口，避免循环 import config 包。
func configPortSafe() int {
	if p := os.Getenv("PORT"); p != "" {
		if n, err := fmt.Sscanf(p, "%d", new(int)); err == nil && n > 0 {
			var v int
			fmt.Sscanf(p, "%d", &v)
			return v
		}
	}
	return 8081
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// GenerateAdminAPIKeyPlain 生成明文 Admin API Key（带 iu_ 前缀，方便客户端识别）。
func GenerateAdminAPIKeyPlain() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "iu_" + hex.EncodeToString(b), nil
}

// Sha256Hex 是 AdminAPIKey.KeyHash 的哈希函数，middleware 和 handlers 用它校验 / 生成密钥。
func Sha256Hex(s string) string {
	return sha256Hex(s)
}
