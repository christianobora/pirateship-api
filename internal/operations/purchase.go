package operations

const (
	// PurchaseInformation returns rated services and billing choices.
	PurchaseInformation = `
query PurchaseInformationQuery($id: ID!) {
  batch(id: $id) {
    id title status step stepText shipDate shipDatePossibleNow numShipments
    eligibleForInsurance emailNotificationPossible labelSize labelFileFormat labelingArtifactType
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
    rateGroups {
      groupKey { string traits { layer value } }
      affectedByUpsRateLimit maximumShipments defaultShipDate
      rateSummaries {
        uniqueId maxWeightOz mailClassTitle serviceTitle deliveryDays totalPrice basePrice averageBasePrice
        flatPrice crossedTotalPrice averageTotalPrice cheapest best fastest savings shipmentCount
        firstZone ratePeriodStartDate ratePeriodEndDate errorMessage valueLimit
        carrier { carrierKey title }
        firstMailClass { mailClassKey }
        packageType { packageTypeKey }
        availableShipDates
        surcharges { surchargeKey title price helpLink crossedPrice }
      }
    }
  }
  company {
    id accountBalance hasAnyPlaidPaymentSource isFirstLabel activeCarriers
    settings {
      defaultPaymentSourceId defaultChargeAmount defaultTrackingEmailsEnabled
      defaultTrackingEmailsDelay defaultEmailTemplateId
    }
    paymentSources {
      id paymentMethodType validationStatus brand last4 expMonth expYear
      resultMessage email title nickname refundableAmount hasConsented
    }
    merchantAccounts { id carrierKey }
    mailTemplates { id subject name senderEmail senderName asDefault }
  }
}`

	// RerateBatch starts re-rating an existing batch.
	RerateBatch = `
mutation RerateBatchMutation($id: ID!, $shipDate: DateTime) {
  rerateBatch(id: $id, shipDate: $shipDate) {
    id step shipDate shipDatePossibleNow
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
  }
}`

	// ModifyInsuranceOnRatedBatch changes insurance and returns refreshed rates.
	ModifyInsuranceOnRatedBatch = `
mutation ModifyInsuranceOnRatedBatchMutation($id: ID!, $insuredValue: Float!) {
  modifyInsuranceOnRatedBatch(id: $id, insuredValue: $insuredValue) {
    id title status stepText
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
    rateGroups {
      groupKey { string traits { layer value } }
      affectedByUpsRateLimit maximumShipments
      rateSummaries {
        uniqueId mailClassTitle serviceTitle totalPrice basePrice cheapest best fastest errorMessage
        carrier { carrierKey title }
        firstMailClass { mailClassKey }
        packageType { packageTypeKey }
      }
    }
  }
}`

	// BuyBatch purchases labels. The server may charge the selected payment source.
	BuyBatch = `
mutation BuyBatchMutation(
  $id: ID!
  $rateSelection: [RateSelectionInput!]!
  $paymentSourceId: ID!
  $shipDate: DateTime!
  $totalCharge: Float!
  $mailTemplateId: ID
  $notifyRecipientsDate: String
) {
  buyBatch(
    id: $id
    rateSelection: $rateSelection
    paymentSourceId: $paymentSourceId
    shipDate: $shipDate
    totalCharge: $totalCharge
    mailTemplateId: $mailTemplateId
    notifyRecipientsDate: $notifyRecipientsDate
  ) {
    id step shipDate numShipments
    shipments { id }
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
  }
}`
)
