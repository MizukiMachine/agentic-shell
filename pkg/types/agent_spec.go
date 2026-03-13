// Package types provides core type definitions for the agentic-shell.
// This file contains AgentSpec types - the intermediate representation.
package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ============================================================================
// AgentSpec - 中間表現 (Intermediate Representation)
// ============================================================================

// AgentSpecMetadata represents metadata for an AgentSpec.
type AgentSpecMetadata struct {
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Author      string   `json:"author,omitempty" yaml:"author,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Validate checks if the AgentSpecMetadata is valid.
func (m *AgentSpecMetadata) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metadata name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("metadata version is required")
	}
	return nil
}

// Capability represents a single capability of an agent.
type Capability struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Category    string   `json:"category" yaml:"category"`
	Level       string   `json:"level" yaml:"level"` // beginner, intermediate, expert
	Keywords    []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
}

// Validate checks if the Capability is valid.
func (c *Capability) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("capability ID is required")
	}
	if c.Name == "" {
		return fmt.Errorf("capability name is required")
	}
	return nil
}

// Skill represents a specific skill an agent possesses.
type Skill struct {
	ID            string   `json:"id" yaml:"id"`
	Name          string   `json:"name" yaml:"name"`
	Description   string   `json:"description" yaml:"description"`
	Prerequisites []string `json:"prerequisites,omitempty" yaml:"prerequisites,omitempty"`
	Examples      []string `json:"examples,omitempty" yaml:"examples,omitempty"`
	Complexity    string   `json:"complexity" yaml:"complexity"` // low, medium, high
}

// Validate checks if the Skill is valid.
func (s *Skill) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("skill ID is required")
	}
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	return nil
}

// ToolParameter represents a parameter for a tool.
type ToolParameter struct {
	Name        string      `json:"name" yaml:"name"`
	Type        string      `json:"type" yaml:"type"` // string, number, boolean, array, object
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty" yaml:"enum,omitempty"`
}

// Validate checks if the ToolParameter is valid.
func (p *ToolParameter) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("parameter name is required")
	}
	if p.Type == "" {
		return fmt.Errorf("parameter type is required")
	}
	return nil
}

// Tool represents a tool available to the agent.
type Tool struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Category    string          `json:"category" yaml:"category"` // io, processing, communication, etc.
	Parameters  []ToolParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Returns     string          `json:"returns,omitempty" yaml:"returns,omitempty"`
	Timeout     int             `json:"timeout,omitempty" yaml:"timeout,omitempty"` // timeout in seconds
	RiskLevel   string          `json:"risk_level" yaml:"risk_level"`               // low, medium, high, critical
}

// Validate checks if the Tool is valid.
func (t *Tool) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("tool ID is required")
	}
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	for i, p := range t.Parameters {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("parameter[%d]: %w", i, err)
		}
	}
	return nil
}

// BehaviorRule represents a rule for agent behavior.
type BehaviorRule struct {
	ID        string   `json:"id" yaml:"id"`
	Name      string   `json:"name" yaml:"name"`
	Condition string   `json:"condition" yaml:"condition"`
	Action    string   `json:"action" yaml:"action"`
	Priority  int      `json:"priority" yaml:"priority"`
	Enabled   bool     `json:"enabled" yaml:"enabled"`
	Tags      []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Validate checks if the BehaviorRule is valid.
func (r *BehaviorRule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if r.Condition == "" {
		return fmt.Errorf("rule condition is required")
	}
	if r.Action == "" {
		return fmt.Errorf("rule action is required")
	}
	return nil
}

// KnowledgeSource represents a knowledge source for the agent.
type KnowledgeSource struct {
	ID          string `json:"id" yaml:"id"`
	Type        string `json:"type" yaml:"type"` // documentation, api, database, file, url
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URI         string `json:"uri,omitempty" yaml:"uri,omitempty"`
	Priority    int    `json:"priority" yaml:"priority"`
	CacheTTL    int    `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"` // cache TTL in seconds
}

// Validate checks if the KnowledgeSource is valid.
func (k *KnowledgeSource) Validate() error {
	if k.ID == "" {
		return fmt.Errorf("knowledge source ID is required")
	}
	if k.Type == "" {
		return fmt.Errorf("knowledge source type is required")
	}
	return nil
}

// CommunicationProtocol represents communication settings.
type CommunicationProtocol struct {
	Type           string   `json:"type" yaml:"type"`                                         // rest, grpc, websocket, cli
	Format         string   `json:"format" yaml:"format"`                                     // json, protobuf, yaml
	Authentication string   `json:"authentication,omitempty" yaml:"authentication,omitempty"` // none, api_key, oauth, etc.
	AllowedMethods []string `json:"allowed_methods,omitempty" yaml:"allowed_methods,omitempty"`
	RateLimit      int      `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"` // requests per minute
}

// Validate checks if the CommunicationProtocol is valid.
func (c *CommunicationProtocol) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("communication protocol type is required")
	}
	if c.Format == "" {
		return fmt.Errorf("communication protocol format is required")
	}
	return nil
}

