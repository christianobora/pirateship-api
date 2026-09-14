package operations

const (
	// ShippingConfiguration returns the metadata needed to construct shipments.
	ShippingConfiguration = `
query ShippingConfigurationQuery {
  company {
    id
    activeCarriers
    settings { defaultWarehouseId defaultShipmentPresetId shipmentLabelSize }
    warehouses {
      id title useOriginAsReturnAddress shy
      originAddress { id fullName company address1 address2 city regionCode postcode zip4 countryCode phone }
      returnAddress { id fullName company address1 address2 city regionCode postcode zip4 countryCode }
    }
    shipmentPresets {
      id originalId title packageTypeId packageTypeKey weight dimensionX dimensionY dimensionZ
      insuredValueFlag insuredValue deliveryConfirmationFlag deliveryConfirmation
      returnLabelFlag returnLabel qualifiesAsMediaMail irregularPackage hazardousMaterialsEnabled
      customsFormEnabled customsSigner customsContentType exporterTaxId recipientTaxId
      customsItems { id title quantity itemValue weight hsTariffNumber countryCodeOfOrigin }
    }
  }
  shipmentBoundaries {
    maxWeight maxCombinedLength maxLengthPlusGirth
    minLongSide maxLongSide minMiddleSide maxMiddleSide minShortSide maxShortSide
  }
  carriers {
    id carrierKey
    packageTypes { id packageTypeKey title description heightRequired weightRequired dimensionsRequired }
  }
  countries {
    id name countryCode: isoAlpha2Code countryCode3: isoAlpha3Code postcodeRequired
  }
}`

	// CreateWarehouse creates a reusable origin/return-address pair.
	CreateWarehouse = `
mutation CreateWarehouseMutation(
  $title: String!
  $saveAddressForReuse: Boolean!
  $useOriginAsReturnAddress: Boolean!
  $originAddress: AddressInput!
  $returnAddress: AddressInput
) {
  createWarehouse(
    title: $title
    saveAddressForReuse: $saveAddressForReuse
    useOriginAsReturnAddress: $useOriginAsReturnAddress
    originAddress: $originAddress
    returnAddress: $returnAddress
  ) {
    id title useOriginAsReturnAddress shy
    originAddress { id fullName company address1 address2 city regionCode postcode zip4 countryCode phone }
    returnAddress { id fullName company address1 address2 city regionCode postcode zip4 countryCode }
  }
}`

	// CreateShipmentPreset creates reusable package/customs settings.
	CreateShipmentPreset = `
mutation CreateShipmentPresetMutation($default: Boolean!, $shipmentPreset: ShipmentPresetInput!) {
  createShipmentPreset(default: $default, shipmentPreset: $shipmentPreset) {
    id originalId title packageTypeId packageTypeKey weight dimensionX dimensionY dimensionZ
    insuredValueFlag insuredValue deliveryConfirmationFlag deliveryConfirmation
    returnLabelFlag returnLabel qualifiesAsMediaMail irregularPackage hazardousMaterialsEnabled
    customsFormEnabled customsSigner customsContentType exporterTaxId recipientTaxId
    customsItems { id title quantity itemValue weight hsTariffNumber countryCodeOfOrigin }
  }
}`

	// UpdateShipmentPreset updates reusable package/customs settings.
	UpdateShipmentPreset = `
mutation UpdateShipmentPresetMutation($default: Boolean!, $shipmentPreset: ShipmentPresetInput!) {
  updateShipmentPreset(default: $default, shipmentPreset: $shipmentPreset) {
    id originalId title packageTypeId packageTypeKey weight dimensionX dimensionY dimensionZ
    insuredValueFlag insuredValue deliveryConfirmationFlag deliveryConfirmation
    returnLabelFlag returnLabel qualifiesAsMediaMail irregularPackage hazardousMaterialsEnabled
    customsFormEnabled customsSigner customsContentType exporterTaxId recipientTaxId
    customsItems { id title quantity itemValue weight hsTariffNumber countryCodeOfOrigin }
  }
}`

	// CreateBatchFromSingleShipment creates and starts rating one shipment.
	CreateBatchFromSingleShipment = `
mutation CreateBatchFromSingleShipmentMutation(
  $warehouseId: ID!
  $shipmentPresetId: ID!
  $shipToAddress: RecipientAddressInput!
  $rubberStamps: RubberStampsInput!
  $validatedAddressId: ID
  $skipCorrection: Boolean!
) {
  createBatchFromSingleShipment(
    warehouseId: $warehouseId
    shipmentPresetId: $shipmentPresetId
    shipToAddress: $shipToAddress
    rubberStamps: $rubberStamps
    validatedAddressId: $validatedAddressId
    skipCorrection: $skipCorrection
  ) { id step status }
}`

	// UpdateBatchTitle changes a batch display title.
	UpdateBatchTitle = `
mutation UpdateBatchTitleMutation($id: ID!, $title: String!) {
  updateBatchTitle(id: $id, title: $title) { id title }
}`

	// DeleteBatch removes an unpurchased batch.
	DeleteBatch = `
mutation DeleteBatchMutation($id: ID!) {
  deleteBatch(id: $id) { id }
}`
)
