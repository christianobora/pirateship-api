# Protocol audit

Audit date: 2026-09-14

This document records the Pirate Ship web-client contract used by this module.
The API is private and undocumented, so this is a compatibility snapshot, not
an upstream guarantee.

## Method

The audit compared three sources:

- the public rate client in `taciturnaxolotl/pirateship-api`;
- the current Pirate Ship web application's shipped JavaScript and source maps;
- read-only inspection of authenticated shipment and batch pages in the normal
  web interface.

The account was not mutated during the audit. No warehouse, preset, shipment,
purchase, label-rendering, refund, or account-settings mutation was sent. No
session cookie, browser storage, payment data, address, tracking number, or
label URL was copied into this repository.

## Transport

- GraphQL endpoint: `https://ship.pirateship.com/api/graphql`
- Method: `POST`
- Operation name query parameter: `opname`
- Authentication: normal same-origin web session cookies
- Authentication error code: `UNAUTHENTICATED`
- Web-client retry behavior: queries only; mutations are not retried

This module mirrors those retry semantics. It also bounds GraphQL response
bodies and preserves partial GraphQL data when errors are returned.

## Public rates

`RatesQuery` accepts origin/destination postal data, residential status,
weight, dimensions, mail class keys, package type keys, pricing types, and the
2x7 UPS visibility flag. Weight is expressed in ounces and dimensions in
inches.

The response includes carrier, mail class, package type, price components,
surcharges, zone, rate period, and cheapest/fastest/best flags. Carrier and
service identifiers remain open string types so schema additions do not break
decoding.

## Shipment setup

The typed setup surface includes:

- shipment configuration (warehouses, presets, carriers, package types, and
  current package limits);
- warehouse creation;
- shipment-preset creation and update;
- single-shipment batch creation;
- batch title update and eligible batch deletion.

Creating a single-shipment batch starts validation/rating but does not buy a
label.

## Rating and purchase

Observed batch states are `NEW`, `IMPORTED`, `VALIDATED`, `RATING`, `RATED`,
`PURCHASING`, `BILLED`, `ERROR`, `REFUNDING`, and `REFUNDED`.

Purchase rate choices are grouped by trait pairs. A service's opaque
`uniqueId` has the form `<mail-class-id>-<package-type-id>-<saturday-flag>`.
`NewRateSelection` validates and converts that value into the purchase input.

`BuyBatchMutation` accepts:

- batch ID;
- one rate selection per group;
- payment-source ID;
- ship date;
- account top-up amount;
- optional mail-template ID and recipient-notification date.

It can charge a saved payment source. `PurchaseBatch` therefore requires the
caller to set `ConfirmCharge: true`, and the transport will send the mutation
only once.

The web client calculates an account top-up as zero when the balance covers the
label total. Otherwise it uses the greatest of one dollar, the balance
shortfall, and the account's default charge amount. `MinimumCharge` implements
that calculation with cent rounding. `BillableLabelTotal` excludes return-only
groups, which are not charged during the initial purchase.

## Labels and refunds

Purchased shipment IDs are submitted to a label-rendering mutation, which
returns a download job ID. The labels query is polled until every artifact is
`FINISHED` or one is `ERROR`. Current formats are PDF, PNG, and ZPL; current
layouts include 2x7, 4x6, and US Letter one-up/two-up variants.

Artifact bytes are streamed instead of buffered. The web client's `/force/0`
download path is changed to `/force/1`; custom GraphQL headers are never sent
to the artifact host.

Batch and individual-shipment refund mutations are available. Callers can poll
the containing batch until `REFUNDED` or `ERROR`.

## Compatibility policy

Every typed method is backed by a named operation in `internal/operations`.
When the web schema changes, update the smallest affected operation and its
types, add a sanitized regression fixture, and release a new module version.
`Client.Do` remains available for fields that have not yet been added to the
typed surface.
