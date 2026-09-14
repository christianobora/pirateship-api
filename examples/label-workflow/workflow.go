// Package labelworkflow demonstrates the authenticated purchase and download flow.
package labelworkflow

import (
	"context"
	"fmt"
	"io"
	"time"

	pirateship "github.com/christianobora/pirateship-api"
)

// PurchaseAndDownload selects the first available service, purchases its
// labels, and writes the first label artifact to destination. The caller must
// pass confirmCharge=true to cross the purchase safety boundary.
func PurchaseAndDownload(
	ctx context.Context,
	client *pirateship.Client,
	batchID pirateship.ID,
	destination io.Writer,
	confirmCharge bool,
) error {
	information, _, err := client.PurchaseInformation(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get purchase information: %w", err)
	}

	selections := make([]pirateship.RateSelection, 0, len(information.Batch.RateGroups))
	selectedSummaryIDs := make(map[string]string, len(information.Batch.RateGroups))
	for _, group := range information.Batch.RateGroups {
		if len(group.RateSummaries) == 0 {
			return fmt.Errorf("rate group %q has no services", group.GroupKey.String)
		}
		summary := group.RateSummaries[0]
		selection, err := pirateship.NewRateSelection(group, summary)
		if err != nil {
			return fmt.Errorf("select service for %q: %w", group.GroupKey.String, err)
		}
		selections = append(selections, selection)
		selectedSummaryIDs[group.GroupKey.String] = summary.UniqueID
	}

	labelTotal, err := pirateship.BillableLabelTotal(
		information.Batch.RateGroups,
		selectedSummaryIDs,
	)
	if err != nil {
		return fmt.Errorf("calculate label total: %w", err)
	}
	topUp, err := pirateship.MinimumCharge(
		labelTotal,
		information.Company.AccountBalance,
		information.Company.Settings.DefaultChargeAmount,
	)
	if err != nil {
		return fmt.Errorf("calculate account charge: %w", err)
	}
	result, _, err := client.PurchaseBatch(ctx, pirateship.PurchaseBatchRequest{
		BatchID:         batchID,
		RateSelection:   selections,
		PaymentSourceID: information.Company.Settings.DefaultPaymentSourceID,
		ShipDate:        information.Batch.ShipDate,
		TotalCharge:     topUp,
		ConfirmCharge:   confirmCharge,
	})
	if err != nil {
		return fmt.Errorf("purchase labels: %w", err)
	}
	if _, err := client.WaitForBatch(ctx, batchID, time.Second, pirateship.BatchStatusBilled); err != nil {
		return fmt.Errorf("wait for purchase: %w", err)
	}

	shipmentIDs := make([]pirateship.ID, 0, len(result.Shipments))
	for _, shipment := range result.Shipments {
		shipmentIDs = append(shipmentIDs, shipment.ID)
	}
	downloadID, _, err := client.TriggerLabels(ctx, pirateship.TriggerLabelsRequest{
		ShipmentIDs: shipmentIDs,
	})
	if err != nil {
		return fmt.Errorf("trigger labels: %w", err)
	}
	labels, err := client.WaitForLabels(
		ctx,
		pirateship.LabelsRequest{DownloadID: downloadID},
		time.Second,
	)
	if err != nil {
		return fmt.Errorf("wait for labels: %w", err)
	}
	if len(labels) == 0 {
		return fmt.Errorf("label job %q returned no files", downloadID)
	}
	if _, err := client.DownloadLabel(ctx, labels[0], destination); err != nil {
		return fmt.Errorf("download label: %w", err)
	}
	return nil
}
