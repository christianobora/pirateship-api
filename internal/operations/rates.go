package operations

// Rates is the public rate-estimate operation.
const Rates = `
query RatesQuery(
  $originZip: String!
  $originCity: String
  $originRegionCode: String
  $destinationZip: String
  $isResidential: Boolean
  $destinationCountryCode: String
  $weight: Float
  $dimensionX: Float
  $dimensionY: Float
  $dimensionZ: Float
  $mailClassKeys: [String!]!
  $packageTypeKeys: [String!]!
  $pricingTypes: [String!]
  $showUpsRatesWhen2x7Selected: Boolean
) {
  rates(
    originZip: $originZip
    originCity: $originCity
    originRegionCode: $originRegionCode
    destinationZip: $destinationZip
    isResidential: $isResidential
    destinationCountryCode: $destinationCountryCode
    weight: $weight
    dimensionX: $dimensionX
    dimensionY: $dimensionY
    dimensionZ: $dimensionZ
    mailClassKeys: $mailClassKeys
    packageTypeKeys: $packageTypeKeys
    pricingTypes: $pricingTypes
    showUpsRatesWhen2x7Selected: $showUpsRatesWhen2x7Selected
  ) {
    title
    deliveryDescription
    trackingDescription
    serviceDescription
    pricingDescription
    cubicTier
    mailClassKey
    mailClass { accuracy international }
    packageTypeKey
    zone
    surcharges { title price }
    carrier { carrierKey title }
    totalPrice
    priceBaseTypeKey
    basePrice
    crossedTotalPrice
    pricingType
    pricingSubType
    ratePeriodId
    learnMoreUrl
    cheapest
    fastest
    best
  }
}`
