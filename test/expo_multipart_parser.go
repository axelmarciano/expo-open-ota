package test

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"strings"
)

// This a reimplementation of the @expo/multipart-body-parser[https://www.npmjs.com/package/@expo/multipart-body-parser] in golang to test manifest response

type MultipartPart struct {
	Body        string
	Headers     map[string]string
	Name        string
	Disposition string
	Parameters  map[string]string
}

func ParseMultipartMixedResponse(contentTypeHeader string, bodyBuffer []byte) ([]MultipartPart, error) {
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return nil, err
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, err
	}

	reader := multipart.NewReader(bytes.NewReader(bodyBuffer), boundary)
	var parts []MultipartPart

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}

		body, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		for key, values := range part.Header {
			headers[key] = strings.Join(values, ", ")
		}

		disposition, params, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))

		parts = append(parts, MultipartPart{
			Body:        string(body),
			Headers:     headers,
			Name:        params["name"],
			Disposition: disposition,
			Parameters:  params,
		})
	}

	return parts, nil
}

func IsMultipartPartWithName(part MultipartPart, name string) bool {
	return part.Name == name
}
