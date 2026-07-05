package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"iupdater/models"
)

func GenerateVersionCode() (int64, error) {
	now := time.Now()
	datePrefix := now.Format("20060102") // YYYYMMDD 格式，例如 20260705

	// 将日期字符串转为整数作为前缀
	var datePrefixNum int64
	fmt.Sscanf(datePrefix, "%d", &datePrefixNum)

	// 计算当天版本号的范围：YYYYMMDD0000 到 YYYYMMDD9999
	minVersionCode := datePrefixNum * 10000
	maxVersionCode := minVersionCode + 9999

	// 查询数据库中当天最大的版本号
	var result struct {
		MaxVersion int64
	}

	models.DB.Raw(`SELECT COALESCE(MAX(version_code), 0) as max_version 
		FROM app_versions 
		WHERE version_code >= ? AND version_code <= ?`,
		minVersionCode, maxVersionCode).Scan(&result)

	var sequence int64
	if result.MaxVersion == 0 || result.MaxVersion < minVersionCode {
		// 当天第一个版本，序号从 0001 开始
		sequence = 1
	} else {
		// 提取当前序号（版本号的后四位）
		sequence = (result.MaxVersion % 10000) + 1
	}

	// 检查是否溢出（超过 9999 个版本）
	if sequence > 9999 {
		return 0, fmt.Errorf("version overflow: exceeded 9999 versions for date %s", datePrefix)
	}

	// 组合最终版本号：YYYYMMDD + 四位序号
	versionCode := minVersionCode + sequence

	return versionCode, nil
}

func GenerateAccessKey() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func ComputeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ComputeBytesHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
