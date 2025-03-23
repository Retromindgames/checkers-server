package walletrequests

import (
	"bytes"
	"checkers-server/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

func SokkerDuelGetWallet(op models.Operator, token string) (*models.WalletResponse, error) {
	// Parse the base URL
	baseUrl, err := url.Parse(op.OperatorWalletBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse base URL: %v", err)
	}
	baseUrl.Path = path.Join(baseUrl.Path, "wallet")

	req, err := http.NewRequest("POST", baseUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}
	req.Header.Set("x-access-token", token)

	// Print request
	fmt.Println("===== API REQUEST =====")
	fmt.Printf("URL: %s\n", baseUrl.String())
	fmt.Printf("Headers: x-access-token=%s\n", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	// Print response
	fmt.Println("===== API RESPONSE =====")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Body: %s\n", string(body))

	// Try to unmarshal as an error first
	var apiError models.SokkerDuelErrorResponse
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Status == "error" {
		return nil, fmt.Errorf(apiError.Resp)
	}

	// If no error, proceed with normal response
	var walletResponse models.WalletResponse
	if err := json.Unmarshal(body, &walletResponse); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %v", err)
	}

	return &walletResponse, nil
}

func SokkerDuelPostBet(session models.Session, betData models.SokkerDuelBet) (*models.SokkerDuelBetResponse, error) {
	baseUrl, err := url.Parse(session.OperatorBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse base URL: %v", err)
	}
	baseUrl.Path = path.Join(baseUrl.Path, "bet")

	jsonData, err := json.Marshal(betData)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize bet data: %v", err)
	}

	// Print request
	fmt.Println("===== API REQUEST =====")
	fmt.Printf("URL: %s\n", baseUrl.String())
	fmt.Printf("Headers: x-access-token=%s\n", session.Token)
	fmt.Printf("Body: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", baseUrl.String(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Failed to create bet request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-access-token", session.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ailed to send bet request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	// Print response
	fmt.Println("===== API RESPONSE =====")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Body: %s\n", string(body))

	// Try to unmarshal as an error first
	var apiError models.SokkerDuelErrorResponse
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Status == "error" {
		return nil, fmt.Errorf("API error: %s", apiError.Resp)
	}

	// If no error, proceed with normal response
	var walletResponse models.SokkerDuelBetResponse
	if err := json.Unmarshal(body, &walletResponse); err != nil {
		return nil, fmt.Errorf("Failed to parse bet response: %v. With err: %v", walletResponse, err)
	}

	return &walletResponse, nil
}

func SokkerDuelPostWin(session models.Session, winData models.SokkerDuelWin) (*models.SokkerDuelWinResponse, error) {
	baseUrl, err := url.Parse(session.OperatorBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse base URL: %v", err)
	}
	baseUrl.Path = path.Join(baseUrl.Path, "win")

	jsonData, err := json.Marshal(winData)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize win data: %v", err)
	}

	// Print request
	fmt.Println("===== API REQUEST =====")
	fmt.Printf("URL: %s\n", baseUrl.String())
	fmt.Printf("Headers: x-access-token=%s\n", session.Token)
	fmt.Printf("Body: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", baseUrl.String(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Failed to create win request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-access-token", session.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send win request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read win response body: %v", err)
	}

	// Print response
	fmt.Println("===== API RESPONSE =====")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Body: %s\n", string(body))

	// Try to unmarshal as an error first
	var apiError models.SokkerDuelErrorResponse
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Status == "error" {
		return nil, fmt.Errorf("API error: %s", apiError.Resp)
	}

	var walletResponse models.SokkerDuelWinResponse
	if err := json.Unmarshal(body, &walletResponse); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %v. With err: %v", walletResponse, err)
	}

	return &walletResponse, nil
}
