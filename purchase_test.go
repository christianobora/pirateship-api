package pirateship

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRateSelection(t *testing.T) {
	t.Parallel()
	group := RateGroup{GroupKey: RateGroupKey{
		String: "domestic",
		Traits: []RateGroupTrait{{Layer: "Destination", Value: "DOMESTIC"}},
	}}
	summary := RateSummary{
		UniqueID: "30-10-1",
	}
	selection, err := NewRateSelection(group, summary)
	if err != nil {
		t.Fatalf("NewRateSelection() error = %v", err)
	}
	if selection.MailClassID != "30" || selection.PackageTypeID != "10" || !selection.IsSaturdayDelivery {
		t.Errorf("selection = %#v", selection)
	}
	group.GroupKey.Traits[0].Value = "CHANGED"
	if selection.GroupKeyInput.Traits[0].Value != "DOMESTIC" {
		t.Error("selection traits alias input traits")
	}
}

func TestNewRateSelectionValidation(t *testing.T) {
	t.Parallel()
	tests := []RateSummary{
		{UniqueID: "invalid"},
		{UniqueID: "30-10-2"},
	}
	for _, summary := range tests {
		_, err := NewRateSelection(RateGroup{}, summary)
		var validationError *ValidationError
		if !errors.As(err, &validationError) {
			t.Errorf("NewRateSelection(%q) error = %v", summary.UniqueID, err)
		}
	}
}

func TestMinimumCharge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                              string
		labelTotal, balance, defaultTopUp float64
		want                              float64
	}{
		{name: "balance covers price", labelTotal: 5, balance: 10, defaultTopUp: 20, want: 0},
		{name: "minimum dollar", labelTotal: 5.25, balance: 5, defaultTopUp: 0, want: 1},
		{name: "default top up", labelTotal: 5, balance: 0, defaultTopUp: 20, want: 20},
		{name: "shortfall exceeds default", labelTotal: 25.99, balance: 0, defaultTopUp: 20, want: 25.99},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := MinimumCharge(test.labelTotal, test.balance, test.defaultTopUp)
			if err != nil || got != test.want {
				t.Errorf("MinimumCharge() = %v, %v; want %v", got, err, test.want)
			}
		})
	}
	if _, err := MinimumCharge(-1, 0, 0); err == nil {
		t.Error("MinimumCharge(-1) error = nil")
	}
}

func TestBillableLabelTotal(t *testing.T) {
	t.Parallel()
	groups := []RateGroup{
		{
			GroupKey:      RateGroupKey{String: "outbound"},
			RateSummaries: []RateSummary{{UniqueID: "1-1-0", TotalPrice: 5.255}},
		},
		{
			GroupKey: RateGroupKey{
				String: "return",
				Traits: []RateGroupTrait{{Layer: "Direction", Value: "RETURN"}},
			},
			RateSummaries: []RateSummary{{UniqueID: "2-1-0", TotalPrice: 7}},
		},
	}
	total, err := BillableLabelTotal(groups, map[string]string{
		"outbound": "1-1-0",
		"return":   "2-1-0",
	})
	if err != nil || total != 5.26 {
		t.Fatalf("BillableLabelTotal() = %v, %v; want 5.26", total, err)
	}
	if _, err := BillableLabelTotal(groups, map[string]string{"outbound": "missing"}); err == nil {
		t.Fatal("BillableLabelTotal() error = nil for incomplete selection")
	}
}

func TestPurchaseRequiresExplicitConfirmation(t *testing.T) {
	t.Parallel()
	var called bool
	client, err := NewClient(WithHTTPClient(httpClientFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("unexpected request")
	})))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.PurchaseBatch(context.Background(), PurchaseBatchRequest{})
	if err == nil {
		t.Fatal("PurchaseBatch() error = nil")
	}
	if called {
		t.Fatal("PurchaseBatch() made an HTTP request without confirmation")
	}
}

func TestPurchaseSendsOneMutation(t *testing.T) {
	t.Parallel()
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		var payload struct {
			Variables map[string]json.RawMessage `json:"variables"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if _, exists := payload.Variables["ConfirmCharge"]; exists {
			t.Error("ConfirmCharge was sent to GraphQL")
		}
		if _, exists := payload.Variables["confirmCharge"]; exists {
			t.Error("confirmCharge was sent to GraphQL")
		}
		_, _ = io.WriteString(writer, `{"data":{"buyBatch":{"id":"batch","step":"BILLED","numShipments":1,"shipments":[{"id":"shipment"}]}}}`)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	result, _, err := client.PurchaseBatch(context.Background(), PurchaseBatchRequest{
		BatchID: "batch", RateSelection: []RateSelection{{
			GroupKeyInput: RateGroupKeyInput{Traits: []RateGroupTrait{{Layer: "A", Value: "B"}}},
			MailClassID:   "30", PackageTypeID: "10",
		}},
		PaymentSourceID: "payment", ShipDate: "2026-09-15", TotalCharge: 1, ConfirmCharge: true,
	})
	if err != nil {
		t.Fatalf("PurchaseBatch() error = %v", err)
	}
	if calls != 1 || result.ID != "batch" || len(result.Shipments) != 1 {
		t.Errorf("calls=%d result=%#v", calls, result)
	}
}
