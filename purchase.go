package pirateship

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/christianobora/pirateship-api/internal/operations"
)

var rateSummaryIDPattern = regexp.MustCompile(`^(\d+)-(\d+)-(\d+)$`)

// PurchaseInformation contains rated services and the account data needed to
// explicitly construct a PurchaseBatchRequest.
type PurchaseInformation struct {
	Batch   PurchaseBatch   `json:"batch"`
	Company PurchaseCompany `json:"company"`
}

// PurchaseBatch contains rate groups for an existing batch.
type PurchaseBatch struct {
	Batch
	EligibleForInsurance      bool        `json:"eligibleForInsurance"`
	EmailNotificationPossible bool        `json:"emailNotificationPossible"`
	LabelSize                 string      `json:"labelSize"`
	LabelFileFormat           string      `json:"labelFileFormat"`
	LabelingArtifactType      string      `json:"labelingArtifactType"`
	RateGroups                []RateGroup `json:"rateGroups"`
}

// PurchaseCompany contains balance, defaults, and payment-source metadata.
type PurchaseCompany struct {
	ID                       ID                `json:"id"`
	AccountBalance           float64           `json:"accountBalance"`
	HasAnyPlaidPaymentSource bool              `json:"hasAnyPlaidPaymentSource"`
	IsFirstLabel             bool              `json:"isFirstLabel"`
	ActiveCarriers           []CarrierKey      `json:"activeCarriers"`
	Settings                 PurchaseSettings  `json:"settings"`
	PaymentSources           []PaymentSource   `json:"paymentSources"`
	MerchantAccounts         []MerchantAccount `json:"merchantAccounts"`
	MailTemplates            []MailTemplate    `json:"mailTemplates"`
}

// PurchaseSettings contains purchase defaults.
type PurchaseSettings struct {
	DefaultPaymentSourceID       ID      `json:"defaultPaymentSourceId"`
	DefaultChargeAmount          float64 `json:"defaultChargeAmount"`
	DefaultTrackingEmailsEnabled bool    `json:"defaultTrackingEmailsEnabled"`
	DefaultTrackingEmailsDelay   string  `json:"defaultTrackingEmailsDelay"`
	DefaultEmailTemplateID       ID      `json:"defaultEmailTemplateId"`
}

// PaymentSource contains display metadata for an account payment source.
type PaymentSource struct {
	ID                ID      `json:"id"`
	PaymentMethodType string  `json:"paymentMethodType"`
	ValidationStatus  string  `json:"validationStatus"`
	Brand             string  `json:"brand"`
	Last4             string  `json:"last4"`
	ExpirationMonth   int     `json:"expMonth"`
	ExpirationYear    int     `json:"expYear"`
	ResultMessage     *string `json:"resultMessage"`
	Email             string  `json:"email"`
	Title             string  `json:"title"`
	Nickname          string  `json:"nickname"`
	RefundableAmount  float64 `json:"refundableAmount"`
	HasConsented      bool    `json:"hasConsented"`
}

// MerchantAccount identifies an enabled carrier account.
type MerchantAccount struct {
	ID         ID         `json:"id"`
	CarrierKey CarrierKey `json:"carrierKey"`
}

// MailTemplate identifies a recipient-notification template.
type MailTemplate struct {
	ID          ID     `json:"id"`
	Subject     string `json:"subject"`
	Name        string `json:"name"`
	SenderEmail string `json:"senderEmail"`
	SenderName  string `json:"senderName"`
	Default     bool   `json:"asDefault"`
}

// RateGroup contains services that can be chosen together for a batch subset.
type RateGroup struct {
	GroupKey               RateGroupKey  `json:"groupKey"`
	AffectedByUPSRateLimit bool          `json:"affectedByUpsRateLimit"`
	MaximumShipments       int           `json:"maximumShipments"`
	DefaultShipDate        string        `json:"defaultShipDate"`
	RateSummaries          []RateSummary `json:"rateSummaries"`
}

// RateGroupKey identifies the shipment traits affected by a service choice.
type RateGroupKey struct {
	String string           `json:"string"`
	Traits []RateGroupTrait `json:"traits"`
}

// RateGroupTrait is one layer/value pair in a rate-group key.
type RateGroupTrait struct {
	Layer string `json:"layer"`
	Value string `json:"value"`
}

