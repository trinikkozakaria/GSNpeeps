package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var errInvalidJSON = errors.New("invalid JSON body")

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalidJSON
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errInvalidJSON
	}
	return nil
}
