package pirateship

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTypedOperations(t *testing.T) {
	t.Parallel()

	responses := map[string]string{
		"ShippingConfigurationQuery":               `{"company":{"id":"company"},"shipmentBoundaries":{},"carriers":[],"countries":[]}`,
		"CreateWarehouseMutation":                  `{"createWarehouse":{"id":"warehouse","title":"Main"}}`,
		"CreateShipmentPresetMutation":             `{"createShipmentPreset":{"id":"preset","title":"Parcel"}}`,
		"UpdateShipmentPresetMutation":             `{"updateShipmentPreset":{"id":"preset","title":"Parcel"}}`,
		"CreateBatchFromSingleShipmentMutation":    `{"createBatchFromSingleShipment":{"id":"batch","status":"RATING"}}`,
		"UpdateBatchTitleMutation":                 `{"updateBatchTitle":{"id":"batch","title":"New title"}}`,
		"DeleteBatchMutation":                      `{"deleteBatch":{"id":"batch"}}`,
		"PurchaseInformationQuery":                 `{"batch":{"id":"batch","rateGroups":[]},"company":{"id":"company","paymentSources":[]}}`,
		"RerateBatchMutation":                      `{"rerateBatch":{"id":"batch","status":"RATING"}}`,
		"ModifyInsuranceOnRatedBatchMutation":      `{"modifyInsuranceOnRatedBatch":{"id":"batch","rateGroups":[]}}`,
		"BatchProcessStatusQuery":                  `{"batch":{"id":"batch","status":"RATED"}}`,
		"RefundBatchMutation":                      `{"refundBatch":{"id":"batch","status":"REFUNDING"}}`,
		"RefundShipmentMutation":                   `{"refundShipment":{"id":"batch","shipments":[{"id":"shipment","status":"REFUND_REQUESTED"}]}}`,
		"TriggerDownloadLabelsByShipmentsMutation": `{"triggerLabelCreationByShipments":"download"}`,
		"LabelsQuery":                              `{"labels":[{"id":"label","status":"FINISHED","fileFormat":"PDF","pageLayout":"LAYOUT_4x6","url":"https://example.com/force/0/label.pdf"}]}`,
		"DownloadMobileCodeForShipment":            `{"downloadMobileCodeForShipment":{"mobileCodeDataUri":"data:image/png;base64,AA=="}}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(writer, "bad request", http.StatusBadRequest)
			return
		}
		data, ok := responses[payload.OperationName]
		if !ok {
			t.Errorf("unexpected operation %q", payload.OperationName)
			http.Error(writer, "unexpected operation", http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprintf(writer, `{"data":%s}`, data)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	ctx := context.Background()

	configuration, _, err := client.ShippingConfiguration(ctx)
	assertNoError(t, err)
	if configuration.Company.ID != "company" {
		t.Errorf("company ID = %q", configuration.Company.ID)
	}

	address := Address{
		FullName: "Example Recipient", Address1: "1 Main St", Address2: "",
		City: "San Francisco", RegionCode: "CA", Postcode: "94105", CountryCode: "US",
	}
	warehouse, _, err := client.CreateWarehouse(ctx, WarehouseRequest{
		Title: "Main", UseOriginAsReturnAddress: true, OriginAddress: address,
	})
	assertNoError(t, err)
	if warehouse.ID != "warehouse" {
		t.Errorf("warehouse ID = %q", warehouse.ID)
	}

	presetInput := ShipmentPresetInput{
		Title: "Parcel", PackageTypeKey: "Parcel", Weight: 16,
		DeliveryConfirmation: DeliveryConfirmationNone, ReturnLabel: ReturnLabelStandard,
	}
	preset, _, err := client.CreateShipmentPreset(ctx, ShipmentPresetRequest{Preset: presetInput})
	assertNoError(t, err)
	if preset.ID != "preset" {
		t.Errorf("preset ID = %q", preset.ID)
	}
	presetInput.ID = "preset"
	_, _, err = client.UpdateShipmentPreset(ctx, ShipmentPresetRequest{Preset: presetInput})
	assertNoError(t, err)

	batch, _, err := client.CreateShipment(ctx, CreateShipmentRequest{
		WarehouseID: "warehouse", ShipmentPresetID: "preset",
		ShipToAddress: RecipientAddress{
			FullName: "Example Recipient", Address1: "1 Main St", Address2: "",
			City: "New York", RegionCode: "NY", Postcode: "10001", CountryCode: "US",
			Email: "recipient@example.com", Phone: "",
		},
	})
	assertNoError(t, err)
	if batch.ID != "batch" {
		t.Errorf("batch ID = %q", batch.ID)
	}

	_, _, err = client.UpdateBatchTitle(ctx, "batch", "New title")
	assertNoError(t, err)
	_, err = client.DeleteBatch(ctx, "batch")
	assertNoError(t, err)

	information, _, err := client.PurchaseInformation(ctx, "batch")
	assertNoError(t, err)
	if information.Company.ID != "company" {
		t.Errorf("purchase company ID = %q", information.Company.ID)
	}
	_, _, err = client.RerateBatch(ctx, "batch", nil)
	assertNoError(t, err)
	_, _, err = client.ModifyInsurance(ctx, "batch", 100)
	assertNoError(t, err)

	status, _, err := client.BatchStatus(ctx, "batch")
	assertNoError(t, err)
	if status.Status != BatchStatusRated {
		t.Errorf("batch status = %q", status.Status)
	}
	_, _, err = client.RefundBatch(ctx, "batch")
	assertNoError(t, err)
	_, _, err = client.RefundShipment(ctx, "shipment")
	assertNoError(t, err)

	downloadID, _, err := client.TriggerLabels(ctx, TriggerLabelsRequest{ShipmentIDs: []ID{"shipment"}})
	assertNoError(t, err)
	if downloadID != "download" {
		t.Errorf("download ID = %q", downloadID)
	}
	labels, _, err := client.Labels(ctx, LabelsRequest{DownloadID: downloadID})
	assertNoError(t, err)
	if len(labels) != 1 || labels[0].ID != "label" {
		t.Errorf("labels = %#v", labels)
	}
	dataURI, _, err := client.DownloadMobileCode(ctx, "shipment")
	assertNoError(t, err)
	if dataURI != "data:image/png;base64,AA==" {
		t.Errorf("mobile code = %q", dataURI)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
