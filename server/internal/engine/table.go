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
	IsSpectator      bool
	JoinNextHand     bool
	RequestedSeat    int
	Revealed         bool
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
	deckMode     string
	shortDeck    bool
	smallBlindAt int
	bigBlindAt   int
	canReveal    bool
}

func NewTable(roomID string) *Table {
	return &Table{
		RoomID:       roomID,
		SmallBlind:   10,
		BigBlind:     20,
		MaxSeats:     9,
		Players:      map[string]*Player{},
		dealerSeat:   -1,
		actingSeat:   -1,
		phase:        model.PhaseWaiting,
		minRaise:     20,
		deckMode:     "classic",
		smallBlindAt: -1,
		bigBlindAt:   -1,
		canReveal:    false,
	}
}

func (t *Table) AddOrReconnectPlayer(userID, name string, buyIn int64) *Player {
	if p, ok := t.Players[userID]; ok {
		p.Connected = true
		if name != "" {
			p.Name = name
		}
		if buyIn > 0 && p.Chips == 0 && t.phase != model.PhasePreflop && t.phase != model.PhaseFlop && t.phase != model.PhaseTurn && t.phase != model.PhaseRiver && t.phase != model.PhaseShowdown {
			p.Chips = buyIn
		}
		return p
	}
	if buyIn <= 0 {
		buyIn = 2000
	}
	p := &Player{UserID: userID, Name: name, Seat: -1, Chips: buyIn, Connected: true, IsSpectator: true, RequestedSeat: -1}
	if !t.isHandRunning() {
		p.IsSpectator = false
		p.Seat = t.firstFreeSeat()
	}
	t.Players[userID] = p
	return p
}

func (t *Table) DisconnectPlayer(userID string) {
	if p, ok := t.Players[userID]; ok {
		p.Connected = false
	}
}

func (t *Table) RemovePlayer(userID string) {
	delete(t.Players, userID)
	if t.dealerSeat >= 0 && t.playerBySeat(t.dealerSeat) == nil {
		t.dealerSeat = -1
	}
	if t.actingSeat >= 0 && t.playerBySeat(t.actingSeat) == nil {
		t.actingSeat = -1
	}
	if t.smallBlindAt >= 0 && t.playerBySeat(t.smallBlindAt) == nil {
		t.smallBlindAt = -1
	}
	if t.bigBlindAt >= 0 && t.playerBySeat(t.bigBlindAt) == nil {
		t.bigBlindAt = -1
	}
}

func (t *Table) SetDeckMode(mode string) {
	switch mode {
	case "short":
		t.deckMode = "short"
		t.shortDeck = true
	default:
		t.deckMode = "classic"
		t.shortDeck = false
	}
}

func (t *Table) RequestJoinTable(userID string, seat int) error {
	p, ok := t.Players[userID]
	if !ok {
		return errors.New("player not found")
	}
	if !p.IsSpectator {
		if seat >= 0 {
			return t.ChangeSeat(userID, seat)
		}
		return nil
	}
	if seat >= t.MaxSeats {
		return errors.New("invalid seat")
	}
	if !t.isHandRunning() {
		if err := t.assignSeat(p, seat); err != nil {
			return err
		}
		p.IsSpectator = false
		p.JoinNextHand = false
		return nil
	}
	p.JoinNextHand = true
	p.RequestedSeat = seat
	return nil
}

func (t *Table) ChangeSeat(userID string, seat int) error {
	if seat < 0 || seat >= t.MaxSeats {
		return errors.New("invalid seat")
	}
	p, ok := t.Players[userID]
	if !ok {
		return errors.New("player not found")
	}
	if p.IsSpectator {
		p.RequestedSeat = seat
		p.JoinNextHand = true
		return nil
	}
	if t.isHandRunning() {
		return errors.New("can only change seat between hands")
	}
	if p.Seat == seat {
		return nil
	}
	other := t.playerBySeat(seat)
	if other == nil {
		p.Seat = seat
		return nil
	}
	if other.IsSpectator {
		p.Seat = seat
		return nil
	}
	other.Seat, p.Seat = p.Seat, other.Seat
	return nil
}

