// Profile: ai-safe
// Edit only AUTHOR ZONE sections. Keep Handshake aligned with plugin.json.

package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"bindunet/sdk"
)

type Plugin struct {
	mu sync.RWMutex
}

func NewPlugin() (*Plugin, error) {
	return &Plugin{}, nil
}

func (p *Plugin) Handshake() sdk.HandshakeInfo {
	return sdk.HandshakeInfo{
		Inputs:          []string{"MASclTrigger"},
		Outputs:         []string{"MASclOcurrence"},
		Capabilities:    []string{"timer:cron"},
		Language:        "go",
		SDKVersion:      "",
		ContractVersion: "bpr/v1",
	}
}

func (p *Plugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
	nodeID := ctx.Info().NodeID
	if nodeID == "" {
		nodeID = "root"
	}
	now := ctx.Now().UTC()
	occ := sdk.Message{
		Type: "MASclOcurrence",
		Payload: map[string]any{
			"occurrence_id":  fmt.Sprintf("occ-bpr-cron-%d", now.Unix()),
			"agent_type":     "BPR",
			"agent_name":     "bindu-mascl-plugin",
			"execution_mode": "CRON",
			"node_id":        nodeID,
			"timestamp":      now.Format(time.RFC3339),
			"status":         "OK",
			"latency_ms":     0.85,
			"details":        "BPR Native Go plugin executed periodic 15-minute cron trigger",
		},
	}
	ctx.Logger().Printf("BPR Plugin emitted cron occurrence on node %s", nodeID)
	return []sdk.Message{occ}, nil
}

func (p *Plugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	if strings.TrimSpace(event.MessageType) != "MASclTrigger" {
		return nil, nil
	}
	nodeID := ctx.Info().NodeID
	if nodeID == "" {
		nodeID = "root"
	}
	now := ctx.Now().UTC()

	source := "external"
	if s, ok := event.Payload["source"].(string); ok && s != "" {
		source = s
	} else if fields, ok := event.Payload["fields"].(map[string]any); ok {
		if s, ok := fields["source"].(string); ok && s != "" {
			source = s
		}
	}

	occ := sdk.Message{
		Type: "MASclOcurrence",
		Payload: map[string]any{
			"occurrence_id":  fmt.Sprintf("occ-bpr-trig-%d", now.Unix()),
			"agent_type":     "BPR",
			"agent_name":     "bindu-mascl-plugin",
			"execution_mode": "TRIGGER",
			"node_id":        nodeID,
			"timestamp":      now.Format(time.RFC3339),
			"status":         "OK",
			"latency_ms":     1.15,
			"details":        fmt.Sprintf("BPR Native Go plugin responded to MASclTrigger event from source '%s'", source),
		},
	}
	ctx.Logger().Printf("BPR Plugin emitted reactive occurrence on node %s for source %s", nodeID, source)
	return []sdk.Message{occ}, nil
}

func main() {
	plugin, err := NewPlugin()
	if err != nil {
		log.Fatalf("plugin init failed: %v", err)
	}
	if err := sdk.Run(plugin); err != nil {
		log.Fatalf("plugin runtime failed: %v", err)
	}
	fmt.Printf("plugin %s %s ready\n", "bindu-mascl-plugin", "1.0.0")
}
