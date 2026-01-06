package subscription

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

// Node is a normalized outbound entry derived from subscriptions.
// Outbound contains sing-box outbound fields without tag.
type Node struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Source   string         `json:"source,omitempty"`
	Outbound map[string]any `json:"outbound"`
}

// Cache stores parsed nodes from a subscription source.
type Cache struct {
	Source    Source `json:"source"`
	UpdatedAt string `json:"updated_at"`
	Nodes     []Node `json:"nodes"`
}

const (
	FormatAuto    = "auto"
	FormatSingBox = "singbox"
	FormatClash   = "clash"
)

func NormalizeFormat(format string) string {
	switch format {
	case "", "auto":
		return FormatAuto
	case "json":
		return FormatAuto
	case "sing-box", "singbox":
		return FormatSingBox
	case "clash":
		return FormatClash
	default:
		return format
	}
}

func (s *Source) NormalizeDefaults(name string) {
	if s.Name == "" {
		s.Name = name
	}
	s.Format = NormalizeFormat(s.Format)
	if s.Format == "" {
		s.Format = FormatAuto
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

func IsActualOutboundType(outType string) bool {
	switch outType {
	case "selector", "urltest", "direct", "block", "dns":
		return false
	default:
		return true
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
