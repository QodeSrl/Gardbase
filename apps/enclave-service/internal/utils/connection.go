package utils

import (
	"encoding/json"
	"log"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
)

func SendResponse[T any](encoder *json.Encoder, res enclaveproto.Response[T]) {
	// encode the response as JSON and send it
	if err := encoder.Encode(res); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func SendError(encoder *json.Encoder, errMsg string) {
	response := enclaveproto.Response[any]{
		Success: false,
		Error:   errMsg,
	}
	log.Printf("Sending error response: %s", errMsg)
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}
