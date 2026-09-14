// Package pirateship provides a typed Go client for Pirate Ship's GraphQL API.
//
// The package supports public rate quotes and the authenticated label lifecycle,
// including shipment setup, rating, purchase, status polling, label generation, and
// refunds. Pirate Ship does not publish this API, so callers should pin a module
// version and handle GraphQL errors defensively.
package pirateship
