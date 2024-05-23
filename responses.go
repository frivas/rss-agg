package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func JsonResponseError(w http.ResponseWriter, statusCode int, message string) {
	if statusCode > 499 {
		log.Printf("Responding with 5XX error: %s", message)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}

	JsonResponse(w, statusCode, errorResponse{
		message,
	})
}

func JsonResponse(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	_, errW := w.Write(dat)
	if errW != nil {
		return
	}
}

func JsonResponseNoPayload(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, errW := w.Write([]byte(""))
	if errW != nil {
		return
	}
}
