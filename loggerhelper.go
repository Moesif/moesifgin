package moesifgin

import (
	b64 "encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func contains(arr []string, str string) bool {
	for _, value := range arr {
		if value == str {
			return true
		}
	}
	return false
}

func getConfigStringValuesForIncomingEvent(fieldName string, c *gin.Context) string {
	if callback, found := moesifOption[fieldName]; found {
		return callback.(func(c *gin.Context) string)(c)
	}
	return ""
}

func getConfigStringValuesForOutgoingEvent(fieldName string, request *http.Request, response *http.Response) string {
	if callback, found := moesifOption[fieldName]; found {
		return callback.(func(*http.Request, *http.Response) string)(request, response)
	}
	return ""
}

func HeaderToMap(header http.Header) map[string]interface{} {
	headerMap := make(map[string]interface{})
	for name, values := range header {
		headerMap[name] = values
	}
	return headerMap
}

func maskHeaders(headers map[string]interface{}, fieldName string) map[string]interface{} {
	var maskFields []string
	if _, found := moesifOption[fieldName]; found {
		maskFields = moesifOption[fieldName].(func() []string)()
		headers = maskData(headers, maskFields)
	}
	return headers
}

func maskData(data map[string]interface{}, maskBody []string) map[string]interface{} {
	for key, val := range data {
		switch val.(type) {
		case map[string]interface{}:
			if contains(maskBody, key) {
				data[key] = "*****"
			} else {
				maskData(val.(map[string]interface{}), maskBody)
			}
		default:
			if contains(maskBody, key) {
				data[key] = "*****"
			}
		}
	}
	return data
}

func parseBody(readReqBody []byte, fieldName string) (interface{}, string) {
	var body interface{}
	bodyEncoding := "json"
	if jsonMarshalErr := json.Unmarshal(readReqBody, &body); jsonMarshalErr != nil {
		if debug {
			log.Printf("About to parse body as base64 ")
		}
		body = b64.StdEncoding.EncodeToString(readReqBody)
		bodyEncoding = "base64"
		if debug {
			log.Printf("Parsed body as base64 - %s", body)
		}
	} else {
		// If the body is a JSON object, optionally mask selected fields from logging
		var maskFields []string
		if _, found := moesifOption[fieldName]; found {
			maskFields = moesifOption[fieldName].(func() []string)()
			if mappedBody, ok := body.(map[string]interface{}); ok {
				body = maskData(mappedBody, maskFields)
			} else {
				log.Printf("Expected body to be a map but got: %T", body)
			}
		}
	}
	return body, bodyEncoding
}

// getContentLength tries to parse the Content-Length header to an int64.
// If parsing fails or the header is not present, it uses the length of the provided body slice.
// Returns a pointer to the determined content length.
func getContentLength(headers http.Header, body []byte) (contentLength *int64) {
	if contentLengthStr := headers.Get("Content-Length"); contentLengthStr != "" {
		parsedLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			if debug {
				log.Printf("Error while parsing content-length: %s.\n", err)
			}
		} else {
			contentLength = &parsedLength
		}
	}
	if contentLength == nil {
		length := int64(len(body))
		contentLength = &length
	}
	return contentLength
}
