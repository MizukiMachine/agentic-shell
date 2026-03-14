package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MizukiMachine/agentic-shell/internal/pipeline"
	"gopkg.in/yaml.v3"
)

func loadPipelineEnvelope(from string) (*pipeline.Envelope, []byte, error) {
	var data []byte
	var err error

	if from != "" {
		data, err = os.ReadFile(from)
	} else {
		data, err = readOptionalStdin()
	}
	if err != nil {
		return nil, nil, err
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return &pipeline.Envelope{}, nil, nil
	}

	env, ok, err := pipeline.DecodeEnvelope(data)
	if err != nil {
		return nil, nil, err
	}
	if ok {
		return env, nil, nil
	}

	return &pipeline.Envelope{}, data, nil
}

func readOptionalStdin() ([]byte, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	return io.ReadAll(os.Stdin)
}

func writeStructuredOutput(value interface{}, format, outputPath string) error {
	var data []byte
	var err error

	switch format {
	case "yaml":
		data, err = yaml.Marshal(value)
	default:
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return err
	}

	if outputPath == "" {
		fmt.Println(string(data))
		return nil
	}

	fullPath := outputPath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(GetOutputDir(), outputPath)
	}
	parent := filepath.Dir(fullPath)
	if parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(fullPath, data, 0644)
}

func ensureParsedEnvelope(env *pipeline.Envelope, raw []byte, stdinName string, args []string) (*pipeline.Envelope, error) {
	if env == nil {
		env = &pipeline.Envelope{}
	}
	if len(args) > 0 {
		return pipeline.ParseFiles(env, args)
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		return pipeline.ParseStdin(env, stdinName, raw)
	}
	return env, nil
}

func ensureExtractedEnvelope(env *pipeline.Envelope, raw []byte, stdinName string, args []string) (*pipeline.Envelope, error) {
	var err error
	env, err = ensureParsedEnvelope(env, raw, stdinName, args)
	if err != nil {
		return nil, err
	}
	if env.Extraction == nil {
		if err := pipeline.Extract(env); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func ensureScannedEnvelope(env *pipeline.Envelope, raw []byte, stdinName string, args []string, skillsDir string) (*pipeline.Envelope, error) {
	var err error
	env, err = ensureExtractedEnvelope(env, raw, stdinName, args)
	if err != nil {
		return nil, err
	}
	if env.SkillScan == nil {
		if err := pipeline.ScanSkills(env, skillsDir); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func ensureMatchedEnvelope(env *pipeline.Envelope, raw []byte, stdinName string, args []string, skillsDir string) (*pipeline.Envelope, error) {
	var err error
	env, err = ensureScannedEnvelope(env, raw, stdinName, args, skillsDir)
	if err != nil {
		return nil, err
	}
	if env.Match == nil {
		if err := pipeline.MatchSkills(env); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func ensureGeneratedEnvelope(env *pipeline.Envelope, raw []byte, stdinName string, args []string, skillsDir string) (*pipeline.Envelope, error) {
	var err error
	env, err = ensureMatchedEnvelope(env, raw, stdinName, args, skillsDir)
	if err != nil {
		return nil, err
	}
	if env.SkillGen == nil {
		if err := pipeline.GenerateMissingSkills(env); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func normalizeOutputFormat(format string) string {
	if strings.EqualFold(format, "json") {
		return "json"
	}
	return "yaml"
}
