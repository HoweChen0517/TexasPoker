package model

type Suit string

type Rank string

const (
	SuitSpade   Suit = "S"
	SuitHeart   Suit = "H"
	SuitDiamond Suit = "D"
	SuitClub    Suit = "C"
)

type Card struct {
	Suit Suit `json:"suit"`
	Rank Rank `json:"rank"`
}

type Phase string

const (
	PhaseWaiting  Phase = "waiting"
	PhasePreflop  Phase = "preflop"
	PhaseFlop     Phase = "flop"
	PhaseTurn     Phase = "turn"
	PhaseRiver    Phase = "river"
	PhaseShowdown Phase = "showdown"
	PhaseComplete Phase = "complete"
)

type ActionType string

const (
	ActionFold  ActionType = "fold"
	ActionCheck ActionType = "check"
	ActionCall  ActionType = "call"
	ActionBet   ActionType = "bet"
	ActionRaise ActionType = "raise"
	ActionAllIn ActionType = "all_in"
)

type PlayerView struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Seat         int    `json:"seat"`
	Chips        int64  `json:"chips"`
	CurrentBet   int64  `json:"current_bet"`
	HasFolded    bool   `json:"has_folded"`
	IsAllIn      bool   `json:"is_all_in"`
	IsConnected  bool   `json:"is_connected"`
	IsSpectator  bool   `json:"is_spectator"`
	JoinNextHand bool   `json:"join_next_hand"`
	IsHost       bool   `json:"is_host"`
	CardsCount   int    `json:"cards_count"`
	IsDealer     bool   `json:"is_dealer"`
	IsSmallBlind bool   `json:"is_small_blind"`
	IsBigBlind   bool   `json:"is_big_blind"`
}

type Snapshot struct {
	RoomID       string       `json:"room_id"`
	HandID       int64        `json:"hand_id"`
	Phase        Phase        `json:"phase"`
	Pot          int64        `json:"pot"`
	CurrentBet   int64        `json:"current_bet"`
	MinRaise     int64        `json:"min_raise"`
	BlindSmall   int64        `json:"blind_small"`
	BlindBig     int64        `json:"blind_big"`
	DeckMode     string       `json:"deck_mode"`
	HostUserID   string       `json:"host_user_id"`
	DealerSeat   int          `json:"dealer_seat"`
	ActingSeat   int          `json:"acting_seat"`
	Board        []Card       `json:"board"`
	Players      []PlayerView `json:"players"`
	RoundMessage string       `json:"round_message"`
	Winners      []WinnerView `json:"winners"`
	YourCards    []Card       `json:"your_cards"`
}

type WinnerView struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Amount  int64  `json:"amount"`
	HandTag string `json:"hand_tag"`
}

type ActionInput struct {
	Type   ActionType `json:"type"`
	Amount int64      `json:"amount"`
}
