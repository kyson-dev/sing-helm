package subscription

import (
	"github.com/kyson-dev/sing-helm/internal/proxy/config/node"
	"github.com/kyson-dev/sing-helm/internal/proxy/config/parser"
)

// Source describes a subscription config file.
type Source struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Format   string   `json:"format"` // auto, singbox, clash
	Enabled  *bool    `json:"enabled"`
	Priority int      `json:"priority"`
	Dedupe   *bool    `json:"dedupe"`
	Tags     []string `json:"tags,omitempty"`
}

// Cache stores parsed nodes from a subscription source.
type Cache struct {
	Source    Source      `json:"source"`
	UpdatedAt string      `json:"updated_at"`
	Nodes     []node.Node `json:"nodes"`
}

func (s *Source) NormalizeDefaults(name string) {
	if s.Name == "" {
		s.Name = name
	}
	s.Format = parser.NormalizeFormat(s.Format)
	if s.Format == "" {
		s.Format = parser.FormatAuto
	}
	if s.Enabled == nil {
		enabled := true
		s.Enabled = &enabled
	}
	if s.Dedupe == nil {
		dedupe := true
		s.Dedupe = &dedupe
	}
}

func (s Source) EnabledValue() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

func (s Source) DedupeValue() bool {
	if s.Dedupe == nil {
		return true
	}
	return *s.Dedupe
}
