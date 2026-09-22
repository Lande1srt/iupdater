package utils

import (
	"time"

	"github.com/pquerna/otp/totp"
)

// ValidateTOTP 验证 TOTP 验证码，返回 true 表示验证通过
// secret: TOTP 秘钥（Base32），code: 用户输入的 6 位数字
// 容差前后各 1 个时间窗口（30s × 2），应对网络延迟或时钟偏移
func ValidateTOTP(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// GenerateTOTP 生成当前时刻的 TOTP 验证码（用于调试/测试）
func GenerateTOTP(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
