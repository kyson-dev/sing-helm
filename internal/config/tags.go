package config

import (
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

func MakeUniqueOutboundTag(baseName, source string, used map[string]bool) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = strings.TrimSpace(source)
	}
	if base == "" {
		base = "node"
	}
	tag := base

	if IsReservedOutboundTag(tag) || used[tag] {
		tag = base + " (" + source + ")"
	}

	if IsReservedOutboundTag(tag) || used[tag] {
		for i := 2; ; i++ {
			candidate := base + " (" + source + ") #" + strconv.Itoa(i)
			if !IsReservedOutboundTag(candidate) && !used[candidate] {
				tag = candidate
				break
			}
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
