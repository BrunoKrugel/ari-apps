package client

import (
	"fmt"
	"net/http"
	"net/url"

	resty "github.com/go-resty/resty/v2"
	response "lineblocs.com/processor/api/model"
)

type HttpRequest struct {
	client *resty.Client
}

const (
	CONTENT_TYPE = "Content-Type"
	VALUE        = "application/json"
)

// TODO: Receive client as parameter in send methods
func NewRestClient() *HttpRequest {
	return &HttpRequest{
		client: resty.New(),
	}
}

func (req *HttpRequest) Get(url string) (*resty.Response, error) {
	return req.client.R().Get(url)
}

func (req *HttpRequest) SendGetRequest(baseURL string, header string, path string, vals map[string]string) (string, error) {
	fullURL := baseURL + path

	request := req.client.R().
		SetHeader(CONTENT_TYPE, VALUE)

	query := url.Values{}
	for k, v := range vals {
		query.Add(k, v)
	}

	request.SetQueryString(query.Encode())

	resp, err := request.Get(fullURL)
	if err != nil {
		return "", err
	}

	body := resp.String()

	status := resp.StatusCode()
	if !(status >= 200 && status <= 299) {
		return "", fmt.Errorf("status: %s result: %s", resp.Status(), body)
	}

	return body, nil
}

func (req *HttpRequest) SendPostHttpRequest(baseURL string, header string, path string, payload []byte) (*response.APIResponse, error) {
	url := baseURL + path

	client := resty.New()
	resp, err := client.R().
		SetHeader("X-Custom-Header", header).
		SetHeader(CONTENT_TYPE, VALUE).
		SetBody(payload).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %v", err)
	}

	status := resp.StatusCode()
	body := resp.Body()

	if status < http.StatusOK || status > http.StatusPartialContent {
		return nil, fmt.Errorf("HTTP request failed with status %s: %s", resp.Status(), body)
	}

	headers := resp.Header()

	return &response.APIResponse{
		Headers: headers,
		Body:    []byte(body),
	}, nil
}

func (req *HttpRequest) SendPutRequest(baseURL string, header string, path string, payload []byte) (string, error) {
	url := baseURL + path

	client := resty.New()
	resp, err := client.R().
		SetHeader(CONTENT_TYPE, VALUE).
		SetBody(payload).
		Put(url)

	if err != nil {
		return "", err
	}

	body := resp.String()

	status := resp.StatusCode()
	if status < 200 || status > 299 {
		return "", fmt.Errorf("status: %s result: %s", resp.Status(), body)
	}

	return body, nil
}
