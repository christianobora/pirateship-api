// Package operations contains the module's audited GraphQL documents.
package operations

const (
	// BatchProcessStatus returns the state of a background batch operation.
	BatchProcessStatus = `
query BatchProcessStatusQuery($id: ID!) {
  batch(id: $id) {
    id status step canInstantRefundBatch numShipments
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
    shipments { id carrierKey }
  }
}`

	// RefundBatch requests a refund for all eligible labels in a batch.
	RefundBatch = `
mutation RefundBatchMutation($id: ID!) {
  refundBatch(id: $id) {
    id step
    runningProcess {
      status itemsInProgressCount itemsTotalCount processKey
      progressPercentage progressTitle secondsLeft
    }
  }
}`

	// RefundShipment requests a refund for one eligible label.
	RefundShipment = `
mutation RefundShipmentMutation($shipmentId: ID!) {
  refundShipment(shipmentId: $shipmentId) {
    id canRefund canInstantRefundBatch
    shipmentStatusSummary { refundableCount printableCount }
    shipments {
      id status hasBeenDownloaded canPrint carrierKey canShowMobileCode
      isShipmentRefundable canInstantRefundShipment additionalRefundNotice
    }
  }
}`

	// TriggerLabels starts label rendering for shipments.
	TriggerLabels = `
mutation TriggerDownloadLabelsByShipmentsMutation(
  $shipmentIds: [ID!]!
  $pageLayout: PageLayout
  $isViaAdminBar: Boolean
) {
  triggerLabelCreationByShipments(
    shipmentIds: $shipmentIds
    pageLayout: $pageLayout
    isViaAdminBar: $isViaAdminBar
  )
}`

	// Labels returns label-rendering status and output URL.
	Labels = `
query LabelsQuery($downloadId: ID!, $shareToken: String, $isViaAdminBar: Boolean) {
  labels(downloadId: $downloadId, shareToken: $shareToken, isViaAdminBar: $isViaAdminBar) {
    id status fileFormat pageLayout url
  }
}`

	// DownloadMobileCode returns a carrier mobile-code data URI.
	DownloadMobileCode = `
mutation DownloadMobileCodeForShipment($id: ID!) {
  downloadMobileCodeForShipment(id: $id) { mobileCodeDataUri }
}`
)
