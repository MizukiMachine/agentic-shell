package spec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/MizukiMachine/agentic-shell/internal/llm"
	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

const maxGatherRounds = 5
const dynamicConfidenceFloor = 0.90

// AgentSpec は types.AgentSpec の別名です。
type AgentSpec = types.AgentSpec

// TimeoutError はタイムアウトの種類とヒントを含むエラー型です。
type TimeoutError struct {
	Type    string // "input" or "total"
	Message string
	Hint    string
}

// Error は error インターフェースを実装します。
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("%s\nヒント: %s", e.Message, e.Hint)
}

// Gatherer は対話入力から AgentSpec を段階的に構築します。
type Gatherer struct {
	input               io.Reader
	reader              *bufio.Reader
	output              io.Writer
	calculator          *ConfidenceCalculator
	interpreter         *Interpreter
	maxRounds           int
	confidenceThreshold float64
	inputTimeout        time.Duration
	useLLMQuestions     bool
	now                 func() time.Time
}

type llmTurnError struct {
	err error
}

func (e *llmTurnError) Error() string {
	return e.err.Error()
}

func (e *llmTurnError) Unwrap() error {
	return e.err
}

// NewGatherer は対話収集に使う Gatherer を生成します。
func NewGatherer(input io.Reader, output io.Writer) *Gatherer {
	var reader *bufio.Reader
	if input != nil {
		reader = bufio.NewReader(input)
	}

	return &Gatherer{
		input:               input,
		reader:              reader,
		output:              output,
		calculator:          &ConfidenceCalculator{},
		maxRounds:           maxGatherRounds,
		confidenceThreshold: ConfidenceThreshold,
		now:                 time.Now,
	}
}

// SetMaxRounds は最大質問ラウンド数を設定します。
func (g *Gatherer) SetMaxRounds(maxRounds int) {
	g.maxRounds = maxRounds
}

// SetConfidenceThreshold は採用信頼度の閾値を設定します。
func (g *Gatherer) SetConfidenceThreshold(threshold float64) {
	g.confidenceThreshold = threshold
}

// SetInputTimeout はユーザー入力待ちのタイムアウトを設定します。
// 0 を設定した場合は無制限（タイムアウトなし）となります。
func (g *Gatherer) SetInputTimeout(timeout time.Duration) {
	g.inputTimeout = timeout
}

// SetUseLLMQuestions は LLM による動的質問生成を有効化します。
func (g *Gatherer) SetUseLLMQuestions(useLLM bool) {
	g.useLLMQuestions = useLLM
}

// SetLLMClient は動的質問生成に使う LLM クライアントを設定します。
func (g *Gatherer) SetLLMClient(client llm.Client) {
	if client == nil {
		g.interpreter = nil
		return
	}
	g.interpreter = NewInterpreter(client)
}

// SetInterpreter はテストや差し替え用に Interpreter を設定します。
func (g *Gatherer) SetInterpreter(interpreter *Interpreter) {
	g.interpreter = interpreter
}

// GatherInteractive は質問と回答を通じて AgentSpec を補完します。
func (g *Gatherer) GatherInteractive(ctx context.Context, initialInput string) (*AgentSpec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	g.ensureDefaults()

	trimmedInput := strings.TrimSpace(initialInput)
	if trimmedInput == "" {
		return nil, fmt.Errorf("initial input is required")
	}

	spec := g.buildInitialSpec(trimmedInput)
	threshold := g.confidenceThreshold
	if g.useLLMQuestions {
		threshold = maxFloat(g.confidenceThreshold, dynamicConfidenceFloor)
		spec.Intent.Metadata.Confidence = g.calculator.Calculate(spec)
		if spec.Intent.Metadata.Confidence >= threshold {
			if err := ValidateWithThreshold(spec, threshold); err != nil {
				return spec, err
			}
			return spec, nil
		}

		dynamicSpec, err := g.gatherDynamic(ctx, trimmedInput, spec, threshold)
		if err == nil {
			return dynamicSpec, nil
		}

		var llmErr *llmTurnError
		if errors.As(err, &llmErr) {
			if _, writeErr := fmt.Fprintf(g.output, "LLM質問生成に失敗したため固定質問にフォールバックします: %v\n", llmErr.err); writeErr != nil {
				return nil, writeErr
			}
			return g.gatherStepBack(ctx, trimmedInput, spec, threshold)
		}

		return nil, err
	}

	return g.gatherStepBack(ctx, trimmedInput, spec, threshold)
}

