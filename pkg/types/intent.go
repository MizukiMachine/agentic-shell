// Package types provides core type definitions for the agentic-shell.
// This file contains IntentSpace types ported from dyna-agent-system.
package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Goals Dimension (g) - 目標次元
// ============================================================================

// GoalPriority represents goal priority levels.
type GoalPriority string

const (
	GoalPriorityCritical GoalPriority = "critical"
	GoalPriorityHigh     GoalPriority = "high"
	GoalPriorityMedium   GoalPriority = "medium"
	GoalPriorityLow      GoalPriority = "low"
)

// GoalType represents goal type classification.
type GoalType string

const (
	GoalTypePrimary   GoalType = "primary"
	GoalTypeSecondary GoalType = "secondary"
	GoalTypeImplicit  GoalType = "implicit"
	GoalTypeDerived   GoalType = "derived"
)

// Goal represents an individual goal definition.
type Goal struct {
	ID              string       `json:"id" yaml:"id"`
	Type            GoalType     `json:"type" yaml:"type"`
	Description     string       `json:"description" yaml:"description"`
	Priority        GoalPriority `json:"priority" yaml:"priority"`
	Measurable      bool         `json:"measurable" yaml:"measurable"`
	SuccessCriteria []string     `json:"success_criteria,omitempty" yaml:"success_criteria,omitempty"`
	Dependencies    []string     `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Deadline        string       `json:"deadline,omitempty" yaml:"deadline,omitempty"` // ISO 8601
}

// Validate checks if the Goal is valid.
func (g *Goal) Validate() error {
	if g.ID == "" {
		return fmt.Errorf("goal ID is required")
	}
	if g.Description == "" {
		return fmt.Errorf("goal description is required")
	}
	if g.Type != GoalTypePrimary && g.Type != GoalTypeSecondary &&
		g.Type != GoalTypeImplicit && g.Type != GoalTypeDerived {
		return fmt.Errorf("invalid goal type: %s", g.Type)
	}
	if g.Priority != GoalPriorityCritical && g.Priority != GoalPriorityHigh &&
		g.Priority != GoalPriorityMedium && g.Priority != GoalPriorityLow {
		return fmt.Errorf("invalid goal priority: %s", g.Priority)
	}
	if g.Deadline != "" {
		if _, err := time.Parse(time.RFC3339, g.Deadline); err != nil {
			return fmt.Errorf("invalid deadline format: %w", err)
		}
	}
	return nil
}

// PrimaryGoals represents explicit main objectives.
type PrimaryGoals struct {
	Main       Goal   `json:"main" yaml:"main"`
	Supporting []Goal `json:"supporting" yaml:"supporting"`
}

// Validate checks if the PrimaryGoals is valid.
func (pg *PrimaryGoals) Validate() error {
	if err := pg.Main.Validate(); err != nil {
		return fmt.Errorf("main goal: %w", err)
	}
	for i, g := range pg.Supporting {
		if err := g.Validate(); err != nil {
			return fmt.Errorf("supporting goal[%d]: %w", i, err)
		}
	}
	return nil
}

// SecondaryGoals represents nice-to-have objectives.
type SecondaryGoals struct {
	Goals         []Goal   `json:"goals" yaml:"goals"`
	PriorityOrder []string `json:"priority_order" yaml:"priority_order"` // goal IDs in priority order
}

// Validate checks if the SecondaryGoals is valid.
func (sg *SecondaryGoals) Validate() error {
	for i, g := range sg.Goals {
		if err := g.Validate(); err != nil {
			return fmt.Errorf("secondary goal[%d]: %w", i, err)
		}
	}
	return nil
}

// ImplicitGoals represents goals inferred from context.
type ImplicitGoals struct {
	Inferred   []Goal  `json:"inferred" yaml:"inferred"`
	Confidence float64 `json:"confidence" yaml:"confidence"` // 0-1, confidence in inference
	Source     string  `json:"source" yaml:"source"`         // what triggered the inference
}

// Validate checks if the ImplicitGoals is valid.
func (ig *ImplicitGoals) Validate() error {
	if ig.Confidence < 0 || ig.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got: %f", ig.Confidence)
	}
	for i, g := range ig.Inferred {
		if err := g.Validate(); err != nil {
			return fmt.Errorf("inferred goal[%d]: %w", i, err)
		}
	}
	return nil
}

// GoalsDimension represents what needs to be achieved.
type GoalsDimension struct {
	Primary   PrimaryGoals   `json:"primary" yaml:"primary"`
	Secondary SecondaryGoals `json:"secondary" yaml:"secondary"`
	Implicit  ImplicitGoals  `json:"implicit" yaml:"implicit"`
	AllGoals  []Goal         `json:"all_goals" yaml:"all_goals"`
}

// Validate checks if the GoalsDimension is valid.
func (gd *GoalsDimension) Validate() error {
	if err := gd.Primary.Validate(); err != nil {
		return fmt.Errorf("primary goals: %w", err)
	}
	if err := gd.Secondary.Validate(); err != nil {
		return fmt.Errorf("secondary goals: %w", err)
	}
	if err := gd.Implicit.Validate(); err != nil {
		return fmt.Errorf("implicit goals: %w", err)
	}
	return nil
}

// ============================================================================
// Preferences Dimension (p) - 選好次元
// ============================================================================

// TradeOff represents a trade-off specification.
type TradeOff struct {
	Dimension1 string  `json:"dimension_1" yaml:"dimension_1"`
	Dimension2 string  `json:"dimension_2" yaml:"dimension_2"`
	Preference float64 `json:"preference" yaml:"preference"` // -1 to 1, negative favors dimension1
	Reason     string  `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// Validate checks if the TradeOff is valid.
func (t *TradeOff) Validate() error {
	if t.Preference < -1 || t.Preference > 1 {
		return fmt.Errorf("preference must be between -1 and 1, got: %f", t.Preference)
	}
	return nil
}

// QualitySpeedBias represents quality vs speed preference bias.
type QualitySpeedBias string

const (
	QualitySpeedBiasQuality  QualitySpeedBias = "quality"
	QualitySpeedBiasBalanced QualitySpeedBias = "balanced"
	QualitySpeedBiasSpeed    QualitySpeedBias = "speed"
)

// QualitySpeedPreference represents quality vs speed preference.
type QualitySpeedPreference struct {
	Bias             QualitySpeedBias `json:"bias" yaml:"bias"`
	QualityThreshold float64          `json:"quality_threshold" yaml:"quality_threshold"` // minimum acceptable quality (0-100)
	SpeedMultiplier  float64          `json:"speed_multiplier" yaml:"speed_multiplier"`   // 1.0 = normal, 2.0 = twice as fast
	AllowDegradation bool             `json:"allow_degradation" yaml:"allow_degradation"` // can quality be sacrificed for speed
}

// Validate checks if the QualitySpeedPreference is valid.
func (qsp *QualitySpeedPreference) Validate() error {
	if qsp.QualityThreshold < 0 || qsp.QualityThreshold > 100 {
		return fmt.Errorf("quality_threshold must be between 0 and 100, got: %f", qsp.QualityThreshold)
	}
	if qsp.SpeedMultiplier <= 0 {
		return fmt.Errorf("speed_multiplier must be positive, got: %f", qsp.SpeedMultiplier)
	}
	return nil
}

// CostPerformanceBias represents cost vs performance preference bias.
type CostPerformanceBias string

const (
	CostPerformanceBiasCost        CostPerformanceBias = "cost"
	CostPerformanceBiasBalanced    CostPerformanceBias = "balanced"
	CostPerformanceBiasPerformance CostPerformanceBias = "performance"
)

// CostPerformancePreference represents cost vs performance preference.
type CostPerformancePreference struct {
	Bias             CostPerformanceBias `json:"bias" yaml:"bias"`
	BudgetLimit      *float64            `json:"budget_limit,omitempty" yaml:"budget_limit,omitempty"` // max cost allowed
	PerformanceFloor float64             `json:"performance_floor" yaml:"performance_floor"`           // minimum acceptable performance
	Elasticity       float64             `json:"elasticity" yaml:"elasticity"`                         // how much extra to pay for improvement
}

// Validate checks if the CostPerformancePreference is valid.
func (cpp *CostPerformancePreference) Validate() error {
	if cpp.PerformanceFloor < 0 {
		return fmt.Errorf("performance_floor must be non-negative, got: %f", cpp.PerformanceFloor)
	}
	if cpp.BudgetLimit != nil && *cpp.BudgetLimit < 0 {
		return fmt.Errorf("budget_limit must be non-negative, got: %f", *cpp.BudgetLimit)
	}
	return nil
}

// AutomationControlBias represents automation vs control preference bias.
type AutomationControlBias string

const (
	AutomationControlBiasFullAuto AutomationControlBias = "full-auto"
	AutomationControlBiasSemiAuto AutomationControlBias = "semi-auto"
	AutomationControlBiasManual   AutomationControlBias = "manual"
)

// AutomationControlPreference represents automation vs control preference.
type AutomationControlPreference struct {
	Bias                 AutomationControlBias `json:"bias" yaml:"bias"`
	ApprovalRequired     []string              `json:"approval_required" yaml:"approval_required"`           // operations requiring approval
	AutoApproveThreshold float64               `json:"auto_approve_threshold" yaml:"auto_approve_threshold"` // quality score for auto-approval
}

// Validate checks if the AutomationControlPreference is valid.
func (acp *AutomationControlPreference) Validate() error {
	if acp.AutoApproveThreshold < 0 || acp.AutoApproveThreshold > 100 {
		return fmt.Errorf("auto_approve_threshold must be between 0 and 100, got: %f", acp.AutoApproveThreshold)
	}
	return nil
}

// RiskTolerance represents risk tolerance levels.
type RiskTolerance string

const (
	RiskToleranceAverse   RiskTolerance = "risk-averse"
	RiskToleranceModerate RiskTolerance = "moderate"
	RiskToleranceTolerant RiskTolerance = "risk-tolerant"
)

// RiskPreference represents risk tolerance levels.
type RiskPreference struct {
	Tolerance           RiskTolerance `json:"tolerance" yaml:"tolerance"`
	MaxRiskScore        float64       `json:"max_risk_score" yaml:"max_risk_score"`               // 0-100
	RequiresReviewAbove float64       `json:"requires_review_above" yaml:"requires_review_above"` // risk score threshold for review
}

// Validate checks if the RiskPreference is valid.
func (rp *RiskPreference) Validate() error {
	if rp.MaxRiskScore < 0 || rp.MaxRiskScore > 100 {
		return fmt.Errorf("max_risk_score must be between 0 and 100, got: %f", rp.MaxRiskScore)
	}
	if rp.RequiresReviewAbove < 0 || rp.RequiresReviewAbove > 100 {
		return fmt.Errorf("requires_review_above must be between 0 and 100, got: %f", rp.RequiresReviewAbove)
	}
	return nil
}

// PreferencesDimension represents how to make trade-offs.
type PreferencesDimension struct {
	QualityVsSpeed      QualitySpeedPreference      `json:"quality_vs_speed" yaml:"quality_vs_speed"`
	CostVsPerformance   CostPerformancePreference   `json:"cost_vs_performance" yaml:"cost_vs_performance"`
	AutomationVsControl AutomationControlPreference `json:"automation_vs_control" yaml:"automation_vs_control"`
	Risk                RiskPreference              `json:"risk" yaml:"risk"`
	CustomTradeOffs     []TradeOff                  `json:"custom_trade_offs" yaml:"custom_trade_offs"`
}

// Validate checks if the PreferencesDimension is valid.
func (pd *PreferencesDimension) Validate() error {
	if err := pd.QualityVsSpeed.Validate(); err != nil {
		return fmt.Errorf("quality_vs_speed: %w", err)
	}
	if err := pd.CostVsPerformance.Validate(); err != nil {
		return fmt.Errorf("cost_vs_performance: %w", err)
	}
	if err := pd.AutomationVsControl.Validate(); err != nil {
		return fmt.Errorf("automation_vs_control: %w", err)
	}
	if err := pd.Risk.Validate(); err != nil {
		return fmt.Errorf("risk: %w", err)
	}
	for i, t := range pd.CustomTradeOffs {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("custom_trade_off[%d]: %w", i, err)
		}
	}
	return nil
}

