package persistence

import "testing"

func TestNewOutboxEventRejectsInvalidContractPayload(t *testing.T) {
	_, err := NewOutboxEvent("catalog", "catalog-1", "reference.catalog.updated", map[string]any{"catalog_id": "catalog-1"})
	if err == nil {
		t.Fatal("expected invalid payload to be rejected before outbox persistence")
	}
}
