package events

import (
	"strings"
	"testing"
	"time"
)

func TestValidateEnvelopeAcceptsRealEventShapes(t *testing.T) {
	for _, event := range []map[string]any{validCatalogEvent(), validTrainingCenterEvent()} {
		if err := ValidateEnvelope(event); err != nil {
			t.Fatalf("expected valid event, got %v", err)
		}
	}
}

func TestValidateEnvelopeRejectsContractViolations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing required payload field", func(event map[string]any) { delete(event["payload"].(map[string]any), "catalog_code") }},
		{"invalid event id", func(event map[string]any) { event["event_id"] = "not-a-uuid" }},
		{"invalid timestamp", func(event map[string]any) { event["timestamp"] = "not-a-timestamp" }},
		{"wrong event type", func(event map[string]any) { event["event_type"] = "reference.catalog.deleted" }},
		{"wrong source service", func(event map[string]any) { event["source_service"] = "other-service" }},
		{"unsupported change type", func(event map[string]any) {
			event["payload"].(map[string]any)["changed_items"].([]any)[0].(map[string]any)["change_type"] = "DEACTIVATED"
		}},
		{"additional payload property", func(event map[string]any) { event["payload"].(map[string]any)["unexpected"] = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := validCatalogEvent()
			test.mutate(event)
			if err := ValidateEnvelope(event); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateOutboxEventRejectsInvalidPayload(t *testing.T) {
	err := ValidateOutboxEvent(CatalogUpdatedEvent, "1e68b101-a8f0-4b4e-87fe-a4db5d4d5045", map[string]any{"catalog_id": "catalog-1"}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "event contract validation") {
		t.Fatalf("expected contract validation error, got %v", err)
	}
}

func validCatalogEvent() map[string]any {
	return envelope(CatalogUpdatedEvent, map[string]any{
		"catalog_id": "catalog-1", "catalog_code": "MODALITY", "updated_by": "admin-1", "updated_at": "2026-09-13T12:00:00Z",
		"changed_items": []any{map[string]any{"item_id": "detail-1", "code": "IN_PERSON", "label": "Presencial", "change_type": "UPDATED"}},
	})
}

func validTrainingCenterEvent() map[string]any {
	return envelope(TrainingCenterCreatedEvent, map[string]any{
		"training_center_id": "center-1", "center_code": "C-001", "name": "Centro", "municipality_id": "municipality-1", "created_by": "admin-1", "created_at": "2026-09-13T12:00:00Z",
	})
}

func envelope(eventType string, payload map[string]any) map[string]any {
	return map[string]any{
		"envelope_version": "1.0", "version": "1.0", "event_id": "1e68b101-a8f0-4b4e-87fe-a4db5d4d5045",
		"event_type": eventType, "source_service": "reference-data-service", "source_port": 8002,
		"timestamp": "2026-09-13T12:00:00Z", "occurred_at": "2026-09-13T12:00:00Z", "payload": payload,
	}
}
