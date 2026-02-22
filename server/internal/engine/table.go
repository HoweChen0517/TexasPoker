package engine

import (
	"errors"
	"fmt"
	"sort"

	"texaspoker/server/internal/model"
)

type Player struct {
	UserID           string
	Name             string
	Seat             int
	Chips            int64
	CurrentBet       int64
	TotalBet         int64
	Cards            []model.Card
	Folded           bool
	AllIn            bool
	Connected        bool
	ContributedRound bool
}

type Table struct {
	RoomID       string
	SmallBlind   int64
	BigBlind     int64
	MaxSeats     int
	Players      map[string]*Player
	dealerSeat   int
	actingSeat   int
	phase        model.Phase
	handID       int64
	pot          int64
	currentBet   int64
	minRaise     int64
	board        []model.Card
	deck         []model.Card
	winners      []model.WinnerView
	roundMessage string
}

func NewTable(roomID string) *Table {
	return &Table{
		RoomID:     roomID,
		SmallBlind: 10,
		BigBlind:   20,
		MaxSeats:   9,
		Players:    map[string]*Player{},
		dealerSeat: -1,
		actingSeat: -1,
		phase:      model.PhaseWaiting,
		minRaise:   20,
	}
}

func (t *Table) AddOrReconnectPlayer(userID, name string) *Player {
	if p, ok := t.Players[userID]; ok {
		p.Connected = true
		if name != "" {
			p.Name = name
		}
		return p
	}
	seat := t.firstFreeSeat()
	p := &Player{UserID: userID, Name: name, Seat: seat, Chips: 2000, Connected: true}
	t.Players[userID] = p
	return p
}

func (t *Table) DisconnectPlayer(userID string) {
	if p, ok := t.Players[userID]; ok {
		p.Connected = false
	}
}

func (t *Table) StartHand() error {
	if t.phase != model.PhaseWaiting && t.phase != model.PhaseComplete {
		return errors.New("a hand is already running")
	}
	active := t.activeForNewHand()
	if len(active) < 2 {
		return errors.New("need at least 2 players with chips")
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Seat < active[j].Seat })
	t.handID++
	t.phase = model.PhasePreflop
	t.pot = 0
	t.board = nil
	t.deck = newDeck()
	t.winners = nil
	t.roundMessage = ""
	t.currentBet = 0
	t.minRaise = t.BigBlind

	for _, p := range t.Players {
		p.CurrentBet = 0
		p.TotalBet = 0
		p.Cards = nil
		p.Folded = true
		p.AllIn = false
		p.ContributedRound = false
	}
	for _, p := range active {
		p.Folded = false
		p.Cards = []model.Card{t.draw(), t.draw()}
	}

	t.dealerSeat = t.nextSeatFrom(t.dealerSeat, func(p *Player) bool {
		return !p.Folded && p.Chips > 0
	})
	sbSeat := t.nextSeatFrom(t.dealerSeat, func(p *Player) bool { return !p.Folded && p.Chips > 0 })
	bbSeat := t.nextSeatFrom(sbSeat, func(p *Player) bool { return !p.Folded && p.Chips > 0 })
	t.postBlind(sbSeat, t.SmallBlind)
	t.postBlind(bbSeat, t.BigBlind)

	t.actingSeat = t.nextSeatFrom(bbSeat, func(p *Player) bool {
		return !p.Folded && !p.AllIn && p.Chips >= 0
	})
	t.roundMessage = "Preflop started"
	return nil
}

