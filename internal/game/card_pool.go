package game

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
)

//go:embed configs/cards.json
var embeddedCardLibrary []byte

type CardLibraryConfig struct {
	ActivePoolSize int                   `json:"activePoolSize"`
	RequiredGroups []CardPoolRequirement `json:"requiredGroups"`
	Cards          []CardConfig          `json:"cards"`
}

type CardPoolRequirement struct {
	ID           string   `json:"id"`
	Min          int      `json:"min"`
	Tags         []string `json:"tags,omitempty"`
	Complexities []string `json:"complexities,omitempty"`
}

type CardConfig struct {
	ID          CardID   `json:"id"`
	Name        string   `json:"name"`
	Cost        int      `json:"cost"`
	Complexity  string   `json:"complexity"`
	Text        string   `json:"text"`
	Starter     bool     `json:"starter"`
	Reusable    bool     `json:"reusable,omitempty"`
	Unplayable  bool     `json:"unplayable,omitempty"`
	Source      string   `json:"source"`
	Tags        []string `json:"tags,omitempty"`
	Excludes    []CardID `json:"excludes,omitempty"`
	MutexGroups []string `json:"mutexGroups,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

var (
	cardLibraryOnce sync.Once
	cardLibraryCfg  *CardLibraryConfig
	cardLibraryErr  error
)

func mustConfigureCardLibrary() {
	cardLibraryOnce.Do(func() {
		cfg := &CardLibraryConfig{}
		if err := json.Unmarshal(embeddedCardLibrary, cfg); err != nil {
			cardLibraryErr = fmt.Errorf("load card library: %w", err)
			return
		}
		for _, entry := range cfg.Cards {
			def, ok := allCards[entry.ID]
			if !ok {
				cardLibraryErr = fmt.Errorf("card %s exists in config but has no runtime implementation", entry.ID)
				return
			}
			def.ID = entry.ID
			def.Name = entry.Name
			def.Cost = entry.Cost
			def.Complexity = entry.Complexity
			def.Text = entry.Text
			def.Starter = entry.Starter
			def.Reusable = entry.Reusable
			def.Unplayable = entry.Unplayable
			def.Source = entry.Source
			def.Tags = cloneStringSlice(entry.Tags)
			def.Excludes = cloneCardIDs(entry.Excludes)
			def.MutexGroups = cloneStringSlice(entry.MutexGroups)
			def.Enabled = entry.Enabled == nil || *entry.Enabled
			allCards[entry.ID] = def
		}
		cardLibraryCfg = cfg
	})
	if cardLibraryErr != nil {
		panic(cardLibraryErr)
	}
}

func buildActivePool(rng *rand.Rand) []CardID {
	mustConfigureCardLibrary()
	if cardLibraryCfg == nil {
		return nil
	}
	pool := buildConfiguredActivePool(rng, cardLibraryCfg)
	if len(pool) > 0 {
		return pool
	}
	return fallbackActivePool()
}

func buildConfiguredActivePool(rng *rand.Rand, cfg *CardLibraryConfig) []CardID {
	eligible := configuredActiveCandidates()
	if len(eligible) == 0 {
		return nil
	}

	target := cfg.ActivePoolSize
	if target <= 0 {
		target = 20
	}
	if target > len(eligible) {
		target = len(eligible)
	}

	selected := make([]CardID, 0, target)
	selectedSet := map[CardID]struct{}{}
	remaining := cloneCardIDs(eligible)

	for _, requirement := range cfg.RequiredGroups {
		alreadyMatched := countRequirementMatches(selected, requirement)
		needed := requirement.Min - alreadyMatched
		if needed <= 0 {
			continue
		}
		matches := filterCandidates(remaining, func(card CardID) bool {
			def := allCards[card]
			return matchesRequirement(def, requirement) && !conflictsWithSelection(card, selectedSet)
		})
		shuffleCards(rng, matches)
		if needed > target-len(selected) {
			needed = target - len(selected)
		}
		added := 0
		for _, candidate := range matches {
			if added >= needed {
				break
			}
			if conflictsWithSelection(candidate, selectedSet) {
				continue
			}
			selected = append(selected, candidate)
			selectedSet[candidate] = struct{}{}
			remaining = removeCardFromSlice(remaining, candidate)
			added++
		}
	}

	candidates := filterCandidates(remaining, func(card CardID) bool {
		return !conflictsWithSelection(card, selectedSet)
	})
	shuffleCards(rng, candidates)
	for _, card := range candidates {
		if len(selected) >= target {
			break
		}
		if conflictsWithSelection(card, selectedSet) {
			continue
		}
		selected = append(selected, card)
		selectedSet[card] = struct{}{}
	}

	return selected
}

func configuredActiveCandidates() []CardID {
	ids := make([]CardID, 0, len(allCards))
	for id, def := range allCards {
		if !def.Enabled || def.Source == "curse" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func fallbackActivePool() []CardID {
	ids := configuredActiveCandidates()
	if len(ids) == 0 {
		for id := range allCards {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	if len(ids) > 20 {
		ids = ids[:20]
	}
	return ids
}

func matchesRequirement(def CardDef, requirement CardPoolRequirement) bool {
	if requirement.ID == "rare" && def.Source != "rare" {
		return false
	}
	if len(requirement.Tags) > 0 {
		for _, tag := range requirement.Tags {
			if !cardHasTag(def, tag) {
				return false
			}
		}
	}
	if len(requirement.Complexities) > 0 {
		matched := false
		for _, complexity := range requirement.Complexities {
			if def.Complexity == complexity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func countRequirementMatches(cards []CardID, requirement CardPoolRequirement) int {
	count := 0
	for _, card := range cards {
		if matchesRequirement(allCards[card], requirement) {
			count++
		}
	}
	return count
}

func cardHasTag(def CardDef, tag string) bool {
	for _, candidate := range def.Tags {
		if candidate == tag {
			return true
		}
	}
	return false
}

func conflictsWithSelection(candidate CardID, selected map[CardID]struct{}) bool {
	def := allCards[candidate]
	for chosen := range selected {
		if candidate == chosen {
			return true
		}
		chosenDef := allCards[chosen]
		if containsCardID(def.Excludes, chosen) || containsCardID(chosenDef.Excludes, candidate) {
			return true
		}
		if sharesMutexGroup(def.MutexGroups, chosenDef.MutexGroups) {
			return true
		}
	}
	return false
}

func sharesMutexGroup(left []string, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, item := range left {
		seen[item] = struct{}{}
	}
	for _, item := range right {
		if _, ok := seen[item]; ok {
			return true
		}
	}
	return false
}

func containsCardID(list []CardID, target CardID) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func cloneCardIDs(src []CardID) []CardID {
	out := make([]CardID, len(src))
	copy(out, src)
	return out
}

func cloneStringSlice(src []string) []string {
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func filterCandidates(cards []CardID, allow func(CardID) bool) []CardID {
	out := make([]CardID, 0, len(cards))
	for _, card := range cards {
		if allow(card) {
			out = append(out, card)
		}
	}
	return out
}

func removeCardFromSlice(cards []CardID, target CardID) []CardID {
	for i, card := range cards {
		if card == target {
			return append(cards[:i], cards[i+1:]...)
		}
	}
	return cards
}

func (g *Game) buildStarterDeck() []CardID {
	starterPool := g.activeCardsMatching(func(def CardDef) bool {
		return def.Starter && !def.Unplayable && def.Source != "curse"
	})
	if len(starterPool) == 0 {
		return defaultStarterDeck()
	}
	sort.Slice(starterPool, func(i, j int) bool {
		left := allCards[starterPool[i]]
		right := allCards[starterPool[j]]
		if left.Cost == right.Cost {
			return starterPool[i] < starterPool[j]
		}
		return left.Cost < right.Cost
	})
	deck := make([]CardID, 0, 10)
	for _, card := range starterProtectionCards() {
		def, ok := allCards[card]
		if !ok || !def.Enabled || def.Unplayable {
			continue
		}
		deck = append(deck, card)
	}
	for len(deck) < 10 {
		deck = append(deck, starterPool[len(deck)%len(starterPool)])
	}
	return deck[:10]
}

func defaultStarterDeck() []CardID {
	return []CardID{CardReverse, CardReverse, CardHeap, CardHeap, CardBubble, CardBinary, CardNullPointer, CardBacktrack, CardMemoize, CardNoop}
}

func (g *Game) activeCardsMatching(match func(CardDef) bool) []CardID {
	cards := make([]CardID, 0, len(g.activePool))
	for _, id := range g.activePool {
		def, ok := allCards[id]
		if !ok || !def.Enabled {
			continue
		}
		if match != nil && !match(def) {
			continue
		}
		cards = append(cards, id)
	}
	return cards
}

func (g *Game) rewardEligibleCards() []CardID {
	pool := g.activeCardsMatching(func(def CardDef) bool {
		return !def.Unplayable && def.Source != "curse"
	})
	if len(pool) > 0 {
		return pool
	}
	return filterCandidates(fallbackActivePool(), func(card CardID) bool {
		def := allCards[card]
		return !def.Unplayable && def.Source != "curse"
	})
}

func (g *Game) drawRewardCardsFromActivePool(n int) []CardID {
	pool := cloneCardIDs(g.rewardEligibleCards())
	if len(pool) == 0 {
		return nil
	}
	shuffleCards(g.rng, pool)
	if n > len(pool) {
		n = len(pool)
	}
	return cloneCardIDs(pool[:n])
}
