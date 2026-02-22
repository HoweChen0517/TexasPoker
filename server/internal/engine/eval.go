package engine

import (
	"sort"

	"texaspoker/server/internal/model"
)

type HandRank struct {
	Category int
	Values   []int
	Label    string
}

func compareRank(a, b HandRank) int {
	if a.Category != b.Category {
		if a.Category > b.Category {
			return 1
		}
		return -1
	}
	limit := len(a.Values)
	if len(b.Values) < limit {
		limit = len(b.Values)
	}
	for i := 0; i < limit; i++ {
		if a.Values[i] == b.Values[i] {
			continue
		}
		if a.Values[i] > b.Values[i] {
			return 1
		}
		return -1
	}
	return 0
}

func bestOfSeven(cards []model.Card) HandRank {
	best := HandRank{Category: -1}
	for a := 0; a < len(cards)-4; a++ {
		for b := a + 1; b < len(cards)-3; b++ {
			for c := b + 1; c < len(cards)-2; c++ {
				for d := c + 1; d < len(cards)-1; d++ {
					for e := d + 1; e < len(cards); e++ {
						h := evaluateFive([]model.Card{cards[a], cards[b], cards[c], cards[d], cards[e]})
						if compareRank(h, best) > 0 {
							best = h
						}
					}
				}
			}
		}
	}
	return best
}

func evaluateFive(cards []model.Card) HandRank {
	rankCounts := map[int]int{}
	suitCounts := map[model.Suit]int{}
	ranksDesc := make([]int, 0, 5)

	for _, c := range cards {
		rv := rankValue(c.Rank)
		rankCounts[rv]++
		suitCounts[c.Suit]++
		ranksDesc = append(ranksDesc, rv)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ranksDesc)))

	flush := false
	for _, count := range suitCounts {
		if count == 5 {
			flush = true
			break
		}
	}

	unique := make([]int, 0, len(rankCounts))
	for r := range rankCounts {
		unique = append(unique, r)
	}
	sort.Ints(unique)
	straightHigh := 0
	if len(unique) == 5 {
		if unique[4]-unique[0] == 4 {
			straightHigh = unique[4]
		} else if unique[0] == 2 && unique[1] == 3 && unique[2] == 4 && unique[3] == 5 && unique[4] == 14 {
			straightHigh = 5
		}
	}

	type pair struct {
		r int
		c int
	}
	groups := make([]pair, 0, len(rankCounts))
	for r, c := range rankCounts {
		groups = append(groups, pair{r: r, c: c})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].c == groups[j].c {
			return groups[i].r > groups[j].r
		}
		return groups[i].c > groups[j].c
	})

	if flush && straightHigh > 0 {
		return HandRank{Category: 8, Values: []int{straightHigh}, Label: "Straight Flush"}
	}
	if groups[0].c == 4 {
		kicker := 0
		for _, g := range groups[1:] {
			if g.c == 1 {
				kicker = g.r
				break
			}
		}
		return HandRank{Category: 7, Values: []int{groups[0].r, kicker}, Label: "Four of a Kind"}
	}
	if groups[0].c == 3 && groups[1].c == 2 {
		return HandRank{Category: 6, Values: []int{groups[0].r, groups[1].r}, Label: "Full House"}
	}
	if flush {
		return HandRank{Category: 5, Values: ranksDesc, Label: "Flush"}
	}
	if straightHigh > 0 {
		return HandRank{Category: 4, Values: []int{straightHigh}, Label: "Straight"}
	}
	if groups[0].c == 3 {
		kickers := make([]int, 0, 2)
		for _, g := range groups[1:] {
			if g.c == 1 {
				kickers = append(kickers, g.r)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(kickers)))
		return HandRank{Category: 3, Values: append([]int{groups[0].r}, kickers...), Label: "Three of a Kind"}
	}
	if groups[0].c == 2 && groups[1].c == 2 {
		highPair := groups[0].r
		lowPair := groups[1].r
		if lowPair > highPair {
			highPair, lowPair = lowPair, highPair
		}
		kicker := 0
		for _, g := range groups[2:] {
			if g.c == 1 {
				kicker = g.r
				break
			}
		}
		return HandRank{Category: 2, Values: []int{highPair, lowPair, kicker}, Label: "Two Pair"}
	}
	if groups[0].c == 2 {
		kickers := make([]int, 0, 3)
		for _, g := range groups[1:] {
			if g.c == 1 {
				kickers = append(kickers, g.r)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(kickers)))
		return HandRank{Category: 1, Values: append([]int{groups[0].r}, kickers...), Label: "One Pair"}
	}
	return HandRank{Category: 0, Values: ranksDesc, Label: "High Card"}
}
