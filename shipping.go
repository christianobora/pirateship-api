package pirateship

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/christianobora/pirateship-api/internal/operations"
)

// ShippingConfiguration contains account-specific fields needed to create a
// shipment. It requires an authenticated HTTP client.
type ShippingConfiguration struct {
	Company            ShippingCompany    `json:"company"`
	ShipmentBoundaries ShipmentBoundaries `json:"shipmentBoundaries"`
	Carriers           []ShippingCarrier  `json:"carriers"`
	Countries          []Country          `json:"countries"`
}

// ShippingCompany contains reusable shipment settings for an account.
type ShippingCompany struct {
	ID              ID               `json:"id"`
	ActiveCarriers  []CarrierKey     `json:"activeCarriers"`
	Settings        ShippingSettings `json:"settings"`
	Warehouses      []Warehouse      `json:"warehouses"`
	ShipmentPresets []ShipmentPreset `json:"shipmentPresets"`
}

// ShippingSettings contains default IDs and label size.
type ShippingSettings struct {
	DefaultWarehouseID      ID     `json:"defaultWarehouseId"`
	DefaultShipmentPresetID ID     `json:"defaultShipmentPresetId"`
	ShipmentLabelSize       string `json:"shipmentLabelSize"`
}

// ShipmentBoundaries contains current carrier package limits.
type ShipmentBoundaries struct {
	MaxWeight          float64 `json:"maxWeight"`
	MaxCombinedLength  float64 `json:"maxCombinedLength"`
	MaxLengthPlusGirth float64 `json:"maxLengthPlusGirth"`
	MinLongSide        float64 `json:"minLongSide"`
	MaxLongSide        float64 `json:"maxLongSide"`
	MinMiddleSide      float64 `json:"minMiddleSide"`
	MaxMiddleSide      float64 `json:"maxMiddleSide"`
	MinShortSide       float64 `json:"minShortSide"`
	MaxShortSide       float64 `json:"maxShortSide"`
}

// ShippingCarrier describes a carrier and its supported package types.
type ShippingCarrier struct {
	ID           ID                   `json:"id"`
	Key          CarrierKey           `json:"carrierKey"`
	PackageTypes []CarrierPackageType `json:"packageTypes"`
}

// CarrierPackageType is a package type advertised for an account and carrier.
type CarrierPackageType struct {
	ID                 ID     `json:"id"`
	Key                string `json:"packageTypeKey"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	HeightRequired     bool   `json:"heightRequired"`
	WeightRequired     bool   `json:"weightRequired"`
	DimensionsRequired bool   `json:"dimensionsRequired"`
}

// Country is a country advertised by the shipment form.
type Country struct {
	ID               ID     `json:"id"`
	Name             string `json:"name"`
	Code             string `json:"countryCode"`
	Code3            string `json:"countryCode3"`
	PostcodeRequired bool   `json:"postcodeRequired"`
}

// WarehouseRequest is the input for CreateWarehouse.
type WarehouseRequest struct {
	Title                    string   `json:"title"`
	SaveAddressForReuse      bool     `json:"saveAddressForReuse"`
	UseOriginAsReturnAddress bool     `json:"useOriginAsReturnAddress"`
	OriginAddress            Address  `json:"originAddress"`
	ReturnAddress            *Address `json:"returnAddress"`
}

// ShipmentPresetRequest is the input for creating or updating a preset.
type ShipmentPresetRequest struct {
	Default bool                `json:"default"`
	Preset  ShipmentPresetInput `json:"shipmentPreset"`
}

// RubberStamps are optional text lines printed on a shipping label.
type RubberStamps struct {
	Line1 string `json:"rubberStamp1"`
	Line2 string `json:"rubberStamp2"`
}

// CreateShipmentRequest creates a single-shipment batch and begins validation
// and rating.
type CreateShipmentRequest struct {
	WarehouseID        ID               `json:"warehouseId"`
	ShipmentPresetID   ID               `json:"shipmentPresetId"`
	ShipToAddress      RecipientAddress `json:"shipToAddress"`
	RubberStamps       RubberStamps     `json:"rubberStamps"`
	ValidatedAddressID *ID              `json:"validatedAddressId,omitempty"`
	SkipCorrection     bool             `json:"skipCorrection"`
}

// ShippingConfiguration returns account shipment metadata.
func (c *Client) ShippingConfiguration(
	ctx context.Context,
) (*ShippingConfiguration, *Response, error) {
	var data ShippingConfiguration
	response, err := c.Do(ctx, Operation{
		Name:  "ShippingConfigurationQuery",
		Query: operations.ShippingConfiguration,
		Type:  OperationQuery,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("get shipping configuration: %w", err)
	}
	return &data, response, nil
}

// CreateWarehouse creates an origin/return-address pair. This is a
// state-changing authenticated operation.
func (c *Client) CreateWarehouse(
	ctx context.Context,
	request WarehouseRequest,
) (*Warehouse, *Response, error) {
	if strings.TrimSpace(request.Title) == "" {
		return nil, nil, &ValidationError{Field: "warehouse title", Problem: "must not be empty"}
	}
	if err := validateAddress(request.OriginAddress); err != nil {
		return nil, nil, fmt.Errorf("validate origin address: %w", err)
	}
	if !request.UseOriginAsReturnAddress {
		if request.ReturnAddress == nil {
			return nil, nil, &ValidationError{Field: "return address", Problem: "is required"}
		}
		if err := validateAddress(*request.ReturnAddress); err != nil {
			return nil, nil, fmt.Errorf("validate return address: %w", err)
		}
	}

	var data struct {
		Warehouse Warehouse `json:"createWarehouse"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "CreateWarehouseMutation",
		Query:     operations.CreateWarehouse,
		Variables: request,
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("create warehouse: %w", err)
	}
	return &data.Warehouse, response, nil
}

