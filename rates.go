package pirateship

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/christianobora/pirateship-api/internal/operations"
)

// PackageTypeKey identifies a carrier package type. It is open-ended because
// authenticated accounts can receive additional carrier-specific values.
type PackageTypeKey string

// Common public package type keys.
const (
	PackageTypeParcel                  PackageTypeKey = "Parcel"
	PackageTypeSoftEnvelope            PackageTypeKey = "SoftEnvelope"
	PackageTypeIrregular               PackageTypeKey = "Irregular"
	PackageTypeFlatRateEnvelope        PackageTypeKey = "FlatRateEnvelope"
	PackageTypeFlatRateLegalEnvelope   PackageTypeKey = "FlatRateLegalEnvelope"
	PackageTypeFlatRatePaddedEnvelope  PackageTypeKey = "FlatRatePaddedEnvelope"
	PackageTypeSmallFlatRateBox        PackageTypeKey = "SmallFlatRateBox"
	PackageTypeMediumFlatRateBox       PackageTypeKey = "MediumFlatRateBox"
	PackageTypeLargeFlatRateBox        PackageTypeKey = "LargeFlatRateBox"
	PackageTypeExpressFlatRateEnvelope PackageTypeKey = "ExpressFlatRateEnvelope"
	PackageTypeExpressFlatRateLegal    PackageTypeKey = "ExpressFlatRateLegalEnvelope"
	PackageTypeExpressFlatRatePadded   PackageTypeKey = "ExpressFlatRatePaddedEnvelope"
)

// MailClassKey identifies a carrier service. It is open-ended to tolerate new
// services without requiring a client release.
type MailClassKey string

// Common public mail class keys.
const (
	MailClassPriorityExpress MailClassKey = "PriorityExpress"
	MailClassFirst           MailClassKey = "First"
	MailClassParcelSelect    MailClassKey = "ParcelSelect"
	MailClassPriority        MailClassKey = "Priority"
	MailClassGroundAdvantage MailClassKey = "GroundAdvantage"
	MailClassMediaMail       MailClassKey = "MediaMail"
)

// RatesRequest describes a public rate estimate. Weight is in ounces and
// dimensions are in inches.
type RatesRequest struct {
	OriginZIP                   string           `json:"originZip"`
	OriginCity                  *string          `json:"originCity,omitempty"`
	OriginRegionCode            *string          `json:"originRegionCode,omitempty"`
	DestinationZIP              *string          `json:"destinationZip,omitempty"`
	Residential                 *bool            `json:"isResidential,omitempty"`
	DestinationCountryCode      *string          `json:"destinationCountryCode,omitempty"`
	WeightOunces                *float64         `json:"weight,omitempty"`
	DimensionXInches            *float64         `json:"dimensionX,omitempty"`
	DimensionYInches            *float64         `json:"dimensionY,omitempty"`
	DimensionZInches            *float64         `json:"dimensionZ,omitempty"`
	MailClassKeys               []MailClassKey   `json:"mailClassKeys"`
	PackageTypeKeys             []PackageTypeKey `json:"packageTypeKeys"`
	PricingTypes                []string         `json:"pricingTypes,omitempty"`
	ShowUPSRatesWhen2x7Selected *bool            `json:"showUpsRatesWhen2x7Selected,omitempty"`
}

// Rate is one carrier service estimate.
type Rate struct {
	Title               string         `json:"title"`
	DeliveryDescription string         `json:"deliveryDescription"`
	TrackingDescription string         `json:"trackingDescription"`
	ServiceDescription  string         `json:"serviceDescription"`
	PricingDescription  string         `json:"pricingDescription"`
	CubicTier           *string        `json:"cubicTier"`
	MailClassKey        MailClassKey   `json:"mailClassKey"`
	MailClass           MailClass      `json:"mailClass"`
	PackageTypeKey      PackageTypeKey `json:"packageTypeKey"`
	Zone                string         `json:"zone"`
	Surcharges          []Surcharge    `json:"surcharges"`
	Carrier             Carrier        `json:"carrier"`
	TotalPrice          float64        `json:"totalPrice"`
	PriceBaseTypeKey    string         `json:"priceBaseTypeKey"`
	BasePrice           float64        `json:"basePrice"`
	CrossedTotalPrice   float64        `json:"crossedTotalPrice"`
	PricingType         string         `json:"pricingType"`
	PricingSubType      string         `json:"pricingSubType"`
	RatePeriodID        int            `json:"ratePeriodId"`
	LearnMoreURL        string         `json:"learnMoreUrl"`
	Cheapest            bool           `json:"cheapest"`
	Fastest             bool           `json:"fastest"`
	Best                bool           `json:"best"`
}