// RateSummary is an account-specific service choice for a rate group.
type RateSummary struct {
	UniqueID            string                 `json:"uniqueId"`
	MaxWeightOunces     float64                `json:"maxWeightOz"`
	MailClassTitle      string                 `json:"mailClassTitle"`
	ServiceTitle        string                 `json:"serviceTitle"`
	DeliveryDays        int                    `json:"deliveryDays"`
	TotalPrice          float64                `json:"totalPrice"`
	BasePrice           float64                `json:"basePrice"`
	AverageBasePrice    float64                `json:"averageBasePrice"`
	FlatPrice           bool                   `json:"flatPrice"`
	CrossedTotalPrice   float64                `json:"crossedTotalPrice"`
	AverageTotalPrice   float64                `json:"averageTotalPrice"`
	Cheapest            bool                   `json:"cheapest"`
	Best                bool                   `json:"best"`
	Fastest             bool                   `json:"fastest"`
	Savings             string                 `json:"savings"`
	ShipmentCount       int                    `json:"shipmentCount"`
	FirstZone           string                 `json:"firstZone"`
	RatePeriodStartDate *string                `json:"ratePeriodStartDate"`
	RatePeriodEndDate   *string                `json:"ratePeriodEndDate"`
	ErrorMessage        *string                `json:"errorMessage"`
	ValueLimit          *float64               `json:"valueLimit"`
	Carrier             Carrier                `json:"carrier"`
	FirstMailClass      RateSummaryMailClass   `json:"firstMailClass"`
	PackageType         RateSummaryPackageType `json:"packageType"`
	AvailableShipDates  []string               `json:"availableShipDates"`
	Surcharges          []RateSummarySurcharge `json:"surcharges"`
}

// RateSummaryMailClass identifies the server-side service record.
type RateSummaryMailClass struct {
	Key MailClassKey `json:"mailClassKey"`
}

// RateSummaryPackageType identifies the server-side package record.
type RateSummaryPackageType struct {
	Key PackageTypeKey `json:"packageTypeKey"`
}

// RateSummarySurcharge is a component of an authenticated rate summary.
type RateSummarySurcharge struct {
	Key          string   `json:"surchargeKey"`
	Title        string   `json:"title"`
	Price        float64  `json:"price"`
	HelpLink     *string  `json:"helpLink"`
	CrossedPrice *float64 `json:"crossedPrice"`
}

// RateSelection is one chosen service for one group of shipments.
type RateSelection struct {
	GroupKeyInput      RateGroupKeyInput `json:"groupKeyInput"`
	MailClassID        ID                `json:"mailClassId"`
	PackageTypeID      ID                `json:"packageTypeId"`
	IsSaturdayDelivery bool              `json:"isSaturdayDelivery"`
}

// RateGroupKeyInput is the input form of a RateGroupKey.
type RateGroupKeyInput struct {
	Traits []RateGroupTrait `json:"traits"`
}

// PurchaseBatchRequest purchases labels and may charge the selected payment
// source. ConfirmCharge must be true as an explicit safety interlock.
type PurchaseBatchRequest struct {
	BatchID              ID              `json:"id"`
	RateSelection        []RateSelection `json:"rateSelection"`
	PaymentSourceID      ID              `json:"paymentSourceId"`
	ShipDate             string          `json:"shipDate"`
	TotalCharge          float64         `json:"totalCharge"`
	MailTemplateID       *ID             `json:"mailTemplateId,omitempty"`
	NotifyRecipientsDate *string         `json:"notifyRecipientsDate,omitempty"`
	ConfirmCharge        bool            `json:"-"`
}

// PurchaseResult contains the batch and shipment IDs returned after purchase.
type PurchaseResult struct {
	Batch
}

