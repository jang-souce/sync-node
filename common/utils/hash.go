package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// MD5 计算字符串的 MD5
func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// FileMD5 计算文件的 MD5
func FileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256 计算字符串的 SHA256
func SHA256(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
