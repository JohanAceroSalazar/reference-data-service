// Package events validates Reference Data events against their embedded JSON Schemas.
package events

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	CatalogUpdatedEvent        = "reference.catalog.updated"
	TrainingCenterCreatedEvent = "reference.training_center.created"
	EnvelopeVersion            = "1.0"
	SourceService              = "reference-data-service"
	SourcePort                 = 8002
)

//go:embed schemas/*.json
var schemasFS embed.FS

var schemaNames = map[string]string{
	CatalogUpdatedEvent:        "schemas/reference.catalog.updated.schema.json",
	TrainingCenterCreatedEvent: "schemas/reference.training_center.created.schema.json",
}

var (
	compiledSchemas map[string]*jsonschema.Schema
	compileOnce     sync.Once
	compileErr      error
)

// ValidateEnvelope validates a complete wire event, including its envelope.
func ValidateEnvelope(envelope any) error {
	if err := compile(); err != nil {
		return fmt.Errorf("event schema initialization: %w", err)
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode event for validation: %w", err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		return fmt.Errorf("decode event for validation: %w", err)
	}
	eventType, ok := document["event_type"].(string)
	if !ok || eventType == "" {
		return fmt.Errorf("event contract validation: event_type is required")
	}
	schema, ok := compiledSchemas[eventType]
	if !ok {
		return fmt.Errorf("event contract validation: unsupported event_type %q", eventType)
	}
	if err := schema.Validate(document); err != nil {
		return fmt.Errorf("event contract validation for %s: %w", eventType, err)
	}
	return nil
}

// ValidateOutboxEvent builds the deterministic Reference Data envelope that will
// be published for an outbox row and validates it before the row is persisted.
func ValidateOutboxEvent(eventType, eventID string, payload any, occurredAt time.Time) error {
	timestamp := occurredAt.UTC().Format(time.RFC3339)
	return ValidateEnvelope(map[string]any{
		"envelope_version": EnvelopeVersion,
		"version":          EnvelopeVersion,
		"event_id":         eventID,
		"event_type":       eventType,
		"source_service":   SourceService,
		"source_port":      SourcePort,
		"timestamp":        timestamp,
		"occurred_at":      timestamp,
		"payload":          payload,
	})
}

func compile() error {
	compileOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiledSchemas = make(map[string]*jsonschema.Schema, len(schemaNames))
		for eventType, name := range schemaNames {
			contents, err := schemasFS.ReadFile(name)
			if err != nil {
				compileErr = err
				return
			}
			var document any
			if err := json.Unmarshal(contents, &document); err != nil {
				compileErr = err
				return
			}
			url := "https://contracts.reference-data-service.local/events/" + strings.TrimPrefix(name, "schemas/")
			if err := compiler.AddResource(url, document); err != nil {
				compileErr = err
				return
			}
			schema, err := compiler.Compile(url)
			if err != nil {
				compileErr = err
				return
			}
			compiledSchemas[eventType] = schema
		}
	})
	return compileErr
}