// ============================================================================
// Objectives Dimension (o) - 目的次元
// ============================================================================

// FunctionalRequirement represents a functional requirement.
type FunctionalRequirement struct {
	ID                 string       `json:"id" yaml:"id"`
	Description        string       `json:"description" yaml:"description"`
	Priority           GoalPriority `json:"priority" yaml:"priority"`
	AcceptanceCriteria []string     `json:"acceptance_criteria" yaml:"acceptance_criteria"`
	Testable           bool         `json:"testable" yaml:"testable"`
	Implemented        bool         `json:"implemented,omitempty" yaml:"implemented,omitempty"`
}

// Validate checks if the FunctionalRequirement is valid.
func (fr *FunctionalRequirement) Validate() error {
	if fr.ID == "" {
		return fmt.Errorf("functional requirement ID is required")
	}
	if fr.Description == "" {
		return fmt.Errorf("functional requirement description is required")
	}
	return nil
}

// NonFunctionalCategory represents non-functional requirement category.
type NonFunctionalCategory string

const (
	NFCategoryPerformance     NonFunctionalCategory = "performance"
	NFCategorySecurity        NonFunctionalCategory = "security"
	NFCategoryReliability     NonFunctionalCategory = "reliability"
	NFCategoryScalability     NonFunctionalCategory = "scalability"
	NFCategoryMaintainability NonFunctionalCategory = "maintainability"
	NFCategoryUsability       NonFunctionalCategory = "usability"
)

