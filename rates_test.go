package pirateship

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, `{"data":{"rates":[{
          "title":"Ground Advantage","mailClassKey":"GroundAdvantage",
          "mailClass":{"accuracy":null,"international":false},
          "packageTypeKey":"Parcel","carrier":{"carrierKey":"usps","title":"USPS"},
          "totalPrice":5.25,"best":true,"surcharges":[]
        }]}}`)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	rates, response, err := client.Rates(context.Background(), validRatesRequest())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}
	if response.StatusCode != http.StatusOK || len(rates) != 1 {
		t.Fatalf("Rates() response=%#v rates=%#v", response, rates)
	}
	if rates[0].Carrier.Key != CarrierUSPS || !rates[0].Best || rates[0].TotalPrice != 5.25 {
		t.Errorf("rate = %#v", rates[0])
	}
}

func TestRatesValidation(t *testing.T) {
	t.Parallel()

	float := func(value float64) *float64 { return &value }
	tests := []struct {
		name   string
		modify func(*RatesRequest)
	}{
		{name: "origin ZIP", modify: func(r *RatesRequest) { r.OriginZIP = "" }},
		{name: "mail classes", modify: func(r *RatesRequest) { r.MailClassKeys = nil }},
		{name: "package types", modify: func(r *RatesRequest) { r.PackageTypeKeys = nil }},
		{name: "missing weight", modify: func(r *RatesRequest) { r.WeightOunces = nil }},
		{name: "weight", modify: func(r *RatesRequest) { r.WeightOunces = float(0) }},
		{name: "missing length", modify: func(r *RatesRequest) { r.DimensionXInches = nil }},
		{name: "missing width", modify: func(r *RatesRequest) { r.DimensionYInches = nil }},
		{name: "missing height", modify: func(r *RatesRequest) { r.DimensionZInches = nil }},
		{name: "length", modify: func(r *RatesRequest) { r.DimensionXInches = float(5.99) }},
		{name: "width", modify: func(r *RatesRequest) { r.DimensionYInches = float(2.99) }},
		{name: "height", modify: func(r *RatesRequest) { r.DimensionZInches = float(0.24) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := validRatesRequest()
			test.modify(&request)
			client, err := NewClient(WithHTTPClient(httpClientFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("HTTP request made for invalid input")
				return nil, nil
			})))
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = client.Rates(context.Background(), request)
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("error = %v, want ValidationError", err)
			}
		})
	}
}

func TestFlatRatePackageDoesNotRequireMinimumDimensions(t *testing.T) {
	t.Parallel()
	value := 0.1
	request := validRatesRequest()
	request.PackageTypeKeys = []PackageTypeKey{PackageTypeFlatRateEnvelope}
	request.DimensionXInches = &value
	request.DimensionYInches = &value
	request.DimensionZInches = &value
	if err := request.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}
}

func validRatesRequest() RatesRequest {
	weight := 16.0
	length := 6.0
	width := 4.0
	height := 2.0
	return RatesRequest{
		OriginZIP:        "94105",
		WeightOunces:     &weight,
		DimensionXInches: &length,
		DimensionYInches: &width,
		DimensionZInches: &height,
		MailClassKeys:    []MailClassKey{MailClassGroundAdvantage},
		PackageTypeKeys:  []PackageTypeKey{PackageTypeParcel},
	}
}
