//go:build pact

package hierarchy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/pact-foundation/pact-go/v2/provider"

	"github.com/code-sena/design-software-reference-data-service/internal/adapter/in/http/security"
	application "github.com/code-sena/design-software-reference-data-service/internal/application/hierarchy"
)

const pactCatalogState = "an active Reference Data catalog exists"

// TestPactProvider verifies the consumer-generated contract through the
// provider's real routes, RBAC middleware and HTTP serialization. The contract
// path is supplied by CI or the caller so this independent repository does not
// assume a local consumer checkout.
func TestPactProvider(t *testing.T) {
	pactFile := os.Getenv("PACT_CONTRACT_PATH")
	if pactFile == "" {
		t.Fatal("PACT_CONTRACT_PATH is required and must point to the consumer-generated Pact JSON")
	}
	if _, err := os.Stat(pactFile); err != nil {
		t.Fatalf("read Pact contract %q: %v", pactFile, err)
	}

	reader := &pactCatalogReader{}
	queries := application.NewQueryService(reader)
	providerMux := http.NewServeMux()
	security.RegisterProtectedRoutes(providerMux, pactTokenValidator{}, NewHandler(queries).Routes())

	rootMux := http.NewServeMux()
	rootMux.Handle("/api/v1/", http.StripPrefix("/api/v1", providerMux))
	server := httptest.NewServer(rootMux)
	t.Cleanup(server.Close)

	requestFilter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Pact requests deliberately omit credentials because the MFE does not
			// own IAM tokens. This test-only token keeps the production RBAC
			// middleware enabled without weakening runtime JWT/JWKS validation.
			r.Header.Set("Authorization", "Bearer pact-provider-token")
			next.ServeHTTP(w, r)
		})
	}

	err := provider.NewVerifier().VerifyProvider(t, provider.VerifyRequest{
		Provider:        "ReferenceDataService",
		ProviderBaseURL: server.URL,
		PactFiles:       []string{pactFile},
		RequestFilter:   requestFilter,
		StateHandlers: models.StateHandlers{
			pactCatalogState: func(setup bool, _ models.ProviderState) (models.ProviderStateResponse, error) {
				reader.SetActive(setup)
				return models.ProviderStateResponse{}, nil
			},
		},
		DisableColoredOutput: true,
	})
	if err != nil {
		t.Fatalf("verify Reference Data provider Pact: %v", err)
	}
}

type pactCatalogReader struct {
	mu     sync.RWMutex
	active bool
}

func (r *pactCatalogReader) SetActive(active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = active
}

func (r *pactCatalogReader) List(_ context.Context, query application.ListQuery) (application.Page, error) {
	r.mu.RLock()
	active := r.active
	r.mu.RUnlock()
	if query.Resource != "catalogs" || !active {
		return application.Page{Data: []json.RawMessage{}, TotalItems: 0}, nil
	}

	return application.Page{
		Data:       []json.RawMessage{json.RawMessage(`{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","code":"DOCUMENT_TYPE","name":"Document type","description":"Reference catalog for document types","is_active":true,"created_at":"2026-01-01T00:00:00Z"}`)},
		TotalItems: 1,
	}, nil
}

func (*pactCatalogReader) Get(context.Context, string, string) (json.RawMessage, error) {
	return nil, application.ErrNotFound
}

func (*pactCatalogReader) ListCatalogDetails(context.Context, string, application.ListQuery) (application.Page, error) {
	return application.Page{Data: []json.RawMessage{}}, nil
}

func (*pactCatalogReader) ReportActiveCatalog(context.Context, string, int, int) ([]application.ActiveCatalogReportEntry, int, error) {
	return []application.ActiveCatalogReportEntry{}, 0, nil
}

type pactTokenValidator struct{}

func (pactTokenValidator) Validate(_ context.Context, token string) (security.Principal, error) {
	if token != "pact-provider-token" {
		return security.Principal{}, security.ErrInvalid
	}
	return security.Principal{Features: []string{"REF_CATALOG_VIEW:GLOBAL"}}, nil
}