// NonFunctionalRequirement represents a non-functional requirement.
type NonFunctionalRequirement struct {
	ID          string                `json:"id" yaml:"id"`
	Category    NonFunctionalCategory `json:"category" yaml:"category"`
	Description string                `json:"description" yaml:"description"`
	Metric      string                `json:"metric" yaml:"metric"`
	Target      interface{}           `json:"target" yaml:"target"`                       // number or string
	Current     interface{}           `json:"current,omitempty" yaml:"current,omitempty"` // number or string
}

// Validate checks if the NonFunctionalRequirement is valid.
func (nfr *NonFunctionalRequirement) Validate() error {
	if nfr.ID == "" {
		return fmt.Errorf("non-functional requirement ID is required")
	}
	if nfr.Description == "" {
		return fmt.Errorf("non-functional requirement description is required")
	}
	return nil
}

// QualityAspect represents quality requirement aspect.
type QualityAspect string

const (
	QualityAspectCodeQuality   QualityAspect = "code-quality"
	QualityAspectTestCoverage  QualityAspect = "test-coverage"
	QualityAspectDocumentation QualityAspect = "documentation"
	QualityAspectAccessibility QualityAspect = "accessibility"
	QualityAspectCompliance    QualityAspect = "compliance"
)

// QualityRequirement represents a quality requirement.
type QualityRequirement struct {
	ID           string        `json:"id" yaml:"id"`
	Aspect       QualityAspect `json:"aspect" yaml:"aspect"`
	Description  string        `json:"description" yaml:"description"`
	MinimumScore float64       `json:"minimum_score" yaml:"minimum_score"` // 0-100
	TargetScore  float64       `json:"target_score" yaml:"target_score"`   // 0-100
	Mandatory    bool          `json:"mandatory" yaml:"mandatory"`
}

