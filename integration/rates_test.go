//go:build integration

// Package integration_test contains opt-in live API checks.
package integration_test

import (
	"context"
	"testing"
	"time"

	pirateship "github.com/christianobora/pirateship-api"
)

func TestPublicRates(t *testing.T) {
	client, err := pirateship.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	weight := 16.0
	length, width, height := 6.0, 4.0, 2.0
	destinationZIP := "10001"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rates, _, err := client.Rates(ctx, pirateship.RatesRequest{
		OriginZIP:        "94105",
		DestinationZIP:   &destinationZIP,
		WeightOunces:     &weight,
		DimensionXInches: &length,
		DimensionYInches: &width,
		DimensionZInches: &height,
		MailClassKeys:    []pirateship.MailClassKey{pirateship.MailClassGroundAdvantage},
		PackageTypeKeys:  []pirateship.PackageTypeKey{pirateship.PackageTypeParcel},
	})
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}
	if len(rates) == 0 {
		t.Fatal("Rates() returned no services")
	}
}
