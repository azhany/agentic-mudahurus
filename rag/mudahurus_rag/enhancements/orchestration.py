"""EH-1 — Multi-agent orchestration (RAG-plane side).

A router classifies a message and dispatches to a specialist agent. v1 ships
only the read-only StorefrontAssistant (the grounded assistant). The
write-capable SellerCopilot and the FulfillmentAgent are scaffolded as classes
that propose actions for human approval — they never auto-execute here.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional


@dataclass
class ProposedAction:
    kind: str
    target: str
    params: dict = field(default_factory=dict)


@dataclass
class AgentResponse:
    agent: str
    answer: str
    grounded: bool = False
    actions: List[ProposedAction] = field(default_factory=list)


class Agent:
    kind = "agent"

    def handle(self, tenant_id: str, message: str) -> AgentResponse:  # pragma: no cover - interface
        raise NotImplementedError


class StorefrontAssistantAgent(Agent):
    """Wraps the v1 read-only assistant service."""
    kind = "storefront_assistant"

    def __init__(self, assistant_service) -> None:
        self._svc = assistant_service

    def handle(self, tenant_id: str, message: str) -> AgentResponse:
        ans = self._svc.ask(tenant_id, message, scope="storefront")
        return AgentResponse(agent=self.kind, answer=ans.answer, grounded=ans.grounded)


class SellerCopilotAgent(Agent):
    """WRITE-capable copilot (scaffold). Proposes actions; never executes."""
    kind = "seller_copilot"

    def handle(self, tenant_id: str, message: str) -> AgentResponse:
        return AgentResponse(
            agent=self.kind,
            answer="(EH-1 scaffold) I can draft this change for your approval.",
            actions=[ProposedAction(kind="draft", target="", params={"message": message})],
        )


class FulfillmentAgent(Agent):
    kind = "fulfillment"

    def handle(self, tenant_id: str, message: str) -> AgentResponse:
        return AgentResponse(agent=self.kind, answer="(EH-1 scaffold) tracking/fulfillment not enabled in v1.")


class Router:
    def __init__(self, agents: Optional[List[Agent]] = None) -> None:
        self._agents = {a.kind: a for a in (agents or [])}

    def route(self, role: str, scope: str, message: str) -> str:
        m = message.lower()
        if scope == "storefront":
            return "storefront_assistant"
        if any(k in m for k in ("track", "shipping", "deliver", "poslaju", "status")):
            return "fulfillment"
        if role == "seller" and any(k in m for k in ("update", "create", "change", "edit", "set price")):
            return "seller_copilot"
        return "storefront_assistant"

    def dispatch(self, tenant_id: str, role: str, scope: str, message: str) -> AgentResponse:
        kind = self.route(role, scope, message)
        agent = self._agents.get(kind)
        if agent is None:
            return AgentResponse(agent=kind, answer="agent not enabled")
        return agent.handle(tenant_id, message)