// PurchaseInformation returns rates and payment-source IDs for a rated batch.
// This query does not buy labels or charge the account.
func (c *Client) PurchaseInformation(
	ctx context.Context,
	batchID ID,
) (*PurchaseInformation, *Response, error) {
	if batchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	var data PurchaseInformation
	response, err := c.Do(ctx, Operation{
		Name:      "PurchaseInformationQuery",
		Query:     operations.PurchaseInformation,
		Variables: map[string]ID{"id": batchID},
		Type:      OperationQuery,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("get purchase information: %w", err)
	}
	return &data, response, nil
}

// RerateBatch starts a fresh rate calculation. shipDate may be nil to let the
// server choose its current default. This operation does not buy labels.
func (c *Client) RerateBatch(
	ctx context.Context,
	batchID ID,
	shipDate *string,
) (*Batch, *Response, error) {
	if batchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	var data struct {
		Batch Batch `json:"rerateBatch"`
	}
	response, err := c.Do(ctx, Operation{
		Name:  "RerateBatchMutation",
		Query: operations.RerateBatch,
		Variables: struct {
			ID       ID      `json:"id"`
			ShipDate *string `json:"shipDate"`
		}{batchID, shipDate},
		Type: OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("rerate batch: %w", err)
	}
	return &data.Batch, response, nil
}

// ModifyInsurance changes insurance on a rated batch and returns refreshed
// rate groups. It does not purchase labels.
func (c *Client) ModifyInsurance(
	ctx context.Context,
	batchID ID,
	insuredValue float64,
) (*PurchaseBatch, *Response, error) {
	if batchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	if insuredValue < 0 || math.IsNaN(insuredValue) || math.IsInf(insuredValue, 0) {
		return nil, nil, &ValidationError{Field: "insured value", Problem: "must be finite and non-negative"}
	}
	var data struct {
		Batch PurchaseBatch `json:"modifyInsuranceOnRatedBatch"`
	}
	response, err := c.Do(ctx, Operation{
		Name:  "ModifyInsuranceOnRatedBatchMutation",
		Query: operations.ModifyInsuranceOnRatedBatch,
		Variables: struct {
			ID           ID      `json:"id"`
			InsuredValue float64 `json:"insuredValue"`
		}{batchID, insuredValue},
		Type: OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("modify insurance: %w", err)
	}
	return &data.Batch, response, nil
}

// NewRateSelection converts a rate summary's opaque selection ID into the
// exact GraphQL purchase input for its group.
func NewRateSelection(group RateGroup, summary RateSummary) (RateSelection, error) {
	matches := rateSummaryIDPattern.FindStringSubmatch(summary.UniqueID)
	if len(matches) != 4 {
		return RateSelection{}, &ValidationError{
			Field: "rate summary ID", Problem: "must have format <mail-class>-<package-type>-<saturday>",
		}
	}
	if matches[3] != "0" && matches[3] != "1" {
		return RateSelection{}, &ValidationError{Field: "rate summary ID", Problem: "Saturday flag must be 0 or 1"}
	}
	return RateSelection{
		GroupKeyInput:      RateGroupKeyInput{Traits: append([]RateGroupTrait(nil), group.GroupKey.Traits...)},
		MailClassID:        ID(matches[1]),
		PackageTypeID:      ID(matches[2]),
		IsSaturdayDelivery: matches[3] == "1",
	}, nil
}

// MinimumCharge calculates the amount Pirate Ship's current purchase screen
// would add to an account balance for a label total.
func MinimumCharge(labelTotal, accountBalance, defaultCharge float64) (float64, error) {
	for field, value := range map[string]float64{
		"label total": labelTotal, "account balance": accountBalance, "default charge": defaultCharge,
	} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, &ValidationError{Field: field, Problem: "must be finite and non-negative"}
		}
	}
	shortfall := roundCurrency(labelTotal - accountBalance)
	if shortfall <= 0 {
		return 0, nil
	}
	return roundCurrency(max(1, shortfall, defaultCharge)), nil
}

// BillableLabelTotal returns the sum of selected service prices, excluding
// return-only groups because Pirate Ship does not charge for those labels at
// initial purchase time. The map keys are RateGroupKey.String values and the
// values are RateSummary.UniqueID values.
func BillableLabelTotal(
	rateGroups []RateGroup,
	selectedSummaryIDs map[string]string,
) (float64, error) {
	if len(rateGroups) == 0 {
		return 0, &ValidationError{Field: "rate groups", Problem: "must not be empty"}
	}
	if len(selectedSummaryIDs) != len(rateGroups) {
		return 0, &ValidationError{Field: "selected summaries", Problem: "must contain one selection per rate group"}
	}

	total := 0.0
	for _, group := range rateGroups {
		selectedID, ok := selectedSummaryIDs[group.GroupKey.String]
		if !ok {
			return 0, &ValidationError{
				Field: "selected summaries", Problem: fmt.Sprintf("missing rate group %q", group.GroupKey.String),
			}
		}
		var selected *RateSummary
		for index := range group.RateSummaries {
			if group.RateSummaries[index].UniqueID == selectedID {
				selected = &group.RateSummaries[index]
				break
			}
		}
		if selected == nil {
			return 0, &ValidationError{
				Field: "selected summaries", Problem: fmt.Sprintf("unknown summary %q for rate group %q", selectedID, group.GroupKey.String),
			}
		}
		if isReturnOnlyGroup(group) {
			continue
		}
		total += selected.TotalPrice
	}
	return roundCurrency(total), nil
}

// PurchaseBatch purchases labels. Unlike all query and setup methods, this can
// charge PaymentSourceID. Mutations are never automatically retried.
func (c *Client) PurchaseBatch(
	ctx context.Context,
	request PurchaseBatchRequest,
) (*PurchaseResult, *Response, error) {
	if !request.ConfirmCharge {
		return nil, nil, &ValidationError{
			Field: "purchase confirmation", Problem: "ConfirmCharge must be true",
		}
	}
	if request.BatchID == "" {
		return nil, nil, &ValidationError{Field: "batch ID", Problem: "is required"}
	}
	if request.PaymentSourceID == "" {
		return nil, nil, &ValidationError{Field: "payment source ID", Problem: "is required"}
	}
	if strings.TrimSpace(request.ShipDate) == "" {
		return nil, nil, &ValidationError{Field: "ship date", Problem: "is required"}
	}
	if len(request.RateSelection) == 0 {
		return nil, nil, &ValidationError{Field: "rate selection", Problem: "must not be empty"}
	}
	if request.TotalCharge < 0 || math.IsNaN(request.TotalCharge) || math.IsInf(request.TotalCharge, 0) {
		return nil, nil, &ValidationError{Field: "total charge", Problem: "must be finite and non-negative"}
	}

	var data struct {
		Result PurchaseResult `json:"buyBatch"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "BuyBatchMutation",
		Query:     operations.BuyBatch,
		Variables: request,
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return nil, response, fmt.Errorf("purchase batch: %w", err)
	}
	return &data.Result, response, nil
}

func roundCurrency(value float64) float64 {
	return math.Round(value*100) / 100
}

func isReturnOnlyGroup(group RateGroup) bool {
	for _, trait := range group.GroupKey.Traits {
		if strings.EqualFold(trait.Layer, "Direction") && strings.EqualFold(trait.Value, "RETURN") {
			return true
		}
	}
	return false
}