func (t *Table) StartHand() error {
	if t.phase != model.PhaseWaiting && t.phase != model.PhaseComplete {
		return errors.New("a hand is already running")
	}
	t.activateQueuedPlayers()
	active := t.activeForNewHand()
	if len(active) < 2 {
		return errors.New("need at least 2 players with chips")
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Seat < active[j].Seat })
	t.handID++
	t.phase = model.PhasePreflop
	t.pot = 0
	t.board = nil
	t.deck = newDeck(t.shortDeck)
	t.winners = nil
	t.roundMessage = ""
	t.canReveal = false
	t.currentBet = 0
	t.minRaise = t.BigBlind

	for _, p := range t.Players {
		p.CurrentBet = 0
		p.TotalBet = 0
		p.Cards = nil
		p.Folded = true
		p.AllIn = false
		p.ContributedRound = false
		p.Revealed = false
	}
	for _, p := range active {
		p.Folded = false
		p.Cards = []model.Card{t.draw(), t.draw()}
	}

	t.dealerSeat = t.nextSeatFrom(t.dealerSeat, func(p *Player) bool {
		return !p.Folded && !p.IsSpectator && p.Chips >= 0 && len(p.Cards) > 0
	})
	sbSeat := -1
	bbSeat := -1
	if len(active) == 2 {
		// Heads-up: dealer is small blind and acts first preflop.
		sbSeat = t.dealerSeat
		bbSeat = t.nextSeatFrom(sbSeat, func(p *Player) bool { return !p.Folded && !p.IsSpectator && len(p.Cards) > 0 })
	} else {
		sbSeat = t.nextSeatFrom(t.dealerSeat, func(p *Player) bool { return !p.Folded && !p.IsSpectator && len(p.Cards) > 0 })
		bbSeat = t.nextSeatFrom(sbSeat, func(p *Player) bool { return !p.Folded && !p.IsSpectator && len(p.Cards) > 0 })
	}
	t.smallBlindAt = sbSeat
	t.bigBlindAt = bbSeat
	t.postBlind(sbSeat, t.SmallBlind)
	t.postBlind(bbSeat, t.BigBlind)
	// Posting blinds is forced, not a voluntary decision in this betting round.
	// Big blind must still have an option when action returns preflop.
	for _, p := range active {
		if !p.AllIn {
			p.ContributedRound = false
		}
	}

	if len(active) == 2 {
		t.actingSeat = sbSeat
	} else {
		t.actingSeat = t.nextSeatFrom(bbSeat, func(p *Player) bool {
			return !p.Folded && !p.AllIn && !p.IsSpectator && len(p.Cards) > 0
		})
	}
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
	if p.Folded || p.AllIn || p.IsSpectator || len(p.Cards) == 0 {
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
	for t.roundShouldAdvance() {
		t.advanceStreet()
		if t.resolveIfOnlyOneAlive() {
			return nil
		}
		if t.phase == model.PhaseShowdown {
			t.resolveShowdown()
			return nil
		}
		// All active players are all-in: auto-run remaining streets without waiting for input.
		if len(t.playersToAct()) != 0 {
			break
		}
	}
	t.actingSeat = t.nextSeatFrom(t.actingSeat, func(np *Player) bool {
		return !np.Folded && !np.AllIn && !np.IsSpectator && len(np.Cards) > 0
	})
	return nil
}

func (t *Table) SnapshotFor(userID, hostUserID string) model.Snapshot {
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
			IsSpectator:  p.IsSpectator,
			JoinNextHand: p.JoinNextHand,
			IsHost:       p.UserID == hostUserID,
			CardsCount:   len(p.Cards),
			ShownCards: func() []model.Card {
				if p.Revealed {
					return append([]model.Card(nil), p.Cards...)
				}
				return nil
			}(),
			IsDealer:     !p.IsSpectator && p.Seat == t.dealerSeat,
			IsSmallBlind: !p.IsSpectator && p.Seat == t.smallBlindSeat(),
			IsBigBlind:   !p.IsSpectator && p.Seat == t.bigBlindSeat(),
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
		DeckMode:     t.deckMode,
		HostUserID:   hostUserID,
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
	s.CanReveal = t.canReveal && len(s.YourCards) > 0
	return s
}

func (t *Table) playersSorted() []*Player {
	out := make([]*Player, 0, len(t.Players))
	for _, p := range t.Players {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		si, sj := out[i].Seat, out[j].Seat
		if si < 0 && sj >= 0 {
			return false
		}
		if si >= 0 && sj < 0 {
			return true
		}
		if si == sj {
			return out[i].UserID < out[j].UserID
		}
		return si < sj
	})
	return out
}

func (t *Table) activeForNewHand() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if p.Connected && !p.IsSpectator && p.Chips > 0 && p.Seat >= 0 {
			out = append(out, p)
		}
	}
	return out
}

func (t *Table) playersAlive() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if !p.Folded && !p.IsSpectator && len(p.Cards) > 0 {
			out = append(out, p)
		}
	}
	return out
}