// Validate checks if the QualityRequirement is valid.
func (qr *QualityRequirement) Validate() error {
	if qr.ID == "" {
		return fmt.Errorf("quality requirement ID is required")
	}
	if qr.MinimumScore < 0 || qr.MinimumScore > 100 {
		return fmt.Errorf("minimum_score must be between 0 and 100, got: %f", qr.MinimumScore)
	}
	if qr.TargetScore < 0 || qr.TargetScore > 100 {
		return fmt.Errorf("target_score must be between 0 and 100, got: %f", qr.TargetScore)
	}
	return nil
}

// ConstraintType represents constraint type.
type ConstraintType string

const (
	ConstraintTypeTechnical  ConstraintType = "technical"
	ConstraintTypeBusiness   ConstraintType = "business"
	ConstraintTypeRegulatory ConstraintType = "regulatory"
	ConstraintTypeResource   ConstraintType = "resource"
)

// ConstraintImpact represents constraint impact level.
type ConstraintImpact string

const (
	ConstraintImpactBlocking ConstraintImpact = "blocking"
	ConstraintImpactLimiting ConstraintImpact = "limiting"
	ConstraintImpactAdvisory ConstraintImpact = "advisory"
)

// Constraint represents a constraint definition.
type Constraint struct {
	ID          string           `json:"id" yaml:"id"`
	Type        ConstraintType   `json:"type" yaml:"type"`
	Description string           `json:"description" yaml:"description"`
	Impact      ConstraintImpact `json:"impact" yaml:"impact"`
	Workaround  string           `json:"workaround,omitempty" yaml:"workaround,omitempty"`
}

