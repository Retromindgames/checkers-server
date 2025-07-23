package main

import "github.com/Lavizord/checkers-server/models"

func IsValidBet(betValue float64) bool {
	found := false
	for _, v := range models.DamasValidBetAmounts {
		if v == betValue {
			found = true
			break
		}
	}
	return found
}
