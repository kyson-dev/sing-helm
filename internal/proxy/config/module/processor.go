package module

import (
	"strings"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
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
}

func NewOutboundProcessor() *OutboundProcessor {
	return &OutboundProcessor{
		usedTags:      make(map[string]bool),
		originalToTag: make(map[string]map[string]string),
		sourceGroups:  make(map[string][]string),
	}
}

// AddNodes processes a list of raw nodes gathered from a provider
func (p *OutboundProcessor) AddNodes(nodes []node.Node) {
	for _, n := range nodes {
		source := strings.TrimSpace(n.Source)
		if source == "" {
			source = "unknown"
		}

		// Ensure uniqueness of tag
		uniqueTag := MakeUniqueOutboundTag(n.Name, source, p.usedTags)
		p.recordMapping(source, n.Name, uniqueTag)

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

func (p *OutboundProcessor) recordMapping(source, original, unique string) {
	if p.originalToTag[source] == nil {
		p.originalToTag[source] = make(map[string]string)
	}
	p.originalToTag[source][original] = unique
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

	ApplyMapToOutbound(&outbound, rawCopy)
	return outbound
}

func (p *OutboundProcessor) resolveDetour(target string) (string, bool) {
	// 1. check globally reserved tags
	if IsReservedOutboundTag(target) {
		return target, true
	}

	// 2. linear search across all mappings
	// A more robust implementation would require knowing the source of the reference,
	// but usually users reference by the original name of a user node.
	for _, mapping := range p.originalToTag {
		if mapped, exists := mapping[target]; exists {
			return mapped, true
		}
	}

	// 3. direct use (assume user knows what they're doing)
	return target, false
}