// Validate checks if the Constraint is valid.
func (c *Constraint) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("constraint ID is required")
	}
	if c.Description == "" {
		return fmt.Errorf("constraint description is required")
	}
	return nil
}

// ObjectivesDimension represents specific requirements to satisfy.
type ObjectivesDimension struct {
	Functional    []FunctionalRequirement    `json:"functional" yaml:"functional"`
	NonFunctional []NonFunctionalRequirement `json:"non_functional" yaml:"non_functional"`
	Quality       []QualityRequirement       `json:"quality" yaml:"quality"`
	Constraints   []Constraint               `json:"constraints" yaml:"constraints"`
}

// Validate checks if the ObjectivesDimension is valid.
func (od *ObjectivesDimension) Validate() error {
	for i, fr := range od.Functional {
		if err := fr.Validate(); err != nil {
			return fmt.Errorf("functional requirement[%d]: %w", i, err)
		}
	}
	for i, nfr := range od.NonFunctional {
		if err := nfr.Validate(); err != nil {
			return fmt.Errorf("non-functional requirement[%d]: %w", i, err)
		}
	}
	for i, qr := range od.Quality {
		if err := qr.Validate(); err != nil {
			return fmt.Errorf("quality requirement[%d]: %w", i, err)
		}
	}
	for i, c := range od.Constraints {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("constraint[%d]: %w", i, err)
		}
	}
	return nil
}

// ============================================================================
// Modality Dimension (m) - 様式次元
// ============================================================================

// OutputModality represents output modality types.
type OutputModality string

const (
	OutputModalityText   OutputModality = "text"
	OutputModalityCode   OutputModality = "code"
	OutputModalityVisual OutputModality = "visual"
	OutputModalityData   OutputModality = "data"
	OutputModalityAudio  OutputModality = "audio"
	OutputModalityMixed  OutputModality = "mixed"
)

// CodeStyle represents code output style.
type CodeStyle string

const (
	CodeStyleVerbose    CodeStyle = "verbose"
	CodeStyleConcise    CodeStyle = "concise"
	CodeStyleDocumented CodeStyle = "documented"
)

// CodeModality represents code output specification.
type CodeModality struct {
	Language      string    `json:"language" yaml:"language"`
	Framework     string    `json:"framework,omitempty" yaml:"framework,omitempty"`
	Style         CodeStyle `json:"style" yaml:"style"`
	IncludeTests  bool      `json:"include_tests" yaml:"include_tests"`
	IncludeTypes  bool      `json:"include_types" yaml:"include_types"`
	TargetVersion string    `json:"target_version,omitempty" yaml:"target_version,omitempty"`
}

// Validate checks if the CodeModality is valid.
func (cm *CodeModality) Validate() error {
	if cm.Language == "" {
		return fmt.Errorf("code modality language is required")
	}
	return nil
}

// TextFormat represents text output format.
type TextFormat string

const (
	TextFormatMarkdown TextFormat = "markdown"
	TextFormatPlain    TextFormat = "plain"
	TextFormatHTML     TextFormat = "html"
	TextFormatJSON     TextFormat = "json"
	TextFormatYAML     TextFormat = "yaml"
)

// TextTone represents text output tone.
type TextTone string

const (
	TextToneFormal    TextTone = "formal"
	TextToneCasual    TextTone = "casual"
	TextToneTechnical TextTone = "technical"
)

