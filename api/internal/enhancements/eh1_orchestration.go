package enhancements

import (
	"context"
	"strings"
)

// EH-1 — Multi-agent orchestration.
// Builds on the v1 RAG plane + retrieval API. A router classifies an incoming
// request and dispatches to a specialist agent:
//   - SellerCopilot     (WRITE-capable: drafts catalog edits, order actions)
//   - StorefrontAssistant (read-only product Q&A — the v1 assistant)
//   - FulfillmentAgent  (status/tracking, ties into EH-2)
//
// v1 ships only the read-only StorefrontAssistant. The write-capable agents are
// scaffolded here as interfaces with guarded stubs so a future PRD slice can
// implement tools without re-architecting the core (ARCHITECTURE §12).

type AgentKind string

const (
	AgentSellerCopilot       AgentKind = "seller_copilot"
	AgentStorefrontAssistant AgentKind = "storefront_assistant"
	AgentFulfillment         AgentKind = "fulfillment"
)

type AgentRequest struct {
	TenantID string
	Role     string // seller | operator | public
	Scope    string // admin | storefront
	Message  string
}

type AgentResponse struct {
	Agent     AgentKind
	Answer    string
	Actions   []ProposedAction // write-capable agents propose; human approves
	Grounded  bool
}

// ProposedAction is a write the Copilot proposes but never auto-executes in the
// scaffold — human-in-the-loop is mandatory until EH-1 is approved.
type ProposedAction struct {
	Kind   string            // e.g. "update_product", "advance_order_status"
	Target string            // resource id
	Params map[string]string
}

// Agent is the specialist contract.
type Agent interface {
	Kind() AgentKind
	Handle(ctx context.Context, req AgentRequest) (AgentResponse, error)
}

// Router selects an agent for a request (keyword heuristic placeholder; a real
// implementation would use an LLM classifier or embeddings).
type Router struct {
	agents map[AgentKind]Agent
}

func NewRouter(agents ...Agent) *Router {
	m := map[AgentKind]Agent{}
	for _, a := range agents {
		m[a.Kind()] = a
	}
	return &Router{agents: m}
}

func (r *Router) Route(req AgentRequest) AgentKind {
	msg := strings.ToLower(req.Message)
	switch {
	case req.Scope == "storefront":
		return AgentStorefrontAssistant
	case containsAny(msg, "track", "shipping", "deliver", "poslaju", "status"):
		return AgentFulfillment
	case req.Role == "seller" && containsAny(msg, "update", "create", "change", "set price", "edit"):
		return AgentSellerCopilot
	default:
		return AgentStorefrontAssistant
	}
}

func (r *Router) Dispatch(ctx context.Context, req AgentRequest) (AgentResponse, error) {
	kind := r.Route(req)
	if a, ok := r.agents[kind]; ok {
		return a.Handle(ctx, req)
	}
	return AgentResponse{Agent: kind, Answer: "agent not enabled", Grounded: false}, nil
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