// PerformanceConfig represents performance configuration.
type PerformanceConfig struct {
	MaxConcurrency int     `json:"max_concurrency" yaml:"max_concurrency"`
	Timeout        int     `json:"timeout" yaml:"timeout"` // default timeout in seconds
	RetryCount     int     `json:"retry_count" yaml:"retry_count"`
	RetryDelay     int     `json:"retry_delay" yaml:"retry_delay"`                       // delay in milliseconds
	MemoryLimit    int64   `json:"memory_limit,omitempty" yaml:"memory_limit,omitempty"` // in bytes
	CPULimit       float64 `json:"cpu_limit,omitempty" yaml:"cpu_limit,omitempty"`       // percentage (0-100)
	Priority       string  `json:"priority" yaml:"priority"`                             // low, normal, high
}

// Validate checks if the PerformanceConfig is valid.
func (p *PerformanceConfig) Validate() error {
	if p.MaxConcurrency < 1 {
		return fmt.Errorf("max_concurrency must be at least 1, got: %d", p.MaxConcurrency)
	}
	if p.Timeout < 1 {
		return fmt.Errorf("timeout must be at least 1 second, got: %d", p.Timeout)
	}
	return nil
}

// SecurityConfig represents security configuration.
type SecurityConfig struct {
	SandboxEnabled     bool     `json:"sandbox_enabled" yaml:"sandbox_enabled"`
	AllowedDomains     []string `json:"allowed_domains,omitempty" yaml:"allowed_domains,omitempty"`
	AllowedCommands    []string `json:"allowed_commands,omitempty" yaml:"allowed_commands,omitempty"`
	DataClassification string   `json:"data_classification" yaml:"data_classification"` // public, internal, confidential, restricted
	AuditEnabled       bool     `json:"audit_enabled" yaml:"audit_enabled"`
	EncryptionRequired bool     `json:"encryption_required" yaml:"encryption_required"`
}

// Validate checks if the SecurityConfig is valid.
func (s *SecurityConfig) Validate() error {
	return nil
}

// AgentSpec represents the intermediate representation of an agent.
// It bridges IntentSpace and ClaudeAgentDefinition.
type AgentSpec struct {
	// Metadata for the agent
	Metadata AgentSpecMetadata `json:"metadata" yaml:"metadata"`

	// Intent space from which this spec was derived
	Intent IntentSpace `json:"intent" yaml:"intent"`

	// Capabilities the agent should have
	Capabilities []Capability `json:"capabilities" yaml:"capabilities"`

	// Skills the agent should possess
	Skills []Skill `json:"skills" yaml:"skills"`

	// Tools available to the agent
	Tools []Tool `json:"tools" yaml:"tools"`

	// Behavior rules for the agent
	BehaviorRules []BehaviorRule `json:"behavior_rules,omitempty" yaml:"behavior_rules,omitempty"`

	// Knowledge sources for the agent
	KnowledgeSources []KnowledgeSource `json:"knowledge_sources,omitempty" yaml:"knowledge_sources,omitempty"`

	// Communication configuration
	Communication CommunicationProtocol `json:"communication" yaml:"communication"`

	// Performance configuration
	Performance PerformanceConfig `json:"performance" yaml:"performance"`

	// Security configuration
	Security SecurityConfig `json:"security" yaml:"security"`
}