// MailClass contains characteristics of a rate's service.
type MailClass struct {
	Accuracy      *string `json:"accuracy"`
	International bool    `json:"international"`
}

// Carrier identifies the carrier for a rate.
type Carrier struct {
	Key   CarrierKey `json:"carrierKey"`
	Title string     `json:"title"`
}

// Surcharge is one component of a quoted rate.
type Surcharge struct {
	Title string  `json:"title"`
	Price float64 `json:"price"`
}

// Rates returns public rate estimates. Authentication is not required.
func (c *Client) Rates(ctx context.Context, request RatesRequest) ([]Rate, *Response, error) {
	if err := request.validate(); err != nil {
		return nil, nil, err
	}

	var data struct {
		Rates []Rate `json:"rates"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "RatesQuery",
		Query:     operations.Rates,
		Variables: request,
		Type:      OperationQuery,
	}, &data)
	if err != nil {
		return data.Rates, response, fmt.Errorf("get rates: %w", err)
	}
	return data.Rates, response, nil
}

func (r RatesRequest) validate() error {
	if strings.TrimSpace(r.OriginZIP) == "" {
		return &ValidationError{Field: "origin ZIP", Problem: "must not be empty"}
	}
	if len(r.MailClassKeys) == 0 {
		return &ValidationError{Field: "mail class keys", Problem: "must not be empty"}
	}
	if len(r.PackageTypeKeys) == 0 {
		return &ValidationError{Field: "package type keys", Problem: "must not be empty"}
	}
	if r.WeightOunces == nil {
		return &ValidationError{Field: "weight", Problem: "is required"}
	}
	if err := validatePositiveFinite("weight", r.WeightOunces); err != nil {
		return err
	}
	for field, value := range map[string]*float64{
		"dimension X": r.DimensionXInches,
		"dimension Y": r.DimensionYInches,
		"dimension Z": r.DimensionZInches,
	} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0) {
			return &ValidationError{Field: field, Problem: "must be finite and non-negative"}
		}
	}

	for _, packageType := range r.PackageTypeKeys {
		if isFlatRatePackage(packageType) {
			continue
		}
		if r.DimensionXInches == nil {
			return &ValidationError{Field: "dimension X", Problem: "is required for this package type"}
		}
		if r.DimensionYInches == nil {
			return &ValidationError{Field: "dimension Y", Problem: "is required for this package type"}
		}
		if packageType != PackageTypeSoftEnvelope && r.DimensionZInches == nil {
			return &ValidationError{Field: "dimension Z", Problem: "is required for this package type"}
		}
		if r.DimensionXInches != nil && *r.DimensionXInches < 6 {
			return &ValidationError{Field: "dimension X", Problem: "must be at least 6 inches"}
		}
		if r.DimensionYInches != nil && *r.DimensionYInches < 3 {
			return &ValidationError{Field: "dimension Y", Problem: "must be at least 3 inches"}
		}
		if packageType != PackageTypeSoftEnvelope && r.DimensionZInches != nil && *r.DimensionZInches < 0.25 {
			return &ValidationError{Field: "dimension Z", Problem: "must be at least 0.25 inches"}
		}
	}
	return nil
}

func validatePositiveFinite(field string, value *float64) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value <= 0 {
		return &ValidationError{Field: field, Problem: "must be finite and positive"}
	}
	return nil
}

func isFlatRatePackage(key PackageTypeKey) bool {
	switch key {
	case PackageTypeFlatRateEnvelope,
		PackageTypeFlatRateLegalEnvelope,
		PackageTypeFlatRatePaddedEnvelope,
		PackageTypeSmallFlatRateBox,
		PackageTypeMediumFlatRateBox,
		PackageTypeLargeFlatRateBox,
		PackageTypeExpressFlatRateEnvelope,
		PackageTypeExpressFlatRateLegal,
		PackageTypeExpressFlatRatePadded:
		return true
	default:
		return false
	}
}
