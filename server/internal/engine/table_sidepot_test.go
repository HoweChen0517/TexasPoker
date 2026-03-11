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

func TestHeadsUpBigBlindGetsOptionAfterSmallBlindCall(t *testing.T) {
	tb := NewTable("r4")
	tb.phase = model.PhasePreflop
	tb.currentBet = 20
	tb.minRaise = 20
	tb.actingSeat = 0 // small blind acts first heads-up preflop
	tb.dealerSeat = 0
	tb.smallBlindAt = 0
	tb.bigBlindAt = 1

	sb := &Player{
		UserID:           "SB",
		Name:             "SB",
		Seat:             0,
		Chips:            190,
		CurrentBet:       10,
		TotalBet:         10,
		Cards:            []model.Card{c("A", model.SuitSpade), c("9", model.SuitHeart)},
		Connected:        true,
		ContributedRound: false,
		IsSpectator:      false,
	}
	bb := &Player{
		UserID:           "BB",
		Name:             "BB",
		Seat:             1,
		Chips:            180,
		CurrentBet:       20,
		TotalBet:         20,
		Cards:            []model.Card{c("K", model.SuitClub), c("Q", model.SuitDiamond)},
		Connected:        true,
		ContributedRound: false,
		IsSpectator:      false,
	}
	tb.Players[sb.UserID] = sb
	tb.Players[bb.UserID] = bb

	if err := tb.ApplyAction("SB", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("small blind call should succeed: %v", err)
	}
	if tb.phase != model.PhasePreflop {
		t.Fatalf("phase should remain preflop, got %s", tb.phase)
	}
	if tb.actingSeat != 1 {
		t.Fatalf("big blind should get decision, got acting seat %d", tb.actingSeat)
	}
}

func TestMultiwayBigBlindGetsOptionWhenActionReturns(t *testing.T) {
	tb := NewTable("r5")
	tb.Players["BTN"] = &Player{UserID: "BTN", Name: "BTN", Seat: 0, Chips: 200, Connected: true}
	tb.Players["SB"] = &Player{UserID: "SB", Name: "SB", Seat: 1, Chips: 200, Connected: true}
	tb.Players["BB"] = &Player{UserID: "BB", Name: "BB", Seat: 2, Chips: 200, Connected: true}
	tb.Players["UTG"] = &Player{UserID: "UTG", Name: "UTG", Seat: 3, Chips: 200, Connected: true}

	if err := tb.StartHand(); err != nil {
		t.Fatalf("start hand should succeed: %v", err)
	}
	if tb.dealerSeat != 0 || tb.smallBlindAt != 1 || tb.bigBlindAt != 2 || tb.actingSeat != 3 {
		t.Fatalf("unexpected opening order dealer=%d sb=%d bb=%d act=%d", tb.dealerSeat, tb.smallBlindAt, tb.bigBlindAt, tb.actingSeat)
	}

	if err := tb.ApplyAction("UTG", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("UTG call should succeed: %v", err)
	}
	if tb.phase != model.PhasePreflop {
		t.Fatalf("phase should still be preflop after UTG action, got %s", tb.phase)
	}
	if tb.actingSeat != 0 {
		t.Fatalf("action should move to button seat 0, got %d", tb.actingSeat)
	}

	if err := tb.ApplyAction("BTN", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("button call should succeed: %v", err)
	}
	if tb.phase != model.PhasePreflop {
		t.Fatalf("phase should still be preflop after button call, got %s", tb.phase)
	}
	if tb.actingSeat != 1 {
		t.Fatalf("small blind should act before big blind, got %d", tb.actingSeat)
	}

	if err := tb.ApplyAction("SB", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("small blind call should succeed: %v", err)
	}
	if tb.phase != model.PhasePreflop {
		t.Fatalf("phase should still be preflop until big blind acts, got %s", tb.phase)
	}
	if tb.actingSeat != 2 {
		t.Fatalf("big blind should get final preflop option, got %d", tb.actingSeat)
	}
}

func TestMultiwayFlopStartsFromSmallBlind(t *testing.T) {
	tb := NewTable("r6")
	tb.Players["BTN"] = &Player{UserID: "BTN", Name: "BTN", Seat: 0, Chips: 200, Connected: true}
	tb.Players["SB"] = &Player{UserID: "SB", Name: "SB", Seat: 1, Chips: 200, Connected: true}
	tb.Players["BB"] = &Player{UserID: "BB", Name: "BB", Seat: 2, Chips: 200, Connected: true}
	tb.Players["UTG"] = &Player{UserID: "UTG", Name: "UTG", Seat: 3, Chips: 200, Connected: true}

	if err := tb.StartHand(); err != nil {
		t.Fatalf("start hand should succeed: %v", err)
	}
	if err := tb.ApplyAction("UTG", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("UTG call should succeed: %v", err)
	}
	if err := tb.ApplyAction("BTN", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("button call should succeed: %v", err)
	}
	if err := tb.ApplyAction("SB", model.ActionInput{Type: model.ActionCall}); err != nil {
		t.Fatalf("small blind call should succeed: %v", err)
	}
	if err := tb.ApplyAction("BB", model.ActionInput{Type: model.ActionCheck}); err != nil {
		t.Fatalf("big blind check should succeed: %v", err)
	}
	if tb.phase != model.PhaseFlop {
		t.Fatalf("phase should advance to flop, got %s", tb.phase)
	}
	if tb.actingSeat != 1 {
		t.Fatalf("small blind should act first on flop, got %d", tb.actingSeat)
	}
}

func TestButtonRotatesAcrossHands(t *testing.T) {
	tb := NewTable("r7")
	tb.Players["P0"] = &Player{UserID: "P0", Name: "P0", Seat: 0, Chips: 200, Connected: true}
	tb.Players["P1"] = &Player{UserID: "P1", Name: "P1", Seat: 1, Chips: 200, Connected: true}
	tb.Players["P2"] = &Player{UserID: "P2", Name: "P2", Seat: 2, Chips: 200, Connected: true}
	tb.Players["P3"] = &Player{UserID: "P3", Name: "P3", Seat: 3, Chips: 200, Connected: true}

	if err := tb.StartHand(); err != nil {
		t.Fatalf("first hand should start: %v", err)
	}
	if tb.dealerSeat != 0 || tb.smallBlindAt != 1 || tb.bigBlindAt != 2 || tb.actingSeat != 3 {
		t.Fatalf("unexpected first hand positions dealer=%d sb=%d bb=%d act=%d", tb.dealerSeat, tb.smallBlindAt, tb.bigBlindAt, tb.actingSeat)
	}

	tb.phase = model.PhaseComplete
	if err := tb.StartHand(); err != nil {
		t.Fatalf("second hand should start: %v", err)
	}
	if tb.dealerSeat != 1 || tb.smallBlindAt != 2 || tb.bigBlindAt != 3 || tb.actingSeat != 0 {
		t.Fatalf("unexpected second hand positions dealer=%d sb=%d bb=%d act=%d", tb.dealerSeat, tb.smallBlindAt, tb.bigBlindAt, tb.actingSeat)
	}
}
