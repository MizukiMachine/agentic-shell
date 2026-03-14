package plugin

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type stubPromptBuilder struct {
	name string
}

func (b *stubPromptBuilder) Name() string {
	return b.name
}

func (b *stubPromptBuilder) Build(input string, _ map[string]interface{}) (string, error) {
	return "built: " + input, nil
}

type stubToolInferencer struct {
	name string
}

func (i *stubToolInferencer) Name() string {
	return i.name
}

func (i *stubToolInferencer) Infer(_ string, _ map[string]interface{}) ([]string, error) {
	return []string{"read"}, nil
}

type stubPlugin struct {
	name       string
	builders   []PromptBuilder
	inferencer []ToolInferencer
}

func (p *stubPlugin) Name() string {
	return p.name
}

func (p *stubPlugin) PromptBuilders() []PromptBuilder {
	return p.builders
}

func (p *stubPlugin) ToolInferencers() []ToolInferencer {
	return p.inferencer
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(&stubPlugin{
		name: "custom",
		builders: []PromptBuilder{
			&stubPromptBuilder{name: "custom-builder"},
		},
		inferencer: []ToolInferencer{
			&stubToolInferencer{name: "custom-inferencer"},
		},
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if _, ok := registry.PromptBuilder("custom-builder"); !ok {
		t.Fatal("expected prompt builder to be registered")
	}
	if _, ok := registry.ToolInferencer("custom-inferencer"); !ok {
		t.Fatal("expected tool inferencer to be registered")
	}
	if len(registry.Plugins()) != 1 {
		t.Fatalf("expected one plugin, got %d", len(registry.Plugins()))
	}
}

func TestRegistryLoadDiscoversManifestPlugins(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, ".claude", "plugins", "review")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	manifest := []byte(`
name: review
prompt_builders:
  - name: review-template
    template: "Review request: {{ .Input }}"
tool_inferencers:
  - name: review-tools
    matchers:
      security: ["read", "grep"]
      review: ["read"]
    default_tools: ["read"]
`)
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	registry := NewRegistry()
	if err := registry.Load(filepath.Join(dir, ".claude", "plugins")); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	builder, ok := registry.PromptBuilder("review-template")
	if !ok {
		t.Fatal("expected discovered prompt builder")
	}
	prompt, err := builder.Build("check this diff", nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if prompt != "Review request: check this diff" {
		t.Fatalf("unexpected prompt: %q", prompt)
	}

	inferencer, ok := registry.ToolInferencer("review-tools")
	if !ok {
		t.Fatal("expected discovered tool inferencer")
	}
	tools, err := inferencer.Infer("security review needed", nil)
	if err != nil {
		t.Fatalf("Infer returned error: %v", err)
	}
	expectedTools := []string{"read", "grep"}
	if !reflect.DeepEqual(tools, expectedTools) {
		t.Fatalf("unexpected tools: got %v want %v", tools, expectedTools)
	}
}

func TestRegistryLoadMissingDirectory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Load(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("expected missing plugin dir to be ignored, got %v", err)
	}
}
