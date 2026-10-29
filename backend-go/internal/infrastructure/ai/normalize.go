package ai

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
)

// normalizeModelJSON 把模型输出的 JSON 对象清洗成后端结构体可直接反序列化的形状。
//
// 推理类模型（如 magpie 上的 conatus）不严格遵守 schema 的字面类型，实测会把
// video_timestamp 输出成 23.93 这样的浮点数、或 "45" 这样的数字字符串；而
// ParsedStep 用的是 *int，一旦类型不符 json.Unmarshal 直接失败，整次解析作废。
// 这里在解码前统一收口：
//   - 数字字符串（"45"、"12.5"）→ 数字；
//   - 带小数的时间/时长（video_timestamp、duration_seconds、estimated_minutes）
//     → 四舍五入为整数；
//   - 单独的 null 原样保留（对应“来源没给”，不允许强行估算）。
//
// 只处理这些已知字段，其余字段一律不动；任何解析异常都退回原 JSON，
// 让上层按原有逻辑报错，不吞掉真实问题。
func normalizeModelJSON(raw string) string {
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return raw
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return raw
	}

	steps, _ := obj["steps"].([]any)
	for _, s := range steps {
		step, ok := s.(map[string]any)
		if !ok {
			continue
		}
		roundIntField(step, "video_timestamp")
		roundIntField(step, "duration_seconds")
	}
	roundIntField(obj, "estimated_minutes")

	out, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return string(out)
}

// roundIntField 把 obj[key] 规范成整数：数字字符串转数字，浮点数四舍五入。
func roundIntField(obj map[string]any, key string) {
	switch v := obj[key].(type) {
	case json.Number:
		if n, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			obj[key] = n
			return
		}
		if f, err := v.Float64(); err == nil {
			obj[key] = int64(math.Round(f))
		}
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			obj[key] = n
			return
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			obj[key] = int64(math.Round(f))
		}
	}
}
