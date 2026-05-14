package sipgateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emiago/diago"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"

	"reporter/internal/domain"
)

var ErrDisabled = errors.New("sip gateway disabled")

type DialResult struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	DialogID string `json:"dialogId,omitempty"`
}

type Gateway interface {
	Dial(ctx context.Context, endpoint domain.SipEndpoint, call domain.CallSession) (DialResult, error)
	Hangup(ctx context.Context, callID string) error
}

type DiagoGateway struct {
	mu      sync.Mutex
	dialogs map[string]*diago.DialogClientSession
	logger  *slog.Logger
}

func NewDiagoGateway(logger *slog.Logger) *DiagoGateway {
	if logger == nil {
		logger = slog.Default()
	}
	return &DiagoGateway{dialogs: map[string]*diago.DialogClientSession{}, logger: logger}
}

func (g *DiagoGateway) Dial(ctx context.Context, endpoint domain.SipEndpoint, call domain.CallSession) (DialResult, error) {
	if !configBool(endpoint.Config, "enabled") {
		return DialResult{Provider: "diago", Status: "disabled"}, ErrDisabled
	}

	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(configString(endpoint.Config, "userAgent", "reporter-sip-gateway")),
		sipgo.WithUserAgentHostname(firstNonEmpty(endpoint.Domain, "localhost")),
	)
	if err != nil {
		return DialResult{Provider: "diago", Status: "failed"}, err
	}

	dg := diago.NewDiago(ua, diago.WithTransport(diago.Transport{
		Transport: configString(endpoint.Config, "transport", "udp"),
		BindHost:  configString(endpoint.Config, "bindHost", "0.0.0.0"),
		BindPort:  configInt(endpoint.Config, "bindPort", 0),
	}))

	recipient, err := recipientURI(endpoint, call.PhoneNumber)
	if err != nil {
		ua.Close()
		return DialResult{Provider: "diago", Status: "failed"}, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(configInt(endpoint.Config, "dialTimeoutSeconds", 15))*time.Second)
	defer cancel()
	dialog, err := dg.Invite(dialCtx, recipient, diago.InviteOptions{
		Transport: configString(endpoint.Config, "transport", "udp"),
		Username:  configString(endpoint.Config, "username", ""),
		Password:  configString(endpoint.Config, "password", ""),
	})
	if err != nil {
		ua.Close()
		return DialResult{Provider: "diago", Status: "failed"}, err
	}

	g.mu.Lock()
	g.dialogs[call.ID] = dialog
	g.mu.Unlock()
	g.logger.Info("diago outbound call established", "callId", call.ID, "dialogId", dialog.ID)

	return DialResult{Provider: "diago", Status: "connected", DialogID: dialog.ID}, nil
}

func (g *DiagoGateway) Hangup(ctx context.Context, callID string) error {
	g.mu.Lock()
	dialog, ok := g.dialogs[callID]
	if ok {
		delete(g.dialogs, callID)
	}
	g.mu.Unlock()
	if !ok {
		return nil
	}
	defer dialog.Close()
	return dialog.Hangup(ctx)
}

func recipientURI(endpoint domain.SipEndpoint, phone string) (sip.Uri, error) {
	raw := configString(endpoint.Config, "trunkUri", "")
	if raw == "" {
		domainPart := firstNonEmpty(configString(endpoint.Config, "trunkDomain", ""), endpoint.Domain)
		if domainPart == "" {
			return sip.Uri{}, fmt.Errorf("sip trunk domain is required")
		}
		raw = "sip:" + sanitizeDialString(phone) + "@" + domainPart
	}
	raw = strings.ReplaceAll(raw, "{phone}", sanitizeDialString(phone))
	uri := sip.Uri{}
	if err := sip.ParseUri(raw, &uri); err != nil {
		return sip.Uri{}, err
	}
	return uri, nil
}

func sanitizeDialString(value string) string {
	value = strings.TrimSpace(value)
	return strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(value)
}

func configString(config map[string]interface{}, key, fallback string) string {
	if config == nil {
		return fallback
	}
	value, ok := config[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func configBool(config map[string]interface{}, key string) bool {
	if config == nil {
		return false
	}
	value, ok := config[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1" || typed == "yes"
	default:
		return false
	}
}

func configInt(config map[string]interface{}, key string, fallback int) int {
	if config == nil {
		return fallback
	}
	value, ok := config[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
