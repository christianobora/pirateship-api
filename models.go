package pirateship

// ID is an opaque Pirate Ship GraphQL identifier.
type ID string

// CarrierKey identifies a shipping carrier. It is intentionally open-ended so
// new carriers remain forward-compatible.
type CarrierKey string

// Supported carrier keys.
const (
	CarrierFedEx CarrierKey = "fedex"
	CarrierUPS   CarrierKey = "ups"
	CarrierUSPS  CarrierKey = "usps"
)

// BatchStatus is the server-side state of a shipment batch.
type BatchStatus string

// Observed batch states.
const (
	BatchStatusNew        BatchStatus = "NEW"
	BatchStatusImported   BatchStatus = "IMPORTED"
	BatchStatusValidated  BatchStatus = "VALIDATED"
	BatchStatusRating     BatchStatus = "RATING"
	BatchStatusRated      BatchStatus = "RATED"
	BatchStatusPurchasing BatchStatus = "PURCHASING"
	BatchStatusBilled     BatchStatus = "BILLED"
	BatchStatusError      BatchStatus = "ERROR"
	BatchStatusRefunding  BatchStatus = "REFUNDING"
	BatchStatusRefunded   BatchStatus = "REFUNDED"
)

// DeliveryConfirmation identifies a signature service.
type DeliveryConfirmation string

// Supported delivery-confirmation values.
const (
	DeliveryConfirmationAdultSignature  DeliveryConfirmation = "adult_signature"
	DeliveryConfirmationDirectSignature DeliveryConfirmation = "direct_signature"
	DeliveryConfirmationNone            DeliveryConfirmation = "none"
	DeliveryConfirmationSignature       DeliveryConfirmation = "signature"
)

// ReturnLabel controls whether a preset creates standard and/or return labels.
type ReturnLabel string

// Supported return-label modes.
const (
	ReturnLabelReturn            ReturnLabel = "return"
	ReturnLabelStandard          ReturnLabel = "standard"
	ReturnLabelStandardAndReturn ReturnLabel = "standard_and_return"
)

