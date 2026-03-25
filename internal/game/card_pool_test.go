package game

import (
	"math/rand"
	"testing"
)

func TestCardLibraryConfigIncludesFiftyCards(t *testing.T) {
	mustConfigureCardLibrary()
	if cardLibraryCfg == nil {
		t.Fatal("expected card library config to be loaded")
	}
	if got := len(cardLibraryCfg.Cards); got != 56 {
		t.Fatalf("expected 56 configured cards, got %d", got)
	}
}

func TestBuildActivePoolRespectsRequirements(t *testing.T) {
	mustConfigureCardLibrary()
	rng := rand.New(rand.NewSource(42))
	pool := buildActivePool(rng)
	if len(pool) == 0 {
		t.Fatal("expected non-empty active pool")
	}

	seen := map[CardID]struct{}{}
	starterCount := 0
	constantCount := 0
	maintenanceCount := 0
	for _, card := range pool {
		if _, ok := seen[card]; ok {
			t.Fatalf("duplicate card in active pool: %s", card)
		}
		seen[card] = struct{}{}
		def := allCards[card]
		if def.Starter {
			starterCount++
		}
		if def.Complexity == "O(1)" {
			constantCount++
		}
		if cardHasTag(def, "maintenance") {
			maintenanceCount++
		}
	}

	if starterCount < 5 {
		t.Fatalf("expected at least 5 starter cards, got %d", starterCount)
	}
	if constantCount < 3 {
		t.Fatalf("expected at least 3 O(1) cards, got %d", constantCount)
	}
	if maintenanceCount < 2 {
		t.Fatalf("expected at least 2 maintenance cards, got %d", maintenanceCount)
	}
}

func TestBuildActivePoolRespectsExclusions(t *testing.T) {
	mustConfigureCardLibrary()
	rng := rand.New(rand.NewSource(7))
	pool := buildActivePool(rng)
	present := map[CardID]struct{}{}
	for _, card := range pool {
		present[card] = struct{}{}
	}

	for _, card := range pool {
		for _, excluded := range allCards[card].Excludes {
			if _, ok := present[excluded]; ok {
				t.Fatalf("card %s should exclude %s from active pool", card, excluded)
			}
		}
	}
}

func TestBuildActivePoolRespectsMutexGroups(t *testing.T) {
	mustConfigureCardLibrary()
	for seed := int64(1); seed <= 64; seed++ {
		pool := buildActivePool(rand.New(rand.NewSource(seed)))
		groups := map[string]CardID{}
		for _, card := range pool {
			for _, group := range allCards[card].MutexGroups {
				if existing, ok := groups[group]; ok {
					t.Fatalf("seed %d produced mutex group conflict %s between %s and %s", seed, group, existing, card)
				}
				groups[group] = card
			}
		}
	}
}
