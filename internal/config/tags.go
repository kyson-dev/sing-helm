package config

import (
	"fmt"
	"strconv"
	"strings"
)

var reservedOutboundTags = map[string]bool{
	"direct": true,
	"block":  true,
	"proxy":  true,
	"auto":   true,
}

func IsReservedOutboundTag(tag string) bool {
	return reservedOutboundTags[tag]
}

// MakeUniqueTag 生成唯一标签，如果冲突则添加后缀
func MakeUniqueTag(baseName string, used map[string]bool) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = "node"
	}
	tag := base

	// 如果不冲突且不是保留字，直接使用
	if !IsReservedOutboundTag(tag) && !used[tag] {
		used[tag] = true
		return tag
	}

	// 冲突处理：循环尝试添加数字后缀
	// 格式: base #2, base #3 ...
	for i := 2; ; i++ {
		// 使用简单的后缀格式，或者可以沿用订阅的格式
		candidate := fmt.Sprintf("%s #%d", base, i)
		if !IsReservedOutboundTag(candidate) && !used[candidate] {
			tag = candidate
			break
		}
	}

	used[tag] = true
	return tag
}

func MakeUniqueOutboundTag(baseName, source string, used map[string]bool) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = strings.TrimSpace(source)
	}
	if base == "" {
		base = "node"
	}
	tag := base

	// 1. 先尝试直接使用 base name（不加 source）
	if !IsReservedOutboundTag(tag) && !used[tag] {
		used[tag] = true
		return tag
	}

	// 2. 冲突了，尝试加 source
	if source != "" {
		tag = base + " (" + source + ")"
		if !IsReservedOutboundTag(tag) && !used[tag] {
			used[tag] = true
			return tag
		}
	}

	// 3. 还是冲突，加数字后缀
	for i := 2; ; i++ {
		var candidate string
		if source != "" {
			candidate = base + " (" + source + ") #" + strconv.Itoa(i)
		} else {
			candidate = base + " #" + strconv.Itoa(i)
		}
		if !IsReservedOutboundTag(candidate) && !used[candidate] {
			tag = candidate
			break
		}
	}

	used[tag] = true
	return tag
}

func IsActualOutboundType(outType string) bool {
	switch outType {
	case "selector", "urltest", "direct", "block", "dns":
		return false
	default:
		return true
	}
}