// TextModality represents text output specification.
type TextModality struct {
	Format    TextFormat `json:"format" yaml:"format"`
	Language  string     `json:"language" yaml:"language"` // natural language (en, ja, etc.)
	Tone      TextTone   `json:"tone" yaml:"tone"`
	MaxLength *int       `json:"max_length,omitempty" yaml:"max_length,omitempty"`
}

// Validate checks if the TextModality is valid.
func (tm *TextModality) Validate() error {
	if tm.Language == "" {
		return fmt.Errorf("text modality language is required")
	}
	return nil
}

// VisualFormat represents visual output format.
type VisualFormat string

const (
	VisualFormatDiagram  VisualFormat = "diagram"
	VisualFormatChart    VisualFormat = "chart"
	VisualFormatImage    VisualFormat = "image"
	VisualFormatUIMockup VisualFormat = "ui-mockup"
)

// VisualModality represents visual output specification.
type VisualModality struct {
	Format     VisualFormat `json:"format" yaml:"format"`
	Tool       string       `json:"tool,omitempty" yaml:"tool,omitempty"` // PlantUML, Mermaid, etc.
	Style      string       `json:"style,omitempty" yaml:"style,omitempty"`
	Resolution string       `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

// Validate checks if the VisualModality is valid.
func (vm *VisualModality) Validate() error {
	return nil
}

// DataFormat represents data output format.
type DataFormat string

const (
	DataFormatJSON   DataFormat = "json"
	DataFormatYAML   DataFormat = "yaml"
	DataFormatCSV    DataFormat = "csv"
	DataFormatSQL    DataFormat = "sql"
	DataFormatBinary DataFormat = "binary"
)

// DataModality represents data output specification.
type DataModality struct {
	Format      DataFormat `json:"format" yaml:"format"`
	Schema      string     `json:"schema,omitempty" yaml:"schema,omitempty"` // reference to schema
	Validation  bool       `json:"validation" yaml:"validation"`
	Compression string     `json:"compression,omitempty" yaml:"compression,omitempty"`
}

// Validate checks if the DataModality is valid.
func (dm *DataModality) Validate() error {
	return nil
}

// ModalityDimension represents how outputs should be formatted.
type ModalityDimension struct {
	Primary   OutputModality   `json:"primary" yaml:"primary"`
	Secondary []OutputModality `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	Code      *CodeModality    `json:"code,omitempty" yaml:"code,omitempty"`
	Text      *TextModality    `json:"text,omitempty" yaml:"text,omitempty"`
	Visual    *VisualModality  `json:"visual,omitempty" yaml:"visual,omitempty"`
	Data      *DataModality    `json:"data,omitempty" yaml:"data,omitempty"`
}

// Validate checks if the ModalityDimension is valid.
func (md *ModalityDimension) Validate() error {
	if md.Code != nil {
		if err := md.Code.Validate(); err != nil {
			return fmt.Errorf("code modality: %w", err)
		}
	}
	if md.Text != nil {
		if err := md.Text.Validate(); err != nil {
			return fmt.Errorf("text modality: %w", err)
		}
	}
	if md.Visual != nil {
		if err := md.Visual.Validate(); err != nil {
			return fmt.Errorf("visual modality: %w", err)
		}
	}
	if md.Data != nil {
		if err := md.Data.Validate(); err != nil {
			return fmt.Errorf("data modality: %w", err)
		}
	}
	return nil
}

// ============================================================================
// Intent Space - Complete Definition
// ============================================================================

// IntentSource represents the source of intent.
type IntentSource string

const (
	IntentSourceUser     IntentSource = "user"
	IntentSourceSystem   IntentSource = "system"
	IntentSourceAgent    IntentSource = "agent"
	IntentSourceInferred IntentSource = "inferred"
)

