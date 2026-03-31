package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type MLRequest struct {
	FileName   string `json:"file_name"`
	MagicBytes string `json:"magic_bytes"`
}

type MLResponse struct {
	Prediction string `json:"prediction"`
}

func PredictWorkload(fileName string, magicBytes []byte) string {
	hexSignature := hex.EncodeToString(magicBytes)

	reqData := MLRequest{
		FileName:   fileName,
		MagicBytes: hexSignature,
	}
	jsonData, _ := json.Marshal(reqData)

	client := &http.Client{Timeout: 200 * time.Millisecond}

	resp, err := client.Post("http://localhost:8000/predict", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("ML Service unavailable. Defaulting to standard routing.")
		return "Unknown"
	}
	defer resp.Body.Close()

	var mlResp MLResponse
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &mlResp)

	return mlResp.Prediction
}
