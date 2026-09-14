package pirateship

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/christianobora/pirateship-api/internal/operations"
	"github.com/christianobora/pirateship-api/internal/poll"
)

// PageLayout is a supported label sheet or thermal-printer layout.
type PageLayout string

// Supported label page layouts.
const (
	PageLayout2x7       PageLayout = "LAYOUT_2x7"
	PageLayout4x6       PageLayout = "LAYOUT_4x6"
	PageLayoutLetter1L  PageLayout = "LAYOUT_85x11_1UP_LEFT"
	PageLayoutLetter1R  PageLayout = "LAYOUT_85x11_1UP_RIGHT"
	PageLayoutLetter2Up PageLayout = "LAYOUT_85x11_2UP"
)

// LabelFileFormat identifies rendered label data.
type LabelFileFormat string

// Supported label file formats.
const (
	LabelFileFormatPDF LabelFileFormat = "PDF"
	LabelFileFormatPNG LabelFileFormat = "PNG"
	LabelFileFormatZPL LabelFileFormat = "ZPL"
)

// LabelStatus is the state of an asynchronous label-rendering job.
type LabelStatus string

// Observed label-rendering states.
const (
	LabelStatusPending  LabelStatus = "PENDING"
	LabelStatusRunning  LabelStatus = "RUNNING"
	LabelStatusFinished LabelStatus = "FINISHED"
	LabelStatusError    LabelStatus = "ERROR"
)

// Label describes one rendered label artifact.
type Label struct {
	ID         ID              `json:"id"`
	Status     LabelStatus     `json:"status"`
	FileFormat LabelFileFormat `json:"fileFormat"`
	PageLayout PageLayout      `json:"pageLayout"`
	URL        string          `json:"url"`
}

// TriggerLabelsRequest starts label rendering for purchased shipments.
type TriggerLabelsRequest struct {
	ShipmentIDs []ID        `json:"shipmentIds"`
	PageLayout  *PageLayout `json:"pageLayout,omitempty"`
	ViaAdminBar *bool       `json:"isViaAdminBar,omitempty"`
}

// LabelsRequest identifies a label-rendering job.
type LabelsRequest struct {
	DownloadID  ID      `json:"downloadId"`
	ShareToken  *string `json:"shareToken,omitempty"`
	ViaAdminBar *bool   `json:"isViaAdminBar,omitempty"`
}

// TriggerLabels starts label rendering and returns its download job ID.
func (c *Client) TriggerLabels(
	ctx context.Context,
	request TriggerLabelsRequest,
) (ID, *Response, error) {
	if len(request.ShipmentIDs) == 0 {
		return "", nil, &ValidationError{Field: "shipment IDs", Problem: "must not be empty"}
	}
	for _, id := range request.ShipmentIDs {
		if id == "" {
			return "", nil, &ValidationError{Field: "shipment ID", Problem: "must not be empty"}
		}
	}
	var data struct {
		DownloadID ID `json:"triggerLabelCreationByShipments"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "TriggerDownloadLabelsByShipmentsMutation",
		Query:     operations.TriggerLabels,
		Variables: request,
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return "", response, fmt.Errorf("trigger labels: %w", err)
	}
	return data.DownloadID, response, nil
}

// Labels returns current label-rendering results.
func (c *Client) Labels(
	ctx context.Context,
	request LabelsRequest,
) ([]Label, *Response, error) {
	if request.DownloadID == "" {
		return nil, nil, &ValidationError{Field: "download ID", Problem: "is required"}
	}
	var data struct {
		Labels []Label `json:"labels"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "LabelsQuery",
		Query:     operations.Labels,
		Variables: request,
		Type:      OperationQuery,
	}, &data)
	if err != nil {
		return data.Labels, response, fmt.Errorf("get labels: %w", err)
	}
	return data.Labels, response, nil
}

// WaitForLabels polls until every label is finished or any label errors.
func (c *Client) WaitForLabels(
	ctx context.Context,
	request LabelsRequest,
	interval time.Duration,
) ([]Label, error) {
	if ctx == nil {
		return nil, &ValidationError{Field: "context", Problem: "must not be nil"}
	}
	if interval == 0 {
		interval = defaultPollInterval
	}
	if interval < 0 {
		return nil, &ValidationError{Field: "poll interval", Problem: "must not be negative"}
	}

	labels, err := poll.Until(ctx, interval, func(ctx context.Context) ([]Label, bool, error) {
		current, _, err := c.Labels(ctx, request)
		if err != nil {
			return nil, false, err
		}
		if len(current) == 0 {
			return current, false, nil
		}
		finished := true
		for _, label := range current {
			switch label.Status {
			case LabelStatusError:
				return nil, false, &LabelCreationError{DownloadID: request.DownloadID}
			case LabelStatusFinished:
			default:
				finished = false
			}
		}
		return current, finished, nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait for labels %q: %w", request.DownloadID, err)
	}
	return labels, nil
}

// DownloadLabel streams a finished label to destination. Custom GraphQL
// headers are not forwarded to the artifact host.
func (c *Client) DownloadLabel(
	ctx context.Context,
	label Label,
	destination io.Writer,
) (*Response, error) {
	if ctx == nil {
		return nil, &ValidationError{Field: "context", Problem: "must not be nil"}
	}
	if isNilInterface(destination) {
		return nil, &ValidationError{Field: "label destination", Problem: "must not be nil"}
	}
	if label.Status != LabelStatusFinished {
		return nil, &ValidationError{Field: "label status", Problem: "must be FINISHED"}
	}
	artifactURL, err := labelDownloadURL(label.URL)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build label download request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download label: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	metadata := &Response{StatusCode: response.StatusCode, Header: response.Header.Clone()}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, truncated, readErr := readLimited(response.Body, c.maxResponseBytes)
		if readErr != nil {
			return metadata, fmt.Errorf("read label download error: %w", readErr)
		}
		return metadata, &HTTPError{
			StatusCode: response.StatusCode, Status: response.Status,
			Header: response.Header.Clone(), Body: body, Truncated: truncated,
		}
	}
	if _, err := io.Copy(destination, response.Body); err != nil {
		return metadata, fmt.Errorf("stream label: %w", err)
	}
	return metadata, nil
}

// DownloadMobileCode returns the data URI for a supported purchased shipment.
func (c *Client) DownloadMobileCode(
	ctx context.Context,
	shipmentID ID,
) (string, *Response, error) {
	if shipmentID == "" {
		return "", nil, &ValidationError{Field: "shipment ID", Problem: "is required"}
	}
	var data struct {
		MobileCode struct {
			DataURI string `json:"mobileCodeDataUri"`
		} `json:"downloadMobileCodeForShipment"`
	}
	response, err := c.Do(ctx, Operation{
		Name:      "DownloadMobileCodeForShipment",
		Query:     operations.DownloadMobileCode,
		Variables: map[string]ID{"id": shipmentID},
		Type:      OperationMutation,
	}, &data)
	if err != nil {
		return "", response, fmt.Errorf("download mobile code: %w", err)
	}
	return data.MobileCode.DataURI, response, nil
}

func labelDownloadURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", &ValidationError{Field: "label URL", Problem: "is invalid"}
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", &ValidationError{Field: "label URL", Problem: "must be an HTTPS URL without user information"}
	}
	parsed.Path = strings.Replace(parsed.Path, "/force/0", "/force/1", 1)
	return parsed.String(), nil
}
