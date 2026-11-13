package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

// ComputeStateHash calculates a deterministic SHA-256 fingerprint of the agent state.
func ComputeStateHash(trajID string, stepIndex int, prompt string, toolCalls []models.ToolCall, mem *models.MemorySnapshot) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%d:%s|", trajID, stepIndex, prompt)))

	// Canonicalize tool calls
	for _, tc := range toolCalls {
		h.Write([]byte(fmt.Sprintf("%s:%s:%t|", tc.ToolName, string(tc.Arguments), tc.IsError)))
	}

	// Canonicalize memory snapshot variables
	if mem != nil && len(mem.Variables) > 0 {
		var rawMap map[string]any
		if err := json.Unmarshal(mem.Variables, &rawMap); err == nil {
			keys := make([]string, 0, len(rawMap))
			for k := range rawMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				valBytes, _ := json.Marshal(rawMap[k])
				h.Write([]byte(fmt.Sprintf("%s=%s;", k, string(valBytes))))
			}
		}
		if mem.Scratchpad != "" {
			h.Write([]byte("|pad=" + mem.Scratchpad))
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}