// CreateShipmentPreset creates reusable shipment settings.
func (c *Client) CreateShipmentPreset(
	ctx context.Context,
	request ShipmentPresetRequest,
) (*ShipmentPreset, *Response, error) {
	if request.Preset.ID != "" {
		return nil, nil, &ValidationError{Field: "shipment preset ID", Problem: "must be empty when creating"}
	}
	return c.saveShipmentPreset(ctx, "CreateShipmentPresetMutation", operations.CreateShipmentPreset, request)
}

// UpdateShipmentPreset updates reusable shipment settings.
func (c *Client) UpdateShipmentPreset(
	ctx context.Context,
	request ShipmentPresetRequest,
) (*ShipmentPreset, *Response, error) {
	if request.Preset.ID == "" {
		return nil, nil, &ValidationError{Field: "shipment preset ID", Problem: "is required when updating"}
	}
	return c.saveShipmentPreset(ctx, "UpdateShipmentPresetMutation", operations.UpdateShipmentPreset, request)
}

func (c *Client) saveShipmentPreset(
	ctx context.Context,
	operationName string,
	query string,
	request ShipmentPresetRequest,
) (*ShipmentPreset, *Response, error) {
	if err := validateShipmentPreset(request.Preset); err != nil {
		return nil, nil, err
	}

	data := make(map[string]ShipmentPreset, 1)
	response, err := c.Do(ctx, Operation{
		Name:      operationName,
		Query:     query,
		Variables: request,
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("save shipment preset: %w", err)
	}
	field := "createShipmentPreset"
	if operationName == "UpdateShipmentPresetMutation" {
		field = "updateShipmentPreset"
	}
	preset := data[field]
	return &preset, response, nil
}

// CreateShipment creates a batch containing one shipment. It does not buy a
// label or charge a payment method.
func (c *Client) CreateShipment(
	ctx context.Context,
	request CreateShipmentRequest,
) (*Batch, *Response, error) {
	if request.WarehouseID == "" {
		return nil, nil, &ValidationError{Field: "warehouse ID", Problem: "is required"}
	}
	if request.ShipmentPresetID == "" {
		return nil, nil, &ValidationError{Field: "shipment preset ID", Problem: "is required"}
	}
	if err := validateRecipientAddress(request.ShipToAddress); err != nil {
		return nil, nil, err
	}

	var data struct {
		Batch Batch `json:"createBatchFromSingleShipment"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "CreateBatchFromSingleShipmentMutation",
		Query:     operations.CreateBatchFromSingleShipment,
		Variables: request,
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("create shipment: %w", err)
	}
	return &data.Batch, response, nil
}

// UpdateBatchTitle changes an existing batch's display title.
func (c *Client) UpdateBatchTitle(
	ctx context.Context,
	id ID,
	title string,
) (*Batch, *Response, error) {
	if id == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	if strings.TrimSpace(title) == "" {
		return nil, nil, &ValidationError{Field: "batch title", Problem: "must not be empty"}
	}
	var data struct {
		Batch Batch `json:"updateBatchTitle"`
	}
	response, err := c.Do(ctx, Operation{
		Name:  "UpdateBatchTitleMutation",
		Query: operations.UpdateBatchTitle,
		Variables: struct {
			ID    ID     `json:"id"`
			Title string `json:"title"`
		}{id, title},
		Type: OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("update batch title: %w", err)
	}
	return &data.Batch, response, nil
}

// DeleteBatch removes an eligible, unpurchased batch.
func (c *Client) DeleteBatch(ctx context.Context, id ID) (*Response, error) {
	if id == "" {
		return nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	response, err := c.Do(ctx, Operation{
		Name:      "DeleteBatchMutation",
		Query:     operations.DeleteBatch,
		Variables: map[string]ID{"id": id},
		Type:      OperationMutation,
	}, nil)
	if err != nil {
		return response, fmt.Errorf("delete batch: %w", err)
	}
	return response, nil
}

func validateAddress(address Address) error {
	if strings.TrimSpace(address.FullName) == "" && strings.TrimSpace(address.Company) == "" {
		return &ValidationError{Field: "address name", Problem: "full name or company is required"}
	}
	for field, value := range map[string]string{
		"address line 1": address.Address1,
		"city":           address.City,
		"postcode":       address.Postcode,
		"country code":   address.CountryCode,
	} {
		if strings.TrimSpace(value) == "" {
			return &ValidationError{Field: field, Problem: "is required"}
		}
	}
	return nil
}

func validateRecipientAddress(address RecipientAddress) error {
	return validateAddress(Address{
		FullName: address.FullName, Company: address.Company, Address1: address.Address1,
		Address2: address.Address2, City: address.City, RegionCode: address.RegionCode,
		Postcode: address.Postcode, ZIP4: address.ZIP4, CountryCode: address.CountryCode,
		Phone: address.Phone,
	})
}

func validateShipmentPreset(preset ShipmentPresetInput) error {
	if strings.TrimSpace(preset.Title) == "" {
		return &ValidationError{Field: "shipment preset title", Problem: "must not be empty"}
	}
	if strings.TrimSpace(preset.PackageTypeKey) == "" {
		return &ValidationError{Field: "package type key", Problem: "is required"}
	}
	if err := validatePositiveFinite("weight", &preset.Weight); err != nil {
		return err
	}
	for field, value := range map[string]float64{
		"dimension X":   preset.DimensionX,
		"dimension Y":   preset.DimensionY,
		"dimension Z":   preset.DimensionZ,
		"insured value": preset.InsuredValue,
	} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return &ValidationError{Field: field, Problem: "must be finite and non-negative"}
		}
	}
	return nil
}
