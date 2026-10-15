package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

// sanitizeEmail 去除首尾空白，并移除 C0 控制字符（0x00-0x1F）与 DEL（0x7F）。
// 防止输入法/自动填充混入的不可见字符导致 email 格式校验失败。
func sanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, email)
}

// cleanEmailInJSON 解析请求体 JSON，对 email 字段执行 sanitizeEmail 后重新序列化。
func cleanEmailInJSON(raw []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw, err
	}
	if v, ok := m["email"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			if cleaned := sanitizeEmail(s); cleaned != s {
				b, err := json.Marshal(cleaned)
				if err != nil {
					return raw, err
				}
				m["email"] = b
			}
		}
	}
	return json.Marshal(m)
}

// prepareBodyWithCleanEmail 读取请求体并清洗 email 字段后写回，供后续 ShouldBindJSON 使用。
func prepareBodyWithCleanEmail(c *gin.Context) error {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	cleaned, err := cleanEmailInJSON(raw)
	if err != nil {
		return err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(cleaned))
	return nil
}