// Address is an origin or return address. Empty optional strings are accepted
// because Pirate Ship's GraphQL inputs require several address keys even when
// they have no value.
type Address struct {
	ID          ID     `json:"id,omitempty"`
	FullName    string `json:"fullName"`
	Company     string `json:"company"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	RegionCode  string `json:"regionCode"`
	Postcode    string `json:"postcode"`
	ZIP4        string `json:"zip4,omitempty"`
	CountryCode string `json:"countryCode"`
	Phone       string `json:"phone,omitempty"`
}

// RecipientAddress is a shipment destination.
type RecipientAddress struct {
	FullName    string `json:"fullName"`
	Company     string `json:"company"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	RegionCode  string `json:"regionCode"`
	Postcode    string `json:"postcode"`
	ZIP4        string `json:"zip4,omitempty"`
	CountryCode string `json:"countryCode"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
}

// CustomsItem describes one line on an international customs form.
type CustomsItem struct {
	ID                  ID      `json:"id,omitempty"`
	Title               string  `json:"title"`
	Quantity            int     `json:"quantity"`
	ItemValue           float64 `json:"itemValue"`
	Weight              float64 `json:"weight"`
	HSTariffNumber      string  `json:"hsTariffNumber"`
	CountryCodeOfOrigin string  `json:"countryCodeOfOrigin"`
}

// ShipmentPresetInput is the GraphQL input for creating or updating reusable
// package, service, insurance, and customs settings.
type ShipmentPresetInput struct {
	ID                       ID                   `json:"id,omitempty"`
	Title                    string               `json:"title"`
	PackageTypeKey           string               `json:"packageTypeKey"`
	Weight                   float64              `json:"weight"`
	DimensionX               float64              `json:"dimensionX"`
	DimensionY               float64              `json:"dimensionY"`
	DimensionZ               float64              `json:"dimensionZ"`
	InsuredValueFlag         bool                 `json:"insuredValueFlag"`
	InsuredValue             float64              `json:"insuredValue"`
	DeliveryConfirmationFlag bool                 `json:"deliveryConfirmationFlag"`
	DeliveryConfirmation     DeliveryConfirmation `json:"deliveryConfirmation"`
	ReturnLabelFlag          bool                 `json:"returnLabelFlag"`
	ReturnLabel              ReturnLabel          `json:"returnLabel"`
	QualifiesAsMediaMail     bool                 `json:"qualifiesAsMediaMail"`
	IrregularPackage         bool                 `json:"irregularPackage"`
	HazardousMaterials       bool                 `json:"hazardousMaterialsEnabled"`
	CustomsFormEnabled       bool                 `json:"customsFormEnabled"`
	CustomsSigner            string               `json:"customsSigner"`
	CustomsContentType       string               `json:"customsContentType"`
	CustomsItems             []CustomsItem        `json:"customsItems,omitempty"`
	ExporterTaxID            string               `json:"exporterTaxId"`
	RecipientTaxID           string               `json:"recipientTaxId"`
}

// ShipmentPreset contains package, service, insurance, and customs settings.
type ShipmentPreset struct {
	ID                       ID                   `json:"id,omitempty"`
	OriginalID               ID                   `json:"originalId,omitempty"`
	Title                    string               `json:"title"`
	PackageTypeID            ID                   `json:"packageTypeId,omitempty"`
	PackageTypeKey           string               `json:"packageTypeKey"`
	Weight                   float64              `json:"weight"`
	DimensionX               float64              `json:"dimensionX"`
	DimensionY               float64              `json:"dimensionY"`
	DimensionZ               float64              `json:"dimensionZ"`
	InsuredValueFlag         bool                 `json:"insuredValueFlag"`
	InsuredValue             float64              `json:"insuredValue"`
	DeliveryConfirmationFlag bool                 `json:"deliveryConfirmationFlag"`
	DeliveryConfirmation     DeliveryConfirmation `json:"deliveryConfirmation"`
	ReturnLabelFlag          bool                 `json:"returnLabelFlag"`
	ReturnLabel              ReturnLabel          `json:"returnLabel"`
	QualifiesAsMediaMail     bool                 `json:"qualifiesAsMediaMail"`
	IrregularPackage         bool                 `json:"irregularPackage"`
	HazardousMaterials       bool                 `json:"hazardousMaterialsEnabled"`
	CustomsFormEnabled       bool                 `json:"customsFormEnabled"`
	CustomsSigner            string               `json:"customsSigner"`
	CustomsContentType       string               `json:"customsContentType"`
	CustomsItems             []CustomsItem        `json:"customsItems,omitempty"`
	ExporterTaxID            string               `json:"exporterTaxId"`
	RecipientTaxID           string               `json:"recipientTaxId"`
}

// Warehouse pairs an origin address with a return address.
type Warehouse struct {
	ID                       ID      `json:"id"`
	Title                    string  `json:"title"`
	Shy                      bool    `json:"shy"`
	UseOriginAsReturnAddress bool    `json:"useOriginAsReturnAddress"`
	OriginAddress            Address `json:"originAddress"`
	ReturnAddress            Address `json:"returnAddress"`
}

// ShipmentStatus is a purchased shipment's lifecycle state.
type ShipmentStatus string

// Shipment is the common subset returned by lifecycle operations.
type Shipment struct {
	ID                       ID             `json:"id"`
	CarrierKey               CarrierKey     `json:"carrierKey,omitempty"`
	Status                   ShipmentStatus `json:"status,omitempty"`
	HasBeenDownloaded        bool           `json:"hasBeenDownloaded,omitempty"`
	CanPrint                 bool           `json:"canPrint,omitempty"`
	CanShowMobileCode        bool           `json:"canShowMobileCode,omitempty"`
	IsShipmentRefundable     bool           `json:"isShipmentRefundable,omitempty"`
	CanInstantRefundShipment bool           `json:"canInstantRefundShipment,omitempty"`
	AdditionalRefundNotice   string         `json:"additionalRefundNotice,omitempty"`
}

// ShipmentStatusSummary contains batch-level shipment action counts.
type ShipmentStatusSummary struct {
	RefundableCount int `json:"refundableCount"`
	PrintableCount  int `json:"printableCount"`
}

// Batch is the common subset returned by shipment lifecycle operations.
type Batch struct {
	ID                    ID                    `json:"id"`
	Title                 string                `json:"title,omitempty"`
	Status                BatchStatus           `json:"status,omitempty"`
	Step                  string                `json:"step,omitempty"`
	StepText              string                `json:"stepText,omitempty"`
	ShipDate              string                `json:"shipDate,omitempty"`
	ShipDatePossibleNow   bool                  `json:"shipDatePossibleNow,omitempty"`
	NumShipments          int                   `json:"numShipments,omitempty"`
	CanInstantRefundBatch bool                  `json:"canInstantRefundBatch,omitempty"`
	CanRefund             bool                  `json:"canRefund,omitempty"`
	ShipmentStatusSummary ShipmentStatusSummary `json:"shipmentStatusSummary,omitempty"`
	Shipments             []Shipment            `json:"shipments,omitempty"`
	RunningProcess        *RunningProcess       `json:"runningProcess,omitempty"`
}

// RunningProcess reports progress for an asynchronous batch job.
type RunningProcess struct {
	Status               string   `json:"status"`
	ItemsInProgressCount int      `json:"itemsInProgressCount"`
	ItemsTotalCount      int      `json:"itemsTotalCount"`
	ProcessKey           string   `json:"processKey"`
	ProgressPercentage   float64  `json:"progressPercentage"`
	ProgressTitle        string   `json:"progressTitle"`
	SecondsLeft          *float64 `json:"secondsLeft"`
}
