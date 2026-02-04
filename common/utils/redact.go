package utils

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// Redact 对字符串进行敏感信息脱敏
// 规则包含：Authorization、token、secret、password、签名参数、邮箱、手机号等常见敏感字段
func Redact(s string) string {
	if s == "" {
		return s
	}
	// 常见键值的脱敏处理：key=xxxx -> key=****
	keyPatterns := []string{
		`(?i)(password)\s*=\s*[^&\s]+`,
		`(?i)(token)\s*=\s*[^&\s]+`,
		`(?i)(secret)\s*=\s*[^&\s]+`,
		`(?i)(signature)\s*=\s*[^&\s]+`,
		`(?i)(access[_-]?key)\s*=\s*[^&\s]+`,
	}
	for _, kp := range keyPatterns {
		re := regexp.MustCompile(kp)
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			parts := strings.SplitN(m, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=****"
			}
			return "****"
		})
	}
	// Authorization 头脱敏
	authRe := regexp.MustCompile(`(?i)(Authorization:\s*Bearer\s+)([A-Za-z0-9\.\-_]+)`)
	s = authRe.ReplaceAllString(s, `$1****`)

	// 邮箱脱敏：保留前两位和域名
	emailRe := regexp.MustCompile(`([A-Za-z0-9._%+\-]{2})[A-Za-z0-9._%+\-]*@([A-Za-z0-9.\-]+\.[A-Za-z]{2,})`)
	s = emailRe.ReplaceAllString(s, `$1****@$2`)

	// 简单手机号/长数字序列脱敏：保留前3后2
	phoneRe := regexp.MustCompile(`\b(\d{3})\d{4,10}(\d{2})\b`)
	s = phoneRe.ReplaceAllString(s, `$1****$2`)

	return s
}

// RedactHook 是一个 logrus Hook，在日志写出前对 message 和 string 字段执行脱敏
type RedactHook struct{}

func (h *RedactHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *RedactHook) Fire(entry *logrus.Entry) error {
	entry.Message = Redact(entry.Message)
	for k, v := range entry.Data {
		if str, ok := v.(string); ok {
			entry.Data[k] = Redact(str)
		}
	}
	return nil
}