func (g *Gatherer) gatherStepBack(ctx context.Context, initialInput string, spec *AgentSpec, threshold float64) (*AgentSpec, error) {
	questions := generateStepBackQuestions(initialInput)
	roundLimit := min(g.maxRounds, len(questions))

	for round := 0; round < roundLimit; round++ {
		confidence := g.calculator.Calculate(spec)
		spec.Intent.Metadata.Confidence = confidence
		if confidence >= threshold {
			if err := ValidateWithThreshold(spec, threshold); err != nil {
				return spec, err
			}
			return spec, nil
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		question := questions[round]
		if _, err := fmt.Fprintf(g.output, "[%d/%d] %s\n> ", round+1, roundLimit, question); err != nil {
			return nil, err
		}

		answer, err := g.readLine(ctx)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(answer) == "" {
			continue
		}

		g.applyStepBackResponse(spec, round, answer)
	}

	spec.Intent.Metadata.Confidence = g.calculator.Calculate(spec)
	if err := ValidateWithThreshold(spec, threshold); err != nil {
		return spec, err
	}

	return spec, nil
}

func (g *Gatherer) gatherDynamic(ctx context.Context, initialInput string, spec *AgentSpec, threshold float64) (*AgentSpec, error) {
	if g.interpreter == nil {
		return nil, &llmTurnError{err: fmt.Errorf("llm interpreter is not configured")}
	}

	state := NewConversationState(initialInput, spec)
	state.Confidence = g.calculator.Calculate(spec)

	response, err := g.interpreter.ProcessTurn(ctx, state)
	if err != nil {
		return nil, &llmTurnError{err: err}
	}

	for round := 0; round < g.maxRounds; round++ {
		if state.Confidence >= threshold {
			if err := ValidateWithThreshold(spec, threshold); err != nil {
				return spec, err
			}
			return spec, nil
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := g.displayTurnOutput(response, round+1); err != nil {
			return nil, err
		}

		answer, err := g.readLine(ctx)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(answer) == "" {
			continue
		}

		state.AddTurn(response.NextQuestion.Prompt, response.NextQuestion.Options, answer)
		g.applyDynamicAnswer(spec, answer)
		state.Confidence = g.calculator.Calculate(spec)
		spec.Intent.Metadata.Confidence = state.Confidence

		response, err = g.interpreter.ProcessTurn(ctx, state)
		if err != nil {
			return nil, &llmTurnError{err: err}
		}
	}

	spec.Intent.Metadata.Confidence = g.calculator.Calculate(spec)
	if err := ValidateWithThreshold(spec, threshold); err != nil {
		return spec, err
	}

	return spec, nil
}

func (g *Gatherer) displayTurnOutput(response *TurnResponse, questionNumber int) error {
	if response == nil {
		return fmt.Errorf("turn response is required")
	}

	if _, err := fmt.Fprintln(g.output, "=== Current Understanding ==="); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(g.output, "Core Intent: %s\n", response.CurrentUnderstanding.CoreIntent); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(g.output, "Primary Goal: %s\n", response.CurrentUnderstanding.PrimaryGoal); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(g.output, "Success Criteria:"); err != nil {
		return err
	}
	for _, criterion := range response.CurrentUnderstanding.SuccessCriteria {
		if _, err := fmt.Fprintf(g.output, "  - %s\n", criterion); err != nil {
			return err
		}
	}
	if len(response.CurrentUnderstanding.SuccessCriteria) == 0 {
		if _, err := fmt.Fprintln(g.output, "  - (none yet)"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(g.output, "\n=== Suggestions ==="); err != nil {
		return err
	}
	for idx, suggestion := range response.Suggestions {
		if _, err := fmt.Fprintf(g.output, "%d. [%s] %s\n", idx+1, nonEmpty(suggestion.Category, "General"), suggestion.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(g.output, "   %s\n", suggestion.Description); err != nil {
			return err
		}
	}
	if len(response.Suggestions) == 0 {
		if _, err := fmt.Fprintln(g.output, "1. [General] No additional suggestions"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(g.output, "\n=== Question [%d] ===\n", questionNumber); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(g.output, "%s\n", response.NextQuestion.Prompt); err != nil {
		return err
	}
	if len(response.NextQuestion.Options) > 0 {
		if _, err := fmt.Fprintln(g.output, "Options:"); err != nil {
			return err
		}
		for idx, option := range response.NextQuestion.Options {
			if _, err := fmt.Fprintf(g.output, "  %d. %s\n", idx+1, option); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprint(g.output, "> ")
	return err
}

func generateStepBackQuestions(input string) []string {
	focus := questionFocus(input)

	questions := []string{
		"What is the core problem we are trying to solve?",
		"Why is this goal important in the bigger picture?",
		"What are the fundamental principles guiding this work?",
		"What would the ideal solution look like?",
		"How does this connect to our broader objectives?",
	}

	if focus == "" {
		return questions
	}

	return []string{
		fmt.Sprintf("What is the core problem we are trying to solve for %s?", focus),
		fmt.Sprintf("Why is %s important in the bigger picture?", focus),
		fmt.Sprintf("What principles should guide the %s work?", focus),
		fmt.Sprintf("What would the ideal %s solution look like?", focus),
		fmt.Sprintf("How does %s connect to our broader objectives?", focus),
	}
}

func (g *Gatherer) ensureDefaults() {
	if g.input == nil {
		g.input = os.Stdin
	}
	if g.output == nil {
		g.output = os.Stdout
	}
	if g.reader == nil {
		g.reader = bufio.NewReader(g.input)
	}
	if g.calculator == nil {
		g.calculator = &ConfidenceCalculator{}
	}
	if g.maxRounds <= 0 {
		g.maxRounds = maxGatherRounds
	}
	if g.confidenceThreshold <= 0 || g.confidenceThreshold > 1 {
		g.confidenceThreshold = ConfidenceThreshold
	}
	if g.now == nil {
		g.now = time.Now
	}
}

func (g *Gatherer) buildInitialSpec(initialInput string) *AgentSpec {
	spec := types.NewAgentSpec(inferName(initialInput), "1.0.0")
	now := g.now().UTC()
	mainGoal := types.Goal{
		ID:          "goal-main",
		Type:        types.GoalTypePrimary,
		Description: initialInput,
		Priority:    types.GoalPriorityHigh,
		Measurable:  true,
	}

	spec.Metadata.Description = initialInput
	spec.Intent = types.IntentSpace{
		Metadata: types.IntentMetadata{
			IntentID:   fmt.Sprintf("intent-%d", now.UnixNano()),
			Source:     types.IntentSourceUser,
			CreatedAt:  now.Format(time.RFC3339),
			Confidence: 0,
			Version:    1,
		},
		Goals: types.GoalsDimension{
			Primary: types.PrimaryGoals{
				Main: mainGoal,
			},
			AllGoals: []types.Goal{mainGoal},
		},
		Preferences: types.PreferencesDimension{
			QualityVsSpeed: types.QualitySpeedPreference{
				SpeedMultiplier: 1.0,
			},
			CostVsPerformance: types.CostPerformancePreference{
				Elasticity: 1.0,
			},
			AutomationVsControl: types.AutomationControlPreference{},
			Risk:                types.RiskPreference{},
		},
		Objectives: types.ObjectivesDimension{
			Functional: []types.FunctionalRequirement{
				{
					ID:          "fr-1",
					Description: initialInput,
					Priority:    types.GoalPriorityHigh,
					AcceptanceCriteria: []string{
						"Specification can be validated without manual reconstruction",
					},
					Testable: true,
				},
			},
		},
		Modality: types.ModalityDimension{
			Primary:   types.OutputModalityData,
			Secondary: []types.OutputModality{types.OutputModalityText},
			Data: &types.DataModality{
				Format:     inferDataFormat(initialInput),
				Validation: true,
			},
			Text: &types.TextModality{
				Format:   types.TextFormatMarkdown,
				Language: "en",
				Tone:     types.TextToneTechnical,
			},
		},
	}

	return spec
}

func (g *Gatherer) readLine(ctx context.Context) (string, error) {
	type lineResult struct {
		line string
		err  error
	}

	resultCh := make(chan lineResult, 1)

	// bufio.Reader の ReadString はキャンセル不能なので goroutine で分離し、ctx を優先監視する。
	go func() {
		line, err := g.reader.ReadString('\n')
		resultCh <- lineResult{
			line: strings.TrimSpace(line),
			err:  normalizeReadLineError(line, err),
		}
	}()

	// 入力タイムアウトと全体タイムアウトを別個に監視
	// inputTimeout が 0 の場合は無制限
	var inputTimer <-chan time.Time
	if g.inputTimeout > 0 {
		inputTimer = time.After(g.inputTimeout)
	}

	select {
	case <-ctx.Done():
		return "", &TimeoutError{
			Type:    "total",
			Message: fmt.Sprintf("全体処理がタイムアウトしました (%v)", ctx.Err()),
			Hint:    "--timeout フラグで延長できます",
		}
	case <-inputTimer:
		return "", &TimeoutError{
			Type:    "input",
			Message: fmt.Sprintf("入力待ちタイムアウト (%v経過)", g.inputTimeout),
			Hint:    "--input-timeout フラグで延長、または0で無制限にできます",
		}
	case result := <-resultCh:
		return result.line, result.err
	}
}

func (g *Gatherer) applyStepBackResponse(spec *AgentSpec, round int, answer string) {
	switch round {
	case 0:
		g.applyCoreProblem(spec, answer)
	case 1:
		g.applyBiggerPicture(spec, answer)
	case 2:
		g.applyPrinciples(spec, answer)
	case 3:
		g.applyIdealSolution(spec, answer)
	case 4:
		g.applyBroaderObjectives(spec, answer)
	}
}

func (g *Gatherer) applyCoreProblem(spec *AgentSpec, answer string) {
	spec.Metadata.Description = mergeNarrative(spec.Metadata.Description, "Core problem: "+answer)
	spec.Intent.Goals.Primary.Main.Description = answer
	spec.Intent.Goals.Primary.Main.SuccessCriteria = appendUnique(
		spec.Intent.Goals.Primary.Main.SuccessCriteria,
		"Core problem is stated in a way that engineering work can begin",
	)
	g.syncPrimaryMainGoal(spec)

	if len(spec.Intent.Objectives.Functional) == 0 {
		spec.Intent.Objectives.Functional = append(spec.Intent.Objectives.Functional, types.FunctionalRequirement{
			ID:          "fr-1",
			Description: answer,
			Priority:    types.GoalPriorityHigh,
			Testable:    true,
		})
	} else {
		spec.Intent.Objectives.Functional[0].Description = answer
		spec.Intent.Objectives.Functional[0].AcceptanceCriteria = appendUnique(
			spec.Intent.Objectives.Functional[0].AcceptanceCriteria,
			"Core problem is captured explicitly",
		)
	}
}

func (g *Gatherer) syncPrimaryMainGoal(spec *AgentSpec) {
	if len(spec.Intent.Goals.AllGoals) == 0 {
		spec.Intent.Goals.AllGoals = append(spec.Intent.Goals.AllGoals, spec.Intent.Goals.Primary.Main)
		return
	}
	spec.Intent.Goals.AllGoals[0] = spec.Intent.Goals.Primary.Main
}

func (g *Gatherer) applyBiggerPicture(spec *AgentSpec, answer string) {
	supportingGoal := types.Goal{
		ID:          fmt.Sprintf("goal-support-%d", len(spec.Intent.Goals.Primary.Supporting)+1),
		Type:        types.GoalTypeSecondary,
		Description: answer,
		Priority:    types.GoalPriorityMedium,
		Measurable:  false,
	}

	spec.Intent.Goals.Primary.Supporting = append(spec.Intent.Goals.Primary.Supporting, supportingGoal)
	spec.Intent.Goals.AllGoals = append(spec.Intent.Goals.AllGoals, supportingGoal)
	spec.Metadata.Description = mergeNarrative(spec.Metadata.Description, "Importance: "+answer)
	spec.Metadata.Tags = appendUnique(spec.Metadata.Tags, extractKeywords(answer)...)
}

func (g *Gatherer) applyPrinciples(spec *AgentSpec, answer string) {
	lowerAnswer := strings.ToLower(answer)

	switch {
	case containsAny(lowerAnswer, "quality", "correct", "accuracy", "reliable", "safe", "safety", "secure"):
		spec.Intent.Preferences.QualityVsSpeed.Bias = types.QualitySpeedBiasQuality
		spec.Intent.Preferences.QualityVsSpeed.QualityThreshold = 90
		spec.Intent.Preferences.QualityVsSpeed.SpeedMultiplier = 1.0
	case containsAny(lowerAnswer, "speed", "fast", "quick", "rapid"):
		spec.Intent.Preferences.QualityVsSpeed.Bias = types.QualitySpeedBiasSpeed
		spec.Intent.Preferences.QualityVsSpeed.QualityThreshold = 60
		spec.Intent.Preferences.QualityVsSpeed.SpeedMultiplier = 1.5
	default:
		spec.Intent.Preferences.QualityVsSpeed.Bias = types.QualitySpeedBiasBalanced
		spec.Intent.Preferences.QualityVsSpeed.QualityThreshold = 80
		spec.Intent.Preferences.QualityVsSpeed.SpeedMultiplier = 1.0
	}

	switch {
	case containsAny(lowerAnswer, "budget", "cost", "efficient"):
		spec.Intent.Preferences.CostVsPerformance.Bias = types.CostPerformanceBiasCost
		spec.Intent.Preferences.CostVsPerformance.PerformanceFloor = 60
		spec.Intent.Preferences.CostVsPerformance.Elasticity = 0.2
	case containsAny(lowerAnswer, "performance", "throughput", "latency"):
		spec.Intent.Preferences.CostVsPerformance.Bias = types.CostPerformanceBiasPerformance
		spec.Intent.Preferences.CostVsPerformance.PerformanceFloor = 85
		spec.Intent.Preferences.CostVsPerformance.Elasticity = 0.8
	default:
		spec.Intent.Preferences.CostVsPerformance.Bias = types.CostPerformanceBiasBalanced
		spec.Intent.Preferences.CostVsPerformance.PerformanceFloor = 70
		spec.Intent.Preferences.CostVsPerformance.Elasticity = 0.5
	}

	switch {
	case containsAny(lowerAnswer, "manual", "human", "review", "approval"):
		spec.Intent.Preferences.AutomationVsControl.Bias = types.AutomationControlBiasManual
		spec.Intent.Preferences.AutomationVsControl.ApprovalRequired = appendUnique(
			spec.Intent.Preferences.AutomationVsControl.ApprovalRequired,
			"destructive changes",
			"production updates",
		)
		spec.Intent.Preferences.AutomationVsControl.AutoApproveThreshold = 0
	default:
		spec.Intent.Preferences.AutomationVsControl.Bias = types.AutomationControlBiasSemiAuto
		spec.Intent.Preferences.AutomationVsControl.ApprovalRequired = appendUnique(
			spec.Intent.Preferences.AutomationVsControl.ApprovalRequired,
			"high-risk operations",
		)
		spec.Intent.Preferences.AutomationVsControl.AutoApproveThreshold = 85
	}

	switch {
	case containsAny(lowerAnswer, "safe", "safety", "security", "compliance", "reliable", "stability"):
		spec.Intent.Preferences.Risk.Tolerance = types.RiskToleranceAverse
		spec.Intent.Preferences.Risk.MaxRiskScore = 20
		spec.Intent.Preferences.Risk.RequiresReviewAbove = 10
	case containsAny(lowerAnswer, "experimental", "aggressive", "move fast"):
		spec.Intent.Preferences.Risk.Tolerance = types.RiskToleranceTolerant
		spec.Intent.Preferences.Risk.MaxRiskScore = 70
		spec.Intent.Preferences.Risk.RequiresReviewAbove = 60
	default:
		spec.Intent.Preferences.Risk.Tolerance = types.RiskToleranceModerate
		spec.Intent.Preferences.Risk.MaxRiskScore = 40
		spec.Intent.Preferences.Risk.RequiresReviewAbove = 25
	}

	spec.Intent.Preferences.CustomTradeOffs = append(spec.Intent.Preferences.CustomTradeOffs, types.TradeOff{
		Dimension1: "quality",
		Dimension2: "speed",
		Preference: tradeOffPreference(spec.Intent.Preferences.QualityVsSpeed.Bias),
		Reason:     answer,
	})

	spec.Intent.Objectives.Constraints = append(spec.Intent.Objectives.Constraints, types.Constraint{
		ID:          fmt.Sprintf("constraint-%d", len(spec.Intent.Objectives.Constraints)+1),
		Type:        types.ConstraintTypeTechnical,
		Description: answer,
		Impact:      types.ConstraintImpactAdvisory,
	})
}

func (g *Gatherer) applyIdealSolution(spec *AgentSpec, answer string) {
	spec.Capabilities = append(spec.Capabilities, types.Capability{
		ID:          fmt.Sprintf("cap-%d", len(spec.Capabilities)+1),
		Name:        "Solution Design",
		Description: answer,
		Category:    "specification",
		Level:       "expert",
		Keywords:    extractKeywords(answer),
	})

	spec.Skills = append(spec.Skills, types.Skill{
		ID:          fmt.Sprintf("skill-%d", len(spec.Skills)+1),
		Name:        "Spec Refinement",
		Description: answer,
		Complexity:  "medium",
	})

	spec.Tools = append(spec.Tools, types.Tool{
		ID:          fmt.Sprintf("tool-%d", len(spec.Tools)+1),
		Name:        inferToolName(answer),
		Description: answer,
		Category:    "processing",
		RiskLevel:   "low",
	})

	spec.Intent.Objectives.NonFunctional = append(spec.Intent.Objectives.NonFunctional, types.NonFunctionalRequirement{
		ID:          fmt.Sprintf("nfr-%d", len(spec.Intent.Objectives.NonFunctional)+1),
		Category:    types.NFCategoryMaintainability,
		Description: answer,
		Metric:      "solution-shape",
		Target:      "clear and operable",
	})

	spec.Intent.Objectives.Quality = append(spec.Intent.Objectives.Quality, types.QualityRequirement{
		ID:           fmt.Sprintf("qr-%d", len(spec.Intent.Objectives.Quality)+1),
		Aspect:       types.QualityAspectCodeQuality,
		Description:  "The gathered specification should be implementation-ready",
		MinimumScore: 85,
		TargetScore:  95,
		Mandatory:    true,
	})

	lowerAnswer := strings.ToLower(answer)
	if containsAny(lowerAnswer, "cli", "command line", "terminal") {
		spec.Communication.Type = "cli"
	}
	if containsAny(lowerAnswer, "api", "service", "http") {
		spec.Communication.Type = "rest"
	}
	if containsAny(lowerAnswer, "json") {
		spec.Intent.Modality.Data = &types.DataModality{
			Format:     types.DataFormatJSON,
			Validation: true,
		}
	}
	if containsAny(lowerAnswer, "yaml") {
		spec.Intent.Modality.Data = &types.DataModality{
			Format:     types.DataFormatYAML,
			Validation: true,
		}
	}
	if containsAny(lowerAnswer, "go", "golang", "package") {
		spec.Intent.Modality.Code = &types.CodeModality{
			Language:     "go",
			Style:        types.CodeStyleDocumented,
			IncludeTests: true,
			IncludeTypes: true,
		}
	}
}

func (g *Gatherer) applyBroaderObjectives(spec *AgentSpec, answer string) {
	spec.Intent.Goals.Primary.Main.SuccessCriteria = appendUnique(
		spec.Intent.Goals.Primary.Main.SuccessCriteria,
		answer,
	)

	goal := types.Goal{
		ID:          fmt.Sprintf("goal-broader-%d", len(spec.Intent.Goals.Primary.Supporting)+1),
		Type:        types.GoalTypeDerived,
		Description: answer,
		Priority:    types.GoalPriorityMedium,
		Measurable:  false,
	}
	spec.Intent.Goals.Primary.Supporting = append(spec.Intent.Goals.Primary.Supporting, goal)
	spec.Intent.Goals.AllGoals = append(spec.Intent.Goals.AllGoals, goal)
	spec.Metadata.Tags = appendUnique(spec.Metadata.Tags, extractKeywords(answer)...)
}

func (g *Gatherer) applyDynamicAnswer(spec *AgentSpec, answer string) {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return
	}

	g.applyBiggerPicture(spec, trimmed)
	g.applyPrinciples(spec, trimmed)
	g.applyIdealSolution(spec, trimmed)
	g.applyBroaderObjectives(spec, trimmed)
}

func inferName(input string) string {
	words := extractASCIIWords(strings.ToLower(input))
	if len(words) == 0 {
		return "agent-spec"
	}
	if len(words) > 4 {
		words = words[:4]
	}
	return strings.Join(words, "-")
}

func inferDataFormat(input string) types.DataFormat {
	lowerInput := strings.ToLower(input)
	if containsAny(lowerInput, "json") && !containsAny(lowerInput, "yaml", "yml") {
		return types.DataFormatJSON
	}
	return types.DataFormatYAML
}

func inferToolName(answer string) string {
	lowerAnswer := strings.ToLower(answer)
	switch {
	case containsAny(lowerAnswer, "yaml", "json", "serialize", "export"):
		return "Structured Exporter"
	case containsAny(lowerAnswer, "interactive", "question", "dialog"):
		return "Interactive Prompt"
	case containsAny(lowerAnswer, "validate", "check"):
		return "Specification Validator"
	default:
		return "Specification Processor"
	}
}

func questionFocus(input string) string {
	genericWords := map[string]struct{}{
		"a": {}, "an": {}, "agent": {}, "build": {}, "create": {}, "feature": {},
		"implement": {}, "make": {}, "support": {}, "system": {}, "tool": {},
	}

	focus := make([]string, 0, 2)
	for _, keyword := range extractKeywords(input) {
		if _, skip := genericWords[keyword]; skip {
			continue
		}
		if len(keyword) <= 2 {
			continue
		}
		focus = append(focus, keyword)
		if len(focus) == 2 {
			break
		}
	}

	return strings.Join(focus, " ")
}

func extractKeywords(input string) []string {
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "this": {}, "that": {},
		"from": {}, "into": {}, "our": {}, "are": {}, "should": {}, "will": {},
	}

	keywords := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, token := range strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := stopWords[token]; ok {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		keywords = append(keywords, token)
		if len(keywords) == 4 {
			break
		}
	}
	return keywords
}

func extractASCIIWords(input string) []string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})

	words := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		words = append(words, part)
	}
	return words
}

func mergeNarrative(existing, addition string) string {
	existing = strings.TrimSpace(existing)
	addition = strings.TrimSpace(addition)

	switch {
	case addition == "":
		return existing
	case existing == "":
		return addition
	case strings.Contains(existing, addition):
		return existing
	default:
		return existing + "\n" + addition
	}
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, addition := range additions {
		addition = strings.TrimSpace(addition)
		if addition == "" {
			continue
		}
		if _, ok := seen[addition]; ok {
			continue
		}
		seen[addition] = struct{}{}
		values = append(values, addition)
	}
	return values
}

func containsAny(input string, keywords ...string) bool {
	tokens := splitWordTokens(strings.ToLower(input))
	if len(tokens) == 0 {
		return false
	}

	for _, keyword := range keywords {
		if containsTokenSequence(tokens, splitWordTokens(strings.ToLower(keyword))) {
			return true
		}
	}
	return false
}

func normalizeReadLineError(line string, err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF {
		if strings.TrimSpace(line) != "" {
			return fmt.Errorf("interactive input ended unexpectedly before a newline: %w", err)
		}
		return fmt.Errorf("interactive input ended unexpectedly: %w", err)
	}
	return err
}

func splitWordTokens(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func containsTokenSequence(tokens, keywordTokens []string) bool {
	if len(keywordTokens) == 0 || len(keywordTokens) > len(tokens) {
		return false
	}

	for start := 0; start <= len(tokens)-len(keywordTokens); start++ {
		matched := true
		for offset, keyword := range keywordTokens {
			if tokens[start+offset] != keyword {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}

	return false
}

func tradeOffPreference(bias types.QualitySpeedBias) float64 {
	switch bias {
	case types.QualitySpeedBiasQuality:
		return -0.75
	case types.QualitySpeedBiasSpeed:
		return 0.75
	default:
		return 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
