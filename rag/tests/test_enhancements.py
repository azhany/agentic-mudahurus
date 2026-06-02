from mudahurus_rag.enhancements.orchestration import (
    FulfillmentAgent, Router, SellerCopilotAgent,
)


def test_router_routing():
    r = Router([SellerCopilotAgent(), FulfillmentAgent()])
    assert r.route("public", "storefront", "do you sell teh?") == "storefront_assistant"
    assert r.route("seller", "admin", "track my poslaju") == "fulfillment"
    assert r.route("seller", "admin", "update price of SKU1") == "seller_copilot"


def test_copilot_proposes_not_executes():
    resp = SellerCopilotAgent().handle("t1", "change price to 10")
    assert resp.actions  # proposes an action
    assert resp.agent == "seller_copilot"