func (t *Table) playersToAct() []*Player {
	out := make([]*Player, 0)
	for _, p := range t.Players {
		if !p.Folded && !p.AllIn && !p.IsSpectator && len(p.Cards) > 0 {
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
		if !p.IsSpectator && p.Seat >= 0 {
			used[p.Seat] = true
		}
	}
	for i := 0; i < t.MaxSeats; i++ {
		if !used[i] {
			return i
		}
	}
	return -1
}

func (t *Table) playerBySeat(seat int) *Player {
	for _, p := range t.Players {
		if !p.IsSpectator && p.Seat == seat {
			return p
		}
	}
	return nil
}

func (t *Table) resetRoundContributions(exceptUser string) {
	for _, p := range t.Players {
		if p.UserID != exceptUser && !p.Folded && !p.AllIn && !p.IsSpectator && len(p.Cards) > 0 {
			p.ContributedRound = false
		}
	}
}

func (t *Table) roundShouldAdvance() bool {
	actors := t.playersToAct()
	if len(actors) == 0 {
		return true
	}
	if len(actors) == 1 {
		// Single remaining actor still needs to respond if facing an unmatched bet.
		if actors[0].CurrentBet < t.currentBet {
			return false
		}
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
		if !p.Folded && !p.AllIn && !p.IsSpectator {
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

	if t.phase == model.PhaseShowdown {
		t.actingSeat = -1
		return
	}
	start := t.nextSeatFrom(t.dealerSeat, func(p *Player) bool {
		return !p.Folded && !p.AllIn && !p.IsSpectator && len(p.Cards) > 0
	})
	t.actingSeat = t.nextSeatFrom(start, func(p *Player) bool {
		return !p.Folded && !p.AllIn && !p.IsSpectator && len(p.Cards) > 0
	})
	if start >= 0 {
		t.actingSeat = start
	}
}

func (t *Table) resolveIfOnlyOneAlive() bool {
	alive := t.playersAlive()
	if len(alive) != 1 {
		return false
	}
	w := alive[0]
	w.Chips += t.pot
	t.winners = []model.WinnerView{{
		UserID:       w.UserID,
		Name:         w.Name,
		Amount:       t.pot,
		PotShare:     t.pot,
		Contribution: w.TotalBet,
		NetGain:      t.pot - w.TotalBet,
		HandTag:      "Won by fold",
	}}
	t.roundMessage = fmt.Sprintf("%s wins %d by fold", w.Name, t.pot)
	t.phase = model.PhaseComplete
	t.canReveal = true
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
		handRanks[p.UserID] = bestOfSeven(append(append([]model.Card{}, p.Cards...), t.board...), t.shortDeck)
		p.Revealed = true
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
			if !p.Folded && !p.IsSpectator && len(p.Cards) > 0 {
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
			UserID:       p.UserID,
			Name:         p.Name,
			Amount:       amount,
			PotShare:     amount,
			Contribution: p.TotalBet,
			NetGain:      amount - p.TotalBet,
			HandTag:      tag,
		})
	}
	t.roundMessage = "Showdown complete"
	t.phase = model.PhaseComplete
	t.canReveal = false
	t.pot = 0
	t.actingSeat = -1
}

func (t *Table) smallBlindSeat() int {
	return t.smallBlindAt
}

func (t *Table) bigBlindSeat() int {
	return t.bigBlindAt
}

func (t *Table) activateQueuedPlayers() {
	for _, p := range t.playersSorted() {
		if !p.IsSpectator || !p.JoinNextHand || p.Chips <= 0 || !p.Connected {
			continue
		}
		if err := t.assignSeat(p, p.RequestedSeat); err != nil {
			continue
		}
		p.IsSpectator = false
		p.JoinNextHand = false
	}
}

func (t *Table) assignSeat(p *Player, requested int) error {
	if requested >= t.MaxSeats {
		return errors.New("invalid seat")
	}
	if requested >= 0 {
		other := t.playerBySeat(requested)
		if other == nil {
			p.Seat = requested
			return nil
		}
		if p.IsSpectator {
			return errors.New("seat occupied")
		}
		other.Seat, p.Seat = p.Seat, other.Seat
		return nil
	}
	free := t.firstFreeSeat()
	if free < 0 {
		return errors.New("no empty seat")
	}
	p.Seat = free
	return nil
}

func (t *Table) isHandRunning() bool {
	return t.phase != model.PhaseWaiting && t.phase != model.PhaseComplete
}

func (t *Table) RestartHand() error {
	if t.isHandRunning() {
		t.phase = model.PhaseComplete
		t.pot = 0
		t.currentBet = 0
		t.minRaise = t.BigBlind
		t.board = nil
		t.winners = nil
		t.actingSeat = -1
		t.roundMessage = "Hand restarted by host"
	}
	return t.StartHand()
}

func (t *Table) RevealCards(userID string) error {
	if !t.canReveal || t.phase != model.PhaseComplete {
		return errors.New("reveal is not available")
	}
	p, ok := t.Players[userID]
	if !ok {
		return errors.New("player not found")
	}
	if len(p.Cards) == 0 {
		return errors.New("no cards to reveal")
	}
	p.Revealed = true
	t.roundMessage = fmt.Sprintf("%s revealed cards", p.Name)
	return nil
}