// IntentMetadata represents intent metadata.
type IntentMetadata struct {
	IntentID   string       `json:"intent_id" yaml:"intent_id"`
	Source     IntentSource `json:"source" yaml:"source"`
	CreatedAt  string       `json:"created_at" yaml:"created_at"`
	ExpiresAt  string       `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Confidence float64      `json:"confidence" yaml:"confidence"` // 0-1
	Version    int          `json:"version" yaml:"version"`
}

// Validate checks if the IntentMetadata is valid.
func (im *IntentMetadata) Validate() error {
	if im.IntentID == "" {
		return fmt.Errorf("intent_id is required")
	}
	if im.Confidence < 0 || im.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got: %f", im.Confidence)
	}
	if im.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339, im.CreatedAt); err != nil {
			return fmt.Errorf("invalid created_at format: %w", err)
		}
	}
	if im.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, im.ExpiresAt); err != nil {
			return fmt.Errorf("invalid expires_at format: %w", err)
		}
	}
	return nil
}

// IntentSpace represents complete intention specification.
// Mathematical representation: I(g, p, o, m) → Objective
type IntentSpace struct {
	// Intent metadata
	Metadata IntentMetadata `json:"metadata" yaml:"metadata"`

	// g: Goals Dimension - 目標次元
	Goals GoalsDimension `json:"goals" yaml:"goals"`

	// p: Preferences Dimension - 選好次元
	Preferences PreferencesDimension `json:"preferences" yaml:"preferences"`

	// o: Objectives Dimension - 目的次元
	Objectives ObjectivesDimension `json:"objectives" yaml:"objectives"`

	// m: Modality Dimension - 様式次元
	Modality ModalityDimension `json:"modality" yaml:"modality"`
}

// Validate checks if the IntentSpace is valid.
func (is *IntentSpace) Validate() error {
	if err := is.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	if err := is.Goals.Validate(); err != nil {
		return fmt.Errorf("goals: %w", err)
	}
	if err := is.Preferences.Validate(); err != nil {
		return fmt.Errorf("preferences: %w", err)
	}
	if err := is.Objectives.Validate(); err != nil {
		return fmt.Errorf("objectives: %w", err)
	}
	if err := is.Modality.Validate(); err != nil {
		return fmt.Errorf("modality: %w", err)
	}
	return nil
}

// ToJSON serializes IntentSpace to JSON.
func (is *IntentSpace) ToJSON() ([]byte, error) {
	return json.MarshalIndent(is, "", "  ")
}

// FromJSON deserializes IntentSpace from JSON.
func (is *IntentSpace) FromJSON(data []byte) error {
	return json.Unmarshal(data, is)
}

// ============================================================================
// Intent Processing
// ============================================================================

// ValidationSeverity represents validation error severity.
type ValidationSeverity string

const (
	ValidationError   ValidationSeverity = "error"
	ValidationWarning ValidationSeverity = "warning"
)

// ValidationErrorItem represents a validation error.
type ValidationErrorItem struct {
	Field    string             `json:"field" yaml:"field"`
	Message  string             `json:"message" yaml:"message"`
	Severity ValidationSeverity `json:"severity" yaml:"severity"`
}

// IntentValidation represents intent validation result.
type IntentValidation struct {
	Valid       bool                  `json:"valid" yaml:"valid"`
	Errors      []ValidationErrorItem `json:"errors" yaml:"errors"`
	Suggestions []string              `json:"suggestions" yaml:"suggestions"`
}

// IntentRefinement represents intent refinement suggestion.
type IntentRefinement struct {
	Original   map[string]interface{} `json:"original" yaml:"original"`
	Refined    map[string]interface{} `json:"refined" yaml:"refined"`
	Reason     string                 `json:"reason" yaml:"reason"`
	Confidence float64                `json:"confidence" yaml:"confidence"`
}

// ValidateIntent validates an IntentSpace and returns the validation result.
func ValidateIntent(intent *IntentSpace) *IntentValidation {
	result := &IntentValidation{
		Valid:       true,
		Errors:      []ValidationErrorItem{},
		Suggestions: []string{},
	}

	if err := intent.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationErrorItem{
			Field:    "intent",
			Message:  err.Error(),
			Severity: ValidationError,
		})
	}

	// Add suggestions for improvement
	if intent.Metadata.Confidence < 0.5 {
		result.Suggestions = append(result.Suggestions,
			"Consider providing more context to improve confidence score")
	}

	if len(intent.Objectives.Functional) == 0 {
		result.Suggestions = append(result.Suggestions,
			"Consider adding functional requirements for clearer objectives")
	}

	return result
}
