// Profile: ai-safe
// Edit only AUTHOR ZONE sections. Keep Handshake aligned with plugin.json.

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"bindunet/sdk"
)

const defaultGreeting = "Hello World"

type Plugin struct {
	mu          sync.RWMutex
	greeting    string
	configReady bool
}

func NewPlugin() (*Plugin, error) {
	return &Plugin{
		greeting: defaultGreeting,
	}, nil
}

func (p *Plugin) Handshake() sdk.HandshakeInfo {
	return sdk.HandshakeInfo{
		Inputs:          []string{"HelloWorldConfig"},
		Outputs:         []string{"HelloWorld", "HelloWorldStats"},
		Capabilities:    []string{"timer:cron"},
		Language:        "go",
		SDKVersion:      "",
		ContractVersion: "bpr/v1",
	}
}

func (p *Plugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
	switch strings.TrimSpace(trigger.ID) {
	case "cron":
		if err := p.ensureGreetingConfig(ctx); err != nil {
			ctx.Logger().Printf("config bootstrap failed; using default greeting: %v", err)
		}
		return []sdk.Message{
			{
				Type: "HelloWorld",
				Payload: map[string]any{
					"greeting": p.currentGreeting(),
				},
			},
		}, nil
	case "cron.2":
		stats, err := buildHelloWorldStats(ctx)
		if err != nil {
			return nil, err
		}
		return []sdk.Message{stats}, nil
	default:
		ctx.Logger().Printf("unknown trigger %q ignored", trigger.ID)
		return nil, nil
	}
}

func (p *Plugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	if strings.TrimSpace(event.MessageType) != "HelloWorldConfig" {
		return nil, nil
	}
	if greeting, ok := extractGreetingText(event.Payload); ok {
		p.setGreeting(greeting)
		ctx.Logger().Printf("hello world config updated from event")
		return nil, nil
	}
	ctx.Logger().Printf("HelloWorldConfig received without greeting_text; keeping previous value")
	return nil, nil
}

func buildHelloWorldStats(ctx *sdk.Context) (sdk.Message, error) {
	until := ctx.Now().UTC().Truncate(time.Second)
	since := until.Add(-15 * time.Minute)

	// Use previous stats window end as continuity anchor if available.
	rows, err := ctx.QueryReadOnly(nil,
		`SELECT json_extract(fields, '$.until') AS until
FROM v_messages
WHERE "$type" = 'HelloWorldStats'
ORDER BY "$when" DESC
LIMIT 1`)
	if err != nil {
		ctx.Logger().Printf("stats anchor query failed; using rolling 15m window: %v", err)
	} else if len(rows) > 0 {
		if lastUntil, ok := rows[0]["until"].(string); ok {
			if parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(lastUntil)); parseErr == nil {
				since = parsed.UTC().Truncate(time.Second)
			}
		}
	}

	countRows, err := ctx.QueryReadOnly(nil,
		`SELECT COUNT(*) AS count
FROM v_messages
WHERE "$type" = ?
  AND "$when" >= ?
  AND "$when" < ?`,
		"HelloWorld",
		since.Format(time.RFC3339),
		until.Format(time.RFC3339),
	)
	if err != nil {
		return sdk.Message{}, err
	}

	count := 0
	if len(countRows) > 0 {
		count = parseCount(countRows[0]["count"])
	}

	return sdk.Message{
		Type: "HelloWorldStats",
		Payload: map[string]any{
			"count": count,
			"since": since.Format(time.RFC3339),
			"until": until.Format(time.RFC3339),
		},
	}, nil
}

func (p *Plugin) ensureGreetingConfig(ctx *sdk.Context) error {
	p.mu.RLock()
	if p.configReady {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	rows, err := ctx.QueryReadOnly(nil,
		`SELECT
  COALESCE(
    json_extract(fields, '$.greeting_text'),
    json_extract(fields, '$.fields.greeting_text')
  ) AS greeting_text
FROM v_messages
WHERE "$type" = 'HelloWorldConfig'
ORDER BY "$when" DESC
LIMIT 1`)
	if err != nil {
		p.mu.Lock()
		p.configReady = true
		p.mu.Unlock()
		return err
	}

	if len(rows) > 0 {
		if greeting, ok := extractGreetingText(rows[0]); ok {
			p.setGreeting(greeting)
		}
	}

	p.mu.Lock()
	p.configReady = true
	p.mu.Unlock()
	return nil
}

func extractGreetingText(payload map[string]any) (string, bool) {
	if payload == nil {
		return "", false
	}
	if raw, ok := payload["greeting_text"]; ok {
		if txt := strings.TrimSpace(fmt.Sprintf("%v", raw)); txt != "" && txt != "<nil>" {
			return txt, true
		}
	}
	if fields, ok := payload["fields"].(map[string]any); ok {
		if raw, ok := fields["greeting_text"]; ok {
			if txt := strings.TrimSpace(fmt.Sprintf("%v", raw)); txt != "" && txt != "<nil>" {
				return txt, true
			}
		}
	}
	return "", false
}

func (p *Plugin) setGreeting(greeting string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.greeting = strings.TrimSpace(greeting)
	if p.greeting == "" {
		p.greeting = defaultGreeting
	}
	p.configReady = true
}

func (p *Plugin) currentGreeting() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.greeting) == "" {
		return defaultGreeting
	}
	return p.greeting
}

func parseCount(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}
	return 0
}

func main() {
	plugin, err := NewPlugin()
	if err != nil {
		log.Fatalf("plugin init failed: %v", err)
	}
	if err := sdk.Run(plugin); err != nil {
		log.Fatalf("plugin runtime failed: %v", err)
	}
	fmt.Printf("plugin %s %s ready\n", "hello-world-plugin", "1.0.0")
}