func (t *Table) ApplyAction(userID string, action model.ActionInput) error {
	p, ok := t.Players[userID]
	if !ok {
		return errors.New("player not found")
	}
	if t.phase == model.PhaseWaiting || t.phase == model.PhaseComplete {
		return errors.New("no active hand")
	}
	if p.Seat != t.actingSeat {
		return errors.New("not your turn")
	}
	if p.Folded || p.AllIn {
		return errors.New("player cannot act")
	}

	callCost := t.currentBet - p.CurrentBet
	if callCost < 0 {
		callCost = 0
	}

	switch action.Type {
	case model.ActionFold:
		p.Folded = true
		t.roundMessage = fmt.Sprintf("%s folded", p.Name)
	case model.ActionCheck:
		if callCost != 0 {
			return errors.New("cannot check when facing a bet")
		}
		p.ContributedRound = true
		t.roundMessage = fmt.Sprintf("%s checked", p.Name)
	case model.ActionCall:
		if callCost == 0 {
			return errors.New("nothing to call")
		}
		if err := t.commitChips(p, callCost); err != nil {
			return err
		}
		p.ContributedRound = true
		t.roundMessage = fmt.Sprintf("%s called %d", p.Name, callCost)
	case model.ActionBet:
		if t.currentBet > 0 {
			return errors.New("cannot bet, use raise")
		}
		if action.Amount < t.BigBlind {
			return fmt.Errorf("bet must be at least %d", t.BigBlind)
		}
		if err := t.commitChips(p, action.Amount); err != nil {
			return err
		}
		t.currentBet = p.CurrentBet
		t.minRaise = action.Amount
		t.resetRoundContributions(p.UserID)
		p.ContributedRound = true
		t.roundMessage = fmt.Sprintf("%s bet %d", p.Name, action.Amount)
	case model.ActionRaise:
		if t.currentBet == 0 {
			return errors.New("no previous bet, use bet")
		}
		target := t.currentBet + action.Amount
		raiseBy := target - t.currentBet
		if raiseBy < t.minRaise {
			return fmt.Errorf("raise must be at least %d", t.minRaise)
		}
		need := target - p.CurrentBet
		if need <= 0 {
			return errors.New("invalid raise")
		}
		if err := t.commitChips(p, need); err != nil {
			return err
		}
		t.currentBet = p.CurrentBet
		t.minRaise = raiseBy
		t.resetRoundContributions(p.UserID)
		p.ContributedRound = true
		t.roundMessage = fmt.Sprintf("%s raised to %d", p.Name, target)
	case model.ActionAllIn:
		if p.Chips <= 0 {
			return errors.New("no chips")
		}
		all := p.Chips
		before := p.CurrentBet
		if err := t.commitChips(p, all); err != nil {
			return err
		}
		target := p.CurrentBet
		if target > t.currentBet {
			raiseBy := target - t.currentBet
			t.currentBet = target
			if raiseBy >= t.minRaise {
				t.minRaise = raiseBy
				t.resetRoundContributions(p.UserID)
			}
		}
		p.ContributedRound = true
		if before < t.currentBet {
			t.roundMessage = fmt.Sprintf("%s all-in for %d", p.Name, target)
		} else {
			t.roundMessage = fmt.Sprintf("%s all-in", p.Name)
		}
	default:
		return errors.New("unsupported action")
	}

	if t.resolveIfOnlyOneAlive() {
		return nil
	}
	if t.roundShouldAdvance() {
		t.advanceStreet()
		if t.resolveIfOnlyOneAlive() {
			return nil
		}
		if t.phase == model.PhaseShowdown {
			t.resolveShowdown()
			return nil
		}
	}
	t.actingSeat = t.nextSeatFrom(t.actingSeat, func(np *Player) bool {
		return !np.Folded && !np.AllIn && np.Chips >= 0
	})
	return nil
}

func (t *Table) SnapshotFor(userID string) model.Snapshot {
	players := make([]model.PlayerView, 0, len(t.Players))
	for _, p := range t.playersSorted() {
		players = append(players, model.PlayerView{
			UserID:       p.UserID,
			Name:         p.Name,
			Seat:         p.Seat,
			Chips:        p.Chips,
			CurrentBet:   p.CurrentBet,
			HasFolded:    p.Folded,
			IsAllIn:      p.AllIn,
			IsConnected:  p.Connected,
			CardsCount:   len(p.Cards),
			IsDealer:     p.Seat == t.dealerSeat,
			IsSmallBlind: p.Seat == t.smallBlindSeat(),
			IsBigBlind:   p.Seat == t.bigBlindSeat(),
		})
	}
	s := model.Snapshot{
		RoomID:       t.RoomID,
		HandID:       t.handID,
		Phase:        t.phase,
		Pot:          t.pot,
		CurrentBet:   t.currentBet,
		MinRaise:     t.minRaise,
		BlindSmall:   t.SmallBlind,
		BlindBig:     t.BigBlind,
		DealerSeat:   t.dealerSeat,
		ActingSeat:   t.actingSeat,
		Board:        append([]model.Card(nil), t.board...),
		Players:      players,
		RoundMessage: t.roundMessage,
		Winners:      append([]model.WinnerView(nil), t.winners...),
	}
	if p, ok := t.Players[userID]; ok {
		s.YourCards = append([]model.Card(nil), p.Cards...)
	}
	return s
}

