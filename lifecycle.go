package pirateship

import (
	"context"
	"fmt"
	"time"

	"github.com/christianobora/pirateship-api/internal/operations"
	"github.com/christianobora/pirateship-api/internal/poll"
)

const defaultPollInterval = time.Second

// BatchStatus returns the latest server-side state of a batch.
func (c *Client) BatchStatus(ctx context.Context, batchID ID) (*Batch, *Response, error) {
	if batchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	var data struct {
		Batch Batch `json:"batch"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "BatchProcessStatusQuery",
		Query:     operations.BatchProcessStatus,
		Variables: map[string]ID{"id": batchID},
		Type:      OperationQuery,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("get batch status: %w", err)
	}
	return &data.Batch, response, nil
}

// WaitForBatch polls until the batch reaches one of terminalStatuses. When no
// statuses are supplied it waits for RATED, BILLED, ERROR, or REFUNDED.
func (c *Client) WaitForBatch(
	ctx context.Context,
	batchID ID,
	interval time.Duration,
	terminalStatuses ...BatchStatus,
) (*Batch, error) {
	if ctx == nil {
		return nil, &ValidationError{Field: "context", Problem: "must not be nil"}
	}
	if interval == 0 {
		interval = defaultPollInterval
	}
	if interval < 0 {
		return nil, &ValidationError{Field: "poll interval", Problem: "must not be negative"}
	}
	if len(terminalStatuses) == 0 {
		terminalStatuses = []BatchStatus{
			BatchStatusRated,
			BatchStatusBilled,
			BatchStatusError,
			BatchStatusRefunded,
		}
	}
	terminal := make(map[BatchStatus]struct{}, len(terminalStatuses))
	for _, status := range terminalStatuses {
		if status == "" {
			return nil, &ValidationError{Field: "terminal status", Problem: "must not be empty"}
		}
		terminal[status] = struct{}{}
	}

	batch, err := poll.Until(ctx, interval, func(ctx context.Context) (*Batch, bool, error) {
		current, _, err := c.BatchStatus(ctx, batchID)
		if err != nil {
			return nil, false, err
		}
		_, done := terminal[current.Status]
		return current, done, nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait for batch %q: %w", batchID, err)
	}
	return batch, nil
}

// RefundBatch requests a refund for every eligible label in a batch.
func (c *Client) RefundBatch(ctx context.Context, batchID ID) (*Batch, *Response, error) {
	if batchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	var data struct {
		Batch Batch `json:"refundBatch"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "RefundBatchMutation",
		Query:     operations.RefundBatch,
		Variables: map[string]ID{"id": batchID},
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("refund batch: %w", err)
	}
	return &data.Batch, response, nil
}

// RefundShipment requests a refund for one eligible shipment label.
func (c *Client) RefundShipment(
	ctx context.Context,
	shipmentID ID,
) (*Batch, *Response, error) {
	if shipmentID == "" {
		return nil, nil, &ValidationError{Field: "shipment ID", Problem: "is required"}
	}
	var data struct {
		Batch Batch `json:"refundShipment"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "RefundShipmentMutation",
		Query:     operations.RefundShipment,
		Variables: map[string]ID{"shipmentId": shipmentID},
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("refund shipment: %w", err)
	}
	return &data.Batch, response, nil
}
