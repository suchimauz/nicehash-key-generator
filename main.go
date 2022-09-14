package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type NowTimeResponse struct {
	ServerTime int64 `json:"serverTime"`
}
type NiceHashRequest struct {
	Action string `json:"action"`
	RigId  string `json:"rigId"`
}
type NiceHashResponse struct {
	Success     bool   `json:"success"`
	SuccessType string `json:"successType"`
}

func main() {
	var action string
	var apiKey string
	var apiSecret string
	var rigId string
	var xOrganizationId string
	var nowTimeResponse NowTimeResponse
	var niceHashResponse NiceHashResponse

	flag.StringVar(&action, "action", "START", "Action to be taken: START, STOP, POWER_MODE")
	flag.StringVar(&apiKey, "api-key", "", "API KEY from nicehash.com")
	flag.StringVar(&apiSecret, "api-secret", "", "API Secret from nicehash.com")
	flag.StringVar(&xOrganizationId, "org-id", "", "Organization id from nicehash.com")
	flag.StringVar(&rigId, "rig-id", "", "Rig id to do action from nicehash.com")
	flag.Parse()

	if action != "START" && action != "STOP" && action != "POWER_MODE" {
		log.Fatalf("Action must be only one of: START, STOP, POWER_MODE")
	}
	if apiKey == "" {
		log.Fatalf("API KEY is required. Use -api-key flag, Get keys from https://www.nicehash.com/my/settings/keys")
	}
	if apiSecret == "" {
		log.Fatalf("API Secret is required. Use -api-key flag. Get keys from https://www.nicehash.com/my/settings/keys")
	}
	if xOrganizationId == "" {
		log.Fatalf("Organization id is required. Use -org-id flag. Get organization id from https://www.nicehash.com/my/settings/keys")
	}
	if rigId == "" {
		log.Fatalf("Rig id is required. Use -org-id flag. Get rig id from https://www.nicehash.com/my/mining/rigs/ of your rig")
	}

	niceHashRequest := NiceHashRequest{
		Action: action,
		RigId:  rigId,
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api2.nicehash.com/api/v2/time", nil)
	if err != nil {
		log.Fatalf("[HTTP] %s", err.Error())
		return
	}
	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[HTTP] %s", err.Error())
		return
	}
	// Read Response Body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return
	}

	err = json.Unmarshal([]byte(respBody), &nowTimeResponse)
	if err != nil {
		log.Fatal(err.Error())
	}
	resp.Body.Close()

	xNonce := uuid.New().String()
	xTime := fmt.Sprint(nowTimeResponse.ServerTime)
	xRequestId := uuid.New().String()

	requestJsonBody, err := json.Marshal(niceHashRequest)
	if err != nil {
		log.Fatalf("[Request] %s", err.Error())
	}

	requestMethod := "POST"
	requestPath := "/main/api/v2/mining/rigs/status2"
	requestQueryString := ""
	requestUrl := fmt.Sprintf("https://api2.nicehash.com%s", requestPath)

	sigKey := []byte(apiSecret)
	sigInput := []byte(apiKey)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(xTime)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(xNonce)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(xOrganizationId)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(requestMethod)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(requestPath)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(requestQueryString)...)
	sigInput = append(sigInput, 0x00)
	sigInput = append(sigInput, []byte(string(requestJsonBody))...)

	sig := hmac.New(sha256.New, sigKey)
	sig.Write(sigInput)

	apiSignature := hex.EncodeToString(sig.Sum(nil))

	xAuth := fmt.Sprintf("%s:%s", apiKey, apiSignature)

	req, err = http.NewRequest(requestMethod, requestUrl, bytes.NewBuffer(requestJsonBody))
	if err != nil {
		log.Fatalf("[HTTP] %s", err.Error())
		return
	}

	// Add required headers
	req.Header.Set("x-time", xTime)
	req.Header.Set("x-nonce", xNonce)
	req.Header.Set("x-organization-id", xOrganizationId)
	req.Header.Set("x-request-id", xRequestId)
	req.Header.Set("x-auth", xAuth)
	req.Header.Set("content-type", "application/json;charset=UTF-8")

	// Fetch Request
	resp, err = client.Do(req)
	if err != nil {
		log.Fatalf("[HTTP] %s", err.Error())
		return
	}
	// Read Response Body
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return
	}
	resp.Body.Close()

	err = json.Unmarshal([]byte(respBody), &niceHashResponse)
	if err != nil {
		log.Fatal(err.Error())
	}

	if niceHashResponse.Success == true {
		fmt.Printf("[Success] Action %s on rig %s is successfully", action, rigId)
	} else {
		fmt.Printf("[Failed] Response: %s", string(respBody))
	}
}