func (t *Table) playersSorted() []*Player {
	out := make([]*Player, 0, len(t.Players))
	for _, p := range t.Players {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seat < out[j].Seat })
	return out
}

func (t *Table) activeForNewHand() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if p.Connected && p.Chips > 0 {
			out = append(out, p)
		}
	}
	return out
}

func (t *Table) playersAlive() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if !p.Folded && len(p.Cards) > 0 {
			out = append(out, p)
		}
	}
	return out
}

func (t *Table) playersToAct() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if !p.Folded && !p.AllIn && len(p.Cards) > 0 {
			out = append(out, p)
		}
	}
	return out
}

func (t *Table) commitChips(p *Player, amount int64) error {
	if amount < 0 {
		return errors.New("invalid amount")
	}
	if amount > p.Chips {
		return errors.New("insufficient chips")
	}
	p.Chips -= amount
	p.CurrentBet += amount
	p.TotalBet += amount
	t.pot += amount
	if p.Chips == 0 {
		p.AllIn = true
	}
	return nil
}

func (t *Table) postBlind(seat int, blind int64) {
	if seat < 0 {
		return
	}
	p := t.playerBySeat(seat)
	if p == nil || p.Folded {
		return
	}
	if blind > p.Chips {
		blind = p.Chips
	}
	_ = t.commitChips(p, blind)
	p.ContributedRound = true
	if p.CurrentBet > t.currentBet {
		t.currentBet = p.CurrentBet
	}
}

func (t *Table) draw() model.Card {
	c := t.deck[0]
	t.deck = t.deck[1:]
	return c
}

func (t *Table) nextSeatFrom(start int, predicate func(*Player) bool) int {
	if len(t.Players) == 0 {
		return -1
	}
	for step := 1; step <= t.MaxSeats; step++ {
		seat := (start + step + t.MaxSeats) % t.MaxSeats
		p := t.playerBySeat(seat)
		if p != nil && predicate(p) {
			return seat
		}
	}
	return -1
}

func (t *Table) firstFreeSeat() int {
	used := map[int]bool{}
	for _, p := range t.Players {
		used[p.Seat] = true
	}
	for i := 0; i < t.MaxSeats; i++ {
		if !used[i] {
			return i
		}
	}
	return len(t.Players) % t.MaxSeats
}

func (t *Table) playerBySeat(seat int) *Player {
	for _, p := range t.Players {
		if p.Seat == seat {
			return p
		}
	}
	return nil
}

func (t *Table) resetRoundContributions(exceptUser string) {
	for _, p := range t.Players {
		if p.UserID != exceptUser && !p.Folded && !p.AllIn {
			p.ContributedRound = false
		}
	}
}

func (t *Table) roundShouldAdvance() bool {
	actors := t.playersToAct()
	if len(actors) == 0 {
		return true
	}
	for _, p := range actors {
		if p.CurrentBet != t.currentBet {
			return false
		}
		if !p.ContributedRound {
			return false
		}
	}
	return true
}

func (t *Table) advanceStreet() {
	for _, p := range t.Players {
		p.CurrentBet = 0
		if !p.Folded && !p.AllIn {
			p.ContributedRound = false
		}
	}
	t.currentBet = 0
	t.minRaise = t.BigBlind

	switch t.phase {
	case model.PhasePreflop:
		t.phase = model.PhaseFlop
		t.board = append(t.board, t.draw(), t.draw(), t.draw())
		t.roundMessage = "Flop"
	case model.PhaseFlop:
		t.phase = model.PhaseTurn
		t.board = append(t.board, t.draw())
		t.roundMessage = "Turn"
	case model.PhaseTurn:
		t.phase = model.PhaseRiver
		t.board = append(t.board, t.draw())
		t.roundMessage = "River"
	case model.PhaseRiver:
		t.phase = model.PhaseShowdown
		t.roundMessage = "Showdown"
	}

	start := t.dealerSeat
	if t.phase == model.PhasePreflop {
		start = t.bigBlindSeat()
	}
	t.actingSeat = t.nextSeatFrom(start, func(p *Player) bool {
		return !p.Folded && !p.AllIn && len(p.Cards) > 0
	})
}

