package node

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	moduleUtils "github.com/kyson-dev/sing-helm/internal/proxy/config/module/utils"
	"github.com/sagernet/sing-box/option"
)

// OutboundProcessor processes raw outbounds, manages tags, and prevents duplication globally.
type OutboundProcessor struct {
	usedTags       map[string]bool
	originalToTag  map[string]map[string]string // source -> original name -> unique tag
	processedNodes []option.Outbound
	actualTags     []string // purely the tags of actual nodes (vless, trojan, etc.)

	// sourceGroups maps source names (or 'user') to their nodes' tags. Useful for grouping.
	sourceGroups map[string][]string

	// globalFingerprints prevents identical nodes (same IP:Port+Type) across all sources.
	globalFingerprints map[string]bool
	fingerprintToTag   map[string]string
	globalNameToTag    map[string]string // original name -> unique tag (only when globally unambiguous)
	ambiguousNames     map[string]bool   // original names that map to multiple unique tags
}

func NewOutboundProcessor() *OutboundProcessor {
	return &OutboundProcessor{
		usedTags:           make(map[string]bool),
		originalToTag:      make(map[string]map[string]string),
		sourceGroups:       make(map[string][]string),
		globalFingerprints: make(map[string]bool),
		fingerprintToTag:   make(map[string]string),
		globalNameToTag:    make(map[string]string),
		ambiguousNames:     make(map[string]bool),
	}
}

// AddNodes processes a list of raw nodes gathered from a provider
func (p *OutboundProcessor) AddNodes(nodes []model.Node) {
	for _, n := range nodes {
		source := strings.TrimSpace(n.Source)
		if source == "" {
			source = "unknown"
		}

		// 1. Global deduplication
		var fp string
		if !n.SkipDedupe {
			fp = p.fingerprint(n)
			if p.globalFingerprints[fp] {
				// Keep duplicate-name mapping to canonical tag so detour references remain valid.
				if canonicalTag, ok := p.fingerprintToTag[fp]; ok {
					p.recordMapping(source, n.Name, canonicalTag)
				}
				continue
			}
			p.globalFingerprints[fp] = true
		}

		// Ensure uniqueness of tag
		uniqueTag := MakeUniqueOutboundTag(n.Name, source, p.usedTags)
		p.recordMapping(source, n.Name, uniqueTag)
		if !n.SkipDedupe {
			p.fingerprintToTag[fp] = uniqueTag
		}

		// Create the option.Outbound structure
		outbound := p.mapToOutbound(n.Type, uniqueTag, n.Outbound)

		p.processedNodes = append(p.processedNodes, outbound)
		p.actualTags = append(p.actualTags, uniqueTag)
		p.sourceGroups[source] = append(p.sourceGroups[source], uniqueTag)
	}
}

// GetProcessedOutbounds returns all properly mapped and tagged outbounds
func (p *OutboundProcessor) GetProcessedOutbounds() []option.Outbound {
	return p.processedNodes
}

// GetActualTags returns the tags of all registered proxy nodes
func (p *OutboundProcessor) GetActualTags() []string {
	return p.actualTags
}

// GetGroups returns tags grouped by their source origin
func (p *OutboundProcessor) GetGroups() map[string][]string {
	return p.sourceGroups
}

// --- Internal helpers ---

func (p *OutboundProcessor) fingerprint(n model.Node) string {
	if n.Outbound == nil {
		return n.Name + "|" + n.Type
	}

	identity := make(map[string]any, len(n.Outbound)+1)
	identity["type"] = n.Type
	for k, v := range n.Outbound {
		switch k {
		case "tag", "detour":
			continue
		default:
			identity[k] = v
		}
	}

	raw, err := json.Marshal(identity)
	if err == nil {
		return string(raw)
	}

	// Fallback to a coarse key only if marshal unexpectedly fails.
	if server, hasServer := n.Outbound["server"].(string); hasServer {
		if port, hasPort := n.Outbound["server_port"]; hasPort {
			return fmt.Sprintf("%s:%v|%s", server, port, n.Type)
		}
	}
	return n.Name + "|" + n.Type
}

func (p *OutboundProcessor) recordMapping(source, original, unique string) {
	if p.originalToTag[source] == nil {
		p.originalToTag[source] = make(map[string]string)
	}
	p.originalToTag[source][original] = unique

	// Keep a deterministic global-name mapping only when unambiguous.
	if original == "" || p.ambiguousNames[original] {
		return
	}
	if existing, ok := p.globalNameToTag[original]; ok && existing != unique {
		delete(p.globalNameToTag, original)
		p.ambiguousNames[original] = true
		return
	}
	p.globalNameToTag[original] = unique
}

func (p *OutboundProcessor) mapToOutbound(outType, tag string, raw map[string]any) option.Outbound {
	var outbound option.Outbound

	// Ensure tag matches our uniqueness guarantee
	rawCopy := make(map[string]any, len(raw))
	for k, v := range raw {
		rawCopy[k] = v
	}
	rawCopy["tag"] = tag
	rawCopy["type"] = outType

	// Handle internal detour logic if it references other nodes
	// e.g. wireguard nodes detour via another proxy
	if detour, ok := rawCopy["detour"].(string); ok && detour != "" {
		if mapped, found := p.resolveDetour(detour); found {
			rawCopy["detour"] = mapped
		}
	}

	moduleUtils.ApplyMapToOutbound(&outbound, rawCopy)
	return outbound
}

func (p *OutboundProcessor) resolveDetour(target string) (string, bool) {
	// 1. check globally reserved tags
	if IsReservedOutboundTag(target) {
		return target, true
	}

	// 2. already a concrete generated tag
	if p.usedTags[target] {
		return target, true
	}

	// 3. source-qualified lookup: "<source>/<original-name>"
	if source, name, ok := strings.Cut(target, "/"); ok && source != "" && name != "" {
		if mapping := p.originalToTag[source]; mapping != nil {
			if mapped, exists := mapping[name]; exists {
				return mapped, true
			}
		}
	}

	// 4. deterministic global-name lookup (only for unambiguous names)
	if mapped, ok := p.globalNameToTag[target]; ok {
		return mapped, true
	}

	// 5. direct use (assume user knows what they're doing)
	return target, false
}
