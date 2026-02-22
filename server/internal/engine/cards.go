package engine

import (
	"math/rand"
	"time"

	"texaspoker/server/internal/model"
)

var suits = []model.Suit{model.SuitSpade, model.SuitHeart, model.SuitDiamond, model.SuitClub}
var fullDeckRanks = []model.Rank{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
var shortDeckRanks = []model.Rank{"6", "7", "8", "9", "T", "J", "Q", "K", "A"}

func newDeck(shortDeck bool) []model.Card {
	deck := make([]model.Card, 0, 52)
	ranks := fullDeckRanks
	if shortDeck {
		ranks = shortDeckRanks
	}
	for _, s := range suits {
		for _, r := range ranks {
			deck = append(deck, model.Card{Suit: s, Rank: r})
		}
	}
	seeded := rand.New(rand.NewSource(time.Now().UnixNano()))
	seeded.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func rankValue(r model.Rank) int {
	switch r {
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "T":
		return 10
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	default:
		return 0
	}
}
