package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Option func(*Client)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiKey     string
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("API returned status %d: %s", e.StatusCode, e.Message)
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = strings.TrimSpace(apiKey)
	}
}

func NewClient(baseURL string, options ...Option) (*Client, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("base URL must include scheme and host")
	}

	client := &Client{
		baseURL: parsedURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = strings.TrimSpace(apiKey)
}

func (c *Client) CreateJob(ctx context.Context, request CreateJobRequest) (Job, error) {
	var response struct {
		Job Job `json:"job"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/jobs", request, true, http.StatusCreated, &response); err != nil {
		return Job{}, err
	}
	return response.Job, nil
}

func (c *Client) ListJobs(ctx context.Context) ([]Job, error) {
	var response struct {
		Jobs []Job `json:"jobs"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/jobs", nil, true, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return response.Jobs, nil
}

func (c *Client) GetJob(ctx context.Context, jobID string) (Job, error) {
	var response struct {
		Job Job `json:"job"`
	}
	if err := c.do(ctx, http.MethodGet, path.Join("/api/v1/jobs", jobID), nil, true, http.StatusOK, &response); err != nil {
		return Job{}, err
	}
	return response.Job, nil
}

func (c *Client) UpdateJob(ctx context.Context, jobID string, request UpdateJobRequest) (Job, error) {
	var response struct {
		Job Job `json:"job"`
	}
	if err := c.do(ctx, http.MethodPatch, path.Join("/api/v1/jobs", jobID), request, true, http.StatusOK, &response); err != nil {
		return Job{}, err
	}
	return response.Job, nil
}

func (c *Client) DeleteJob(ctx context.Context, jobID string) error {
	return c.do(ctx, http.MethodDelete, path.Join("/api/v1/jobs", jobID), nil, true, http.StatusOK, nil)
}

func (c *Client) PauseJob(ctx context.Context, jobID string) (Job, error) {
	var response struct {
		Job Job `json:"job"`
	}
	if err := c.do(ctx, http.MethodPost, path.Join("/api/v1/jobs", jobID, "pause"), nil, true, http.StatusOK, &response); err != nil {
		return Job{}, err
	}
	return response.Job, nil
}

func (c *Client) ResumeJob(ctx context.Context, jobID string) (Job, error) {
	var response struct {
		Job Job `json:"job"`
	}
	if err := c.do(ctx, http.MethodPost, path.Join("/api/v1/jobs", jobID, "resume"), nil, true, http.StatusOK, &response); err != nil {
		return Job{}, err
	}
	return response.Job, nil
}

func (c *Client) TriggerJob(ctx context.Context, jobID string) error {
	return c.do(ctx, http.MethodPost, path.Join("/api/v1/jobs", jobID, "trigger"), nil, true, http.StatusAccepted, nil)
}

func (c *Client) ListJobRuns(ctx context.Context, jobID string) ([]Run, error) {
	var response struct {
		Runs []Run `json:"runs"`
	}
	if err := c.do(ctx, http.MethodGet, path.Join("/api/v1/jobs", jobID, "runs"), nil, true, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (Run, error) {
	var response struct {
		Run Run `json:"run"`
	}
	if err := c.do(ctx, http.MethodGet, path.Join("/api/v1/runs", runID), nil, true, http.StatusOK, &response); err != nil {
		return Run{}, err
	}
	return response.Run, nil
}

func (c *Client) CreateAPIKey(ctx context.Context, request CreateAPIKeyRequest) (CreateAPIKeyResponse, error) {
	var response CreateAPIKeyResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/api-keys", request, true, http.StatusCreated, &response); err != nil {
		return CreateAPIKeyResponse{}, err
	}
	return response, nil
}

func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var response struct {
		APIKeys []APIKey `json:"api_keys"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/api-keys", nil, true, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return response.APIKeys, nil
}

func (c *Client) RevokeAPIKey(ctx context.Context, keyID string) (APIKey, error) {
	var response struct {
		APIKey APIKey `json:"api_key"`
	}
	if err := c.do(ctx, http.MethodPost, path.Join("/api/v1/api-keys", keyID, "revoke"), nil, true, http.StatusOK, &response); err != nil {
		return APIKey{}, err
	}
	return response.APIKey, nil
}

func (c *Client) do(ctx context.Context, method string, resourcePath string, requestBody any, requiresAuth bool, expectedStatus int, output any) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, resourcePath)

	httpRequest, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	if requestBody != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	if requiresAuth {
		if strings.TrimSpace(c.apiKey) == "" {
			return fmt.Errorf("API key is required for this endpoint")
		}
		httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if httpResponse.StatusCode != expectedStatus {
		return parseAPIError(httpResponse.StatusCode, responseBody)
	}

	if output == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func parseAPIError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}

	var structured struct {
		Message string `json:"message"`
		Error   struct {
			ErrorMessage string `json:"error_message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &structured); err == nil {
		if structured.Error.ErrorMessage != "" {
			message = structured.Error.ErrorMessage
		} else if structured.Message != "" {
			message = structured.Message
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}