// Validate checks if the AgentSpec is valid.
func (s *AgentSpec) Validate() error {
	if err := s.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	if err := s.Intent.Validate(); err != nil {
		return fmt.Errorf("intent: %w", err)
	}
	for i, c := range s.Capabilities {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("capability[%d]: %w", i, err)
		}
	}
	for i, sk := range s.Skills {
		if err := sk.Validate(); err != nil {
			return fmt.Errorf("skill[%d]: %w", i, err)
		}
	}
	for i, t := range s.Tools {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("tool[%d]: %w", i, err)
		}
	}
	for i, r := range s.BehaviorRules {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("behavior_rule[%d]: %w", i, err)
		}
	}
	for i, k := range s.KnowledgeSources {
		if err := k.Validate(); err != nil {
			return fmt.Errorf("knowledge_source[%d]: %w", i, err)
		}
	}
	if err := s.Communication.Validate(); err != nil {
		return fmt.Errorf("communication: %w", err)
	}
	if err := s.Performance.Validate(); err != nil {
		return fmt.Errorf("performance: %w", err)
	}
	if err := s.Security.Validate(); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	return nil
}

// ToJSON serializes AgentSpec to JSON.
func (s *AgentSpec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// ToYAML serializes AgentSpec to a YAML string.
func (s *AgentSpec) ToYAML() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}

	var builder strings.Builder
	writeYAMLValue(&builder, value, 0)
	return builder.String(), nil
}

// FromJSON deserializes AgentSpec from JSON.
func (s *AgentSpec) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// NewAgentSpec creates a new AgentSpec with default values.
func NewAgentSpec(name, version string) *AgentSpec {
	return &AgentSpec{
		Metadata: AgentSpecMetadata{
			Name:    name,
			Version: version,
			Tags:    []string{},
		},
		Capabilities:     []Capability{},
		Skills:           []Skill{},
		Tools:            []Tool{},
		BehaviorRules:    []BehaviorRule{},
		KnowledgeSources: []KnowledgeSource{},
		Communication: CommunicationProtocol{
			Type:   "rest",
			Format: "json",
		},
		Performance: PerformanceConfig{
			MaxConcurrency: 1,
			Timeout:        30,
			RetryCount:     3,
			RetryDelay:     1000,
			Priority:       "normal",
		},
		Security: SecurityConfig{
			SandboxEnabled:     true,
			DataClassification: "internal",
			AuditEnabled:       true,
			EncryptionRequired: false,
		},
	}
}

func writeYAMLValue(builder *strings.Builder, value interface{}, indent int) {
	switch v := value.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			writeIndent(builder, indent)
			builder.WriteString(key)
			if isScalarYAMLValue(v[key]) {
				builder.WriteString(": ")
				builder.WriteString(formatYAMLScalar(v[key]))
				builder.WriteByte('\n')
				continue
			}
			builder.WriteString(":\n")
			writeYAMLValue(builder, v[key], indent+2)
		}
	case []interface{}:
		if len(v) == 0 {
			writeIndent(builder, indent)
			builder.WriteString("[]\n")
			return
		}
		for _, item := range v {
			writeIndent(builder, indent)
			builder.WriteString("-")
			if isScalarYAMLValue(item) {
				builder.WriteByte(' ')
				builder.WriteString(formatYAMLScalar(item))
				builder.WriteByte('\n')
				continue
			}
			builder.WriteByte('\n')
			writeYAMLValue(builder, item, indent+2)
		}
	default:
		writeIndent(builder, indent)
		builder.WriteString(formatYAMLScalar(v))
		builder.WriteByte('\n')
	}
}

func writeIndent(builder *strings.Builder, indent int) {
	for i := 0; i < indent; i++ {
		builder.WriteByte(' ')
	}
}

func isScalarYAMLValue(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return false
	default:
		return true
	}
}

func formatYAMLScalar(value interface{}) string {
	if value == nil {
		return "null"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}
