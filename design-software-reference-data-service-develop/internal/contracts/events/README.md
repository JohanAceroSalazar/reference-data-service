# Reference Data event contracts

This package embeds the JSON Schema Draft 2020-12 contracts used by the
Reference Data publisher. The service validates the complete event envelope
before direct publication and before an outbox row is persisted.

Validated events:

- `reference.catalog.updated`
- `reference.training_center.created`

The schemas reject unknown envelope and payload properties. `correlation_id`
is optional because the HTTP request identifier is not guaranteed to be a UUID.
`event_id` is a UUID and timestamps use RFC3339 date-time format.

## Change type compatibility

The running publisher emits `DEACTIVADO`; therefore this is the only accepted
deactivation value in the `reference.catalog.updated` schema. An older
documentation entry uses `DEACTIVATED`. The publisher was not changed because
doing so could break deployed consumers. Consumers and documentation owners
must agree on a migration before the emitted value is changed.

## Running contract checks

```powershell
cd design-software-reference-data-service-develop
go test ./...
```

Tests in this package validate positive and negative samples. Database and AMQP
integration tests remain opt-in and are skipped when their environment
variables are not configured.
