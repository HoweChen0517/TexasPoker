package engine

import (
	"testing"

	"texaspoker/server/internal/model"
)

func c(rank model.Rank, suit model.Suit) model.Card {
	return model.Card{Rank: rank, Suit: suit}
}

func TestResolveShowdown_SidePotAllIn(t *testing.T) {
	tb := NewTable("r1")
	tb.phase = model.PhaseShowdown
	tb.board = []model.Card{c("Q", model.SuitSpade), c("J", model.SuitSpade), c("2", model.SuitSpade), c("5", model.SuitDiamond), c("9", model.SuitClub)}

	a := &Player{UserID: "A", Name: "A", Seat: 0, Chips: 0, TotalBet: 100, Cards: []model.Card{c("A", model.SuitSpade), c("K", model.SuitSpade)}}
	b := &Player{UserID: "B", Name: "B", Seat: 1, Chips: 0, TotalBet: 300, Cards: []model.Card{c("A", model.SuitHeart), c("A", model.SuitClub)}}
	cc := &Player{UserID: "C", Name: "C", Seat: 2, Chips: 0, TotalBet: 300, Cards: []model.Card{c("8", model.SuitHeart), c("7", model.SuitDiamond)}}

	tb.Players[a.UserID] = a
	tb.Players[b.UserID] = b
	tb.Players[cc.UserID] = cc
	tb.pot = 700

	tb.resolveShowdown()

	if a.Chips != 300 {
		t.Fatalf("A chips got %d want 300", a.Chips)
	}
	if b.Chips != 400 {
		t.Fatalf("B chips got %d want 400", b.Chips)
	}
	if cc.Chips != 0 {
		t.Fatalf("C chips got %d want 0", cc.Chips)
	}
	if tb.pot != 0 {
		t.Fatalf("pot should be 0, got %d", tb.pot)
	}
}

func TestResolveShowdown_FoldedContributorNotEligible(t *testing.T) {
	tb := NewTable("r2")
	tb.phase = model.PhaseShowdown
	tb.board = []model.Card{c("T", model.SuitSpade), c("8", model.SuitHeart), c("6", model.SuitClub), c("4", model.SuitDiamond), c("2", model.SuitSpade)}

	a := &Player{UserID: "A", Name: "A", Seat: 0, Chips: 0, TotalBet: 100, Cards: []model.Card{c("A", model.SuitHeart), c("A", model.SuitDiamond)}}
	b := &Player{UserID: "B", Name: "B", Seat: 1, Chips: 0, TotalBet: 300, Folded: true, Cards: []model.Card{c("K", model.SuitHeart), c("K", model.SuitDiamond)}}
	cc := &Player{UserID: "C", Name: "C", Seat: 2, Chips: 0, TotalBet: 300, Cards: []model.Card{c("Q", model.SuitHeart), c("Q", model.SuitDiamond)}}

	tb.Players[a.UserID] = a
	tb.Players[b.UserID] = b
	tb.Players[cc.UserID] = cc
	tb.pot = 700

	tb.resolveShowdown()

	if a.Chips != 300 {
		t.Fatalf("A chips got %d want 300", a.Chips)
	}
	if cc.Chips != 400 {
		t.Fatalf("C chips got %d want 400", cc.Chips)
	}
	if b.Chips != 0 {
		t.Fatalf("B chips got %d want 0", b.Chips)
	}
}

func TestAllInDoesNotSkipPendingResponse(t *testing.T) {
	tb := NewTable("r3")
	tb.phase = model.PhasePreflop
	tb.currentBet = 20
	tb.minRaise = 20
	tb.actingSeat = 0

	a := &Player{
		UserID:      "A",
		Name:        "A",
		Seat:        0,
		Chips:       100,
		CurrentBet:  10,
		TotalBet:    10,
		Cards:       []model.Card{c("A", model.SuitSpade), c("K", model.SuitSpade)},
		Connected:   true,
		IsSpectator: false,
	}
	b := &Player{
		UserID:      "B",
		Name:        "B",
		Seat:        1,
		Chips:       200,
		CurrentBet:  20,
		TotalBet:    20,
		Cards:       []model.Card{c("Q", model.SuitHeart), c("Q", model.SuitClub)},
		Connected:   true,
		IsSpectator: false,
	}

	tb.Players[a.UserID] = a
	tb.Players[b.UserID] = b
	tb.dealerSeat = 0
	tb.smallBlindAt = 0
	tb.bigBlindAt = 1

	if err := tb.ApplyAction("A", model.ActionInput{Type: model.ActionAllIn}); err != nil {
		t.Fatalf("all-in action should succeed: %v", err)
	}
	if tb.phase != model.PhasePreflop {
		t.Fatalf("hand should stay preflop awaiting response, got %s", tb.phase)
	}
	if tb.actingSeat != 1 {
		t.Fatalf("acting seat should move to B, got %d", tb.actingSeat)
	}
	if tb.currentBet <= b.CurrentBet {
		t.Fatalf("B should face a higher bet after A all-in")
	}
}
