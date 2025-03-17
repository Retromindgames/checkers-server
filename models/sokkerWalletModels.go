package models

type Operator struct {
	ID                    int
	OperatorName          string
	OperatorGameName      string
	GameName              string
	Active                bool
	GameBaseUrl           string
	OperatorWalletBaseUrl string
}

type WalletResponse struct {
	Status string `json:"status"`
	Data   struct {
		Username string `json:"username"`
		Balance  int    `json:"balance"`
		Currency string `json:"currency"`
	} `json:"data"`
}

type SokkerDuelGamelaunchResponse struct {
	Token string `json:"token"`
	Url   string `json:"url"`
}
