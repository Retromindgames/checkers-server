package walletrequests

import (
	"checkers-server/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

func SokkerDuelGetWallet(op models.Operator, token string) (*models.WalletResponse, error) {
	// Parse the base URL	TODO: HANDLE HTTPS
	baseUrl, err := url.Parse(op.OperatorWalletBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %v", err)
	}
	baseUrl.Path = path.Join(baseUrl.Path, "wallet")

	req, err := http.NewRequest("POST", baseUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("x-access-token", token)

	// Send the request using the default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body using io.ReadAll
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var walletResponse models.WalletResponse
	if err := json.Unmarshal(body, &walletResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Return the parsed WalletResponse
	return &walletResponse, nil
}

func SokkerDuelPostBet() {

}

func SokkerDuelPostWin() {

}