func (t *Table) resolveIfOnlyOneAlive() bool {
	alive := t.playersAlive()
	if len(alive) != 1 {
		return false
	}
	w := alive[0]
	w.Chips += t.pot
	t.winners = []model.WinnerView{{UserID: w.UserID, Name: w.Name, Amount: t.pot, HandTag: "Won by fold"}}
	t.roundMessage = fmt.Sprintf("%s wins %d by fold", w.Name, t.pot)
	t.phase = model.PhaseComplete
	t.pot = 0
	t.actingSeat = -1
	return true
}

func (t *Table) resolveShowdown() {
	alive := t.playersAlive()
	if len(alive) == 0 {
		t.phase = model.PhaseComplete
		return
	}

	handRanks := map[string]HandRank{}
	for _, p := range alive {
		handRanks[p.UserID] = bestOfSeven(append(append([]model.Card{}, p.Cards...), t.board...))
	}

	players := t.playersSorted()
	levelsMap := map[int64]bool{}
	for _, p := range players {
		if p.TotalBet > 0 {
			levelsMap[p.TotalBet] = true
		}
	}
	levels := make([]int64, 0, len(levelsMap))
	for lv := range levelsMap {
		levels = append(levels, lv)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	payout := map[string]int64{}
	prev := int64(0)
	for _, lv := range levels {
		slice := lv - prev
		if slice <= 0 {
			continue
		}

		contributors := make([]*Player, 0)
		for _, p := range players {
			if p.TotalBet >= lv {
				contributors = append(contributors, p)
			}
		}
		if len(contributors) == 0 {
			prev = lv
			continue
		}
		potSize := slice * int64(len(contributors))

		eligible := make([]*Player, 0)
		for _, p := range contributors {
			if !p.Folded && len(p.Cards) > 0 {
				eligible = append(eligible, p)
			}
		}
		if len(eligible) == 0 {
			prev = lv
			continue
		}

		best := HandRank{Category: -1}
		for _, p := range eligible {
			if compareRank(handRanks[p.UserID], best) > 0 {
				best = handRanks[p.UserID]
			}
		}

		winners := make([]*Player, 0)
		for _, p := range eligible {
			if compareRank(handRanks[p.UserID], best) == 0 {
				winners = append(winners, p)
			}
		}
		sort.Slice(winners, func(i, j int) bool { return winners[i].Seat < winners[j].Seat })
		share := potSize / int64(len(winners))
		remain := potSize - share*int64(len(winners))
		for idx, w := range winners {
			gain := share
			if int64(idx) < remain {
				gain++
			}
			payout[w.UserID] += gain
		}
		prev = lv
	}

	t.winners = make([]model.WinnerView, 0, len(payout))
	for _, p := range players {
		amount := payout[p.UserID]
		if amount <= 0 {
			continue
		}
		p.Chips += amount
		tag := handRanks[p.UserID].Label
		t.winners = append(t.winners, model.WinnerView{
			UserID:  p.UserID,
			Name:    p.Name,
			Amount:  amount,
			HandTag: tag,
		})
	}
	t.roundMessage = "Showdown complete"
	t.phase = model.PhaseComplete
	t.pot = 0
	t.actingSeat = -1
}

func (t *Table) smallBlindSeat() int {
	if t.dealerSeat < 0 {
		return -1
	}
	return t.nextSeatFrom(t.dealerSeat, func(p *Player) bool { return !p.Folded && len(p.Cards) > 0 })
}

func (t *Table) bigBlindSeat() int {
	sb := t.smallBlindSeat()
	if sb < 0 {
		return -1
	}
	return t.nextSeatFrom(sb, func(p *Player) bool { return !p.Folded && len(p.Cards) > 0 })
}
