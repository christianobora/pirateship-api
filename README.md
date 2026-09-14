# Pirate Ship API for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/christianobora/pirateship-api.svg)](https://pkg.go.dev/github.com/christianobora/pirateship-api)
[![CI](https://github.com/christianobora/pirateship-api/actions/workflows/ci.yml/badge.svg)](https://github.com/christianobora/pirateship-api/actions/workflows/ci.yml)
[![CodeQL](https://github.com/christianobora/pirateship-api/actions/workflows/codeql.yml/badge.svg)](https://github.com/christianobora/pirateship-api/actions/workflows/codeql.yml)

An idiomatic, dependency-free Go client for Pirate Ship's GraphQL API. It
supports public rate estimates and the authenticated shipment lifecycle:
warehouses, shipment presets, single-shipment batches, rating, insurance,
label purchase, asynchronous status, label files, mobile codes, and refunds.

> [!WARNING]
> Pirate Ship does not publish or support this GraphQL API. The schema can
> change without notice. Pin a module version, test your workflow, and treat
> authenticated operations as integration points that may need maintenance.

## Install

```sh
go get github.com/christianobora/pirateship-api
```

The module requires Go 1.24 or newer.

## Public rates

The public rates endpoint does not require a Pirate Ship session.

```go
package main

import (
	"context"
	"fmt"
	"log"

	pirateship "github.com/christianobora/pirateship-api"
)

func main() {
	client, err := pirateship.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	weight := 16.0 // ounces
	length, width, height := 6.0, 4.0, 2.0 // inches
	rates, _, err := client.Rates(context.Background(), pirateship.RatesRequest{
		OriginZIP:        "94105",
		DestinationZIP:   stringPointer("10001"),
		WeightOunces:     &weight,
		DimensionXInches: &length,
		DimensionYInches: &width,
		DimensionZInches: &height,
		MailClassKeys:    []pirateship.MailClassKey{pirateship.MailClassGroundAdvantage},
		PackageTypeKeys:  []pirateship.PackageTypeKey{pirateship.PackageTypeParcel},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, rate := range rates {
		fmt.Printf("%s: $%.2f\n", rate.Title, rate.TotalPrice)
	}
}

func stringPointer(value string) *string { return &value }
```

A command-line version is available in [`examples/rates`](examples/rates).

## Authentication

Authenticated calls use the normal Pirate Ship web session. Supply an
`*http.Client` whose cookie jar already contains a valid session:

```go
client, err := pirateship.NewClient(
	pirateship.WithHTTPClient(authenticatedHTTPClient),
)
```

The package does not read browser profiles, cookies, passwords, or local
storage. Session acquisition is deliberately left to the application so it can
use an appropriate interactive login and secret-storage policy. Never commit
session cookies or log GraphQL response bodies from authenticated requests.

## Label workflow

The common authenticated flow is:

1. Call `ShippingConfiguration` to obtain warehouse, preset, carrier, and
   package-type IDs.
2. Create or select a warehouse and shipment preset.
3. Call `CreateShipment`; this creates and rates a batch but does not buy a
   label.
4. Call `WaitForBatch` until the batch is `RATED`.
5. Call `PurchaseInformation`, select one `RateSummary` per `RateGroup`, and
   convert each with `NewRateSelection`.
6. Calculate the charged label total with `BillableLabelTotal`, then calculate
   the account top-up using `MinimumCharge`.
7. Call `PurchaseBatch` with `ConfirmCharge: true`.
8. Wait for `BILLED`, then call `TriggerLabels`, `WaitForLabels`, and
   `DownloadLabel`.

See [`examples/label-workflow`](examples/label-workflow) for a complete typed
workflow function.

### Purchase safety

`PurchaseBatch` can charge the selected payment source. It refuses to make an
HTTP request unless `ConfirmCharge` is explicitly `true`. Mutations are never
retried automatically, which prevents an ambiguous network failure from
silently repeating a purchase.

`TotalCharge` is the amount added to the Pirate Ship account balance, not the
sum of label prices. Use `MinimumCharge` with the values returned by
`PurchaseInformation`.

## Custom operations

The typed API covers the shipment and label lifecycle. `Client.Do` is public as
a forward-compatible escape hatch for newer GraphQL operations:

```go
var result struct {
	Company struct {
		ID pirateship.ID `json:"id"`
	} `json:"company"`
}

_, err := client.Do(ctx, pirateship.Operation{
	Name:  "CompanyIDQuery",
	Query: `query CompanyIDQuery { company { id } }`,
	Type:  pirateship.OperationQuery,
}, &result)
```

Queries use bounded exponential retry with jitter. Mutations are always sent
at most once. Every operation accepts `context.Context`, response bodies are
bounded, label files are streamed, and GraphQL partial data is decoded before
errors are returned.

## Development checks

```sh
make check
```

The opt-in integration test makes one anonymous public-rate request. It never
uses an account or sends a mutation:

```sh
go test -tags=integration ./integration
```

## Project layout

```text
.
├── internal/operations   GraphQL documents audited against the web client
├── internal/poll         context-aware asynchronous polling
├── examples              runnable and reusable examples
├── .github/workflows     tests, linting, vulnerability checks, and CodeQL
└── *.go                  stable public SDK surface
```

## Compatibility and support

This project is community-maintained and is not affiliated with Pirate Ship.
Open a GitHub issue with a sanitized request shape and GraphQL error when a
schema change is suspected. Do not include addresses, tracking numbers,
session values, payment details, or label URLs.

## License

MIT. See [`LICENSE`](LICENSE) and [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).
