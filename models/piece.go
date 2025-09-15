package models

type PieceInterface interface {
	GetType() string
	GetPlayerID() string
	GetID() string
	IsPieceKinged() bool
	SetIsPieceKinged(value bool)
}

type DamasPiece struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
	PieceID  string `json:"piece_id"`
	IsKinged bool   `json:"is_kinged"`
}

type ChessPiece struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
	PieceID  string `json:"piece_id"`
	Color    string `json:"color"`
	IsAlive  bool   `json:"is_alive"`
}

func (p *DamasPiece) GetID() string               { return p.PieceID }
func (p *DamasPiece) GetType() string             { return p.Type }
func (p *DamasPiece) GetPlayerID() string         { return p.PlayerID }
func (p *DamasPiece) IsPieceKinged() bool         { return p.IsKinged }
func (p *DamasPiece) SetIsPieceKinged(value bool) { p.IsKinged = value }

func (p *ChessPiece) GetID() string               { return p.PieceID }
func (p *ChessPiece) GetType() string             { return p.Type }
func (p *ChessPiece) GetPlayerID() string         { return p.PlayerID }
func (p *ChessPiece) IsPieceKinged() bool         { return false }
func (p *ChessPiece) SetIsPieceKinged(value bool) {}
