package response

import "net/http"

type APIResponse struct {
	Headers http.Header
	Body    []byte
}
