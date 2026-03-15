package spec

import (
	"fmt"
	"strings"
)

// BuildDynamicGatherPrompt は動的質問生成用のプロンプトを組み立てます。
func BuildDynamicGatherPrompt(state *ConversationState) string {
	if state == nil {
		return ""
	}

	var transcript strings.Builder
	for idx, turn := range state.Turns {
		fmt.Fprintf(&transcript, "Turn %d Question: %s\n", idx+1, turn.Question)
		if len(turn.Options) > 0 {
			fmt.Fprintf(&transcript, "Turn %d Options: %s\n", idx+1, strings.Join(turn.Options, " | "))
		}
		fmt.Fprintf(&transcript, "Turn %d Answer: %s\n", idx+1, turn.Answer)
	}
	if transcript.Len() == 0 {
		transcript.WriteString("No follow-up turns yet.\n")
	}

	return fmt.Sprintf(`You are refining an agent specification through a live dialogue.
Analyze the user's request and conversation history, then return only JSON matching this schema:
{
  "current_understanding": {
    "core_intent": "string",
    "primary_goal": "string",
    "success_criteria": ["string"]
  },
  "suggestions": [
    {
      "category": "string",
      "title": "string",
      "description": "string"
    }
  ],
  "next_question": {
    "prompt": "string",
    "options": ["string"]
  }
}

Rules:
- Keep the response grounded in the user's actual statements.
- "current_understanding" must summarize the current best interpretation.
- Provide 1 to 3 actionable suggestions.
- Ask exactly one next question unless the request is already sufficiently specified.
- "next_question.options" should contain 2 to 4 concise options when useful.
- Do not include markdown or explanatory text outside JSON.

Initial request:
%s

Current confidence: %.2f

Conversation history:
%s`, state.InitialInput, state.Confidence, transcript.String())
}
