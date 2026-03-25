package game

import (
	"fmt"
	"sort"
)

func init() {
	registerExtendedCards()
}

type statScriptOptions struct {
	Damage           int
	IgnoreArmor      bool
	AttackDelta      int
	ArmorDelta       int
	CountdownDelta   int
	ScriptDelta      int
	EntropyDelta     int
	Draw             int
	Heal             int
	MPGain           int
	PointerGain      int
	LoadDelta        int
	PendingLoadDelta int
	PurgeGarbage     bool
	InjectCard       CardID
}

func registerExtendedCards() {
	for id, def := range extendedCards() {
		allCards[id] = def
	}
}

func extendedCards() map[CardID]CardDef {
	return map[CardID]CardDef{
		CardID("fft_scan.m"):           makeFFTScanCard(CardID("fft_scan.m"), 3, "O(n log n)"),
		CardID("cache_prefetch.c"):     makeStatScript(CardID("cache_prefetch.c"), 1, "O(1)", statScriptOptions{MPGain: 1, Draw: 1}),
		CardID("mutex_lock.h"):         makeStatScript(CardID("mutex_lock.h"), 2, "O(1)", statScriptOptions{AttackDelta: -2, CountdownDelta: 1}),
		CardID("linked_splice.js"):     makeLinkedSpliceCard(CardID("linked_splice.js"), 2, "O(n)"),
		CardID("segment_tree.cc"):      makeSegmentTreeCard(CardID("segment_tree.cc"), 4, "O(log n)"),
		CardID("lazy_prop.sql"):        makeStatScript(CardID("lazy_prop.sql"), 2, "O(log n)", statScriptOptions{CountdownDelta: 1, ScriptDelta: -2}),
		CardID("union_find.swift"):     makeStatScript(CardID("union_find.swift"), 1, "O(log n)", statScriptOptions{PendingLoadDelta: -1, Heal: 1, Draw: 1}),
		CardID("topo_sort.hs"):         makeTopoSortCard(CardID("topo_sort.hs"), 3, "O(n)"),
		CardID("dijkstra_route.ex"):    makeDijkstraRouteCard(CardID("dijkstra_route.ex"), 3, "O(n log n)"),
		CardID("bfs_queue.dart"):       makeStatScript(CardID("bfs_queue.dart"), 1, "O(n)", statScriptOptions{CountdownDelta: 1}),
		CardID("dfs_probe.fs"):         makeStatScript(CardID("dfs_probe.fs"), 1, "O(n)", statScriptOptions{Damage: 3, Draw: 1}),
		CardID("rolling_hash.pl"):      makeRollingHashCard(CardID("rolling_hash.pl"), 1, "O(n)"),
		CardID("lru_cache.zig"):        makeLruCacheCard(CardID("lru_cache.zig"), 2, "O(1)"),
		CardID("bitshift_gate.v"):      makeBitshiftGateCard(CardID("bitshift_gate.v"), 1, "O(1)"),
		CardID("stack_unwind.mm"):      makeStatScript(CardID("stack_unwind.mm"), 2, "O(n)", statScriptOptions{PendingLoadDelta: -2, MPGain: 1, Draw: 1}),
		CardID("branch_predictor.clj"): makeBranchPredictorCard(CardID("branch_predictor.clj"), 2, "O(1)"),
		CardID("trie_probe.php"):       makeTrieProbeCard(CardID("trie_probe.php"), 1, "O(log n)"),
		CardID("sparse_table.r"):       makeStatScript(CardID("sparse_table.r"), 2, "O(1)", statScriptOptions{Damage: 4, IgnoreArmor: true, EntropyDelta: -1}),
		CardID("bloom_filter.pm"):      makeStatScript(CardID("bloom_filter.pm"), 1, "O(n)", statScriptOptions{AttackDelta: -2, ScriptDelta: -1, Draw: 1}),
		CardID("cycle_detect.sc"):      makeCycleDetectCard(CardID("cycle_detect.sc"), 2, "O(n)"),
		CardID("packet_sniffer.ps1"):   makeStatScript(CardID("packet_sniffer.ps1"), 2, "O(n)", statScriptOptions{PointerGain: 1, MPGain: 1, Draw: 1}),
		CardID("race_patch.cr"):        makeStatScript(CardID("race_patch.cr"), 2, "O(1)", statScriptOptions{Damage: 2, ArmorDelta: -1, CountdownDelta: 1}),
		CardID("scheduler_tick.nim"):   makeStatScript(CardID("scheduler_tick.nim"), 1, "O(1)", statScriptOptions{PendingLoadDelta: -1, Draw: 1}),
		CardID("deadlock_breaker.ml"):  makeDeadlockBreakerCard(CardID("deadlock_breaker.ml"), 4, "O(n^2)"),
		CardID("sandbox_escape.bat"):   makeStatScript(CardID("sandbox_escape.bat"), 4, "O(1)", statScriptOptions{Damage: 7, IgnoreArmor: true, InjectCard: CardOverflow}),
		CardID("vectorize.simd"):       makeVectorizeCard(CardID("vectorize.simd"), 2, "O(n)"),
		CardID("parity_flip.el"):       makeParityFlipCard(CardID("parity_flip.el"), 1, "O(n)"),
		CardID("entropy_seed.jl"):      makeStatScript(CardID("entropy_seed.jl"), 1, "O(1)", statScriptOptions{EntropyDelta: -4, AttackDelta: -1}),
		CardID("gc_mark.scm"):          makeStatScript(CardID("gc_mark.scm"), 2, "O(n)", statScriptOptions{PurgeGarbage: true, Heal: 3, PendingLoadDelta: -1}),
		CardID("shard_balance.tf"):     makeShardBalanceCard(CardID("shard_balance.tf"), 3, "O(n log n)"),
		CardID("tail_call.lisp"):       makeTailCallCard(CardID("tail_call.lisp"), 1, "O(1)"),
		CardID("diff_fuzzer.groovy"):   makeDiffFuzzerCard(CardID("diff_fuzzer.groovy"), 3, "O(n^2)"),
	}
}

func makeStatScript(id CardID, cost int, complexity string, opts statScriptOptions) CardDef {
	return CardDef{
		ID:         id,
		Name:       string(id),
		Cost:       cost,
		Complexity: complexity,
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			logs := []string{fmt.Sprintf("execute %s -> patch committed.", id)}

			if amount := scaledCount(opts.Damage, replay); amount > 0 {
				done := b.damageEnemy(amount, opts.IgnoreArmor)
				logs = append(logs, fmt.Sprintf("integrity damage %d applied.", done))
			}
			if delta := scaledSigned(opts.AttackDelta, replay); delta != 0 {
				applyEnemyDelta(b, 1, delta)
				logs = append(logs, fmt.Sprintf("enemy attack %+d.", delta))
			}
			if delta := scaledSigned(opts.ArmorDelta, replay); delta != 0 {
				applyEnemyDelta(b, 2, delta)
				logs = append(logs, fmt.Sprintf("enemy armor %+d.", delta))
			}
			if delta := scaledSigned(opts.CountdownDelta, replay); delta != 0 {
				applyEnemyDelta(b, 3, delta)
				logs = append(logs, fmt.Sprintf("enemy countdown %+d.", delta))
			}
			if delta := scaledSigned(opts.ScriptDelta, replay); delta != 0 {
				applyEnemyDelta(b, 4, delta)
				logs = append(logs, fmt.Sprintf("enemy script %+d.", delta))
			}
			if delta := scaledSigned(opts.EntropyDelta, replay); delta != 0 {
				applyEnemyDelta(b, 5, delta)
				logs = append(logs, fmt.Sprintf("enemy entropy %+d.", delta))
			}
			if draws := scaledCount(opts.Draw, replay); draws > 0 {
				b.draw(draws)
				logs = append(logs, fmt.Sprintf("drew %d card(s).", draws))
			}
			if heal := scaledCount(opts.Heal, replay); heal > 0 {
				b.game.player.HP = min(b.game.player.MaxHP, b.game.player.HP+heal)
				logs = append(logs, fmt.Sprintf("operator restored %d HP.", heal))
			}
			if gain := scaledCount(opts.MPGain, replay); gain > 0 {
				b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+gain)
				logs = append(logs, fmt.Sprintf("operator recovered %d MP.", gain))
			}
			if delta := scaledSigned(opts.LoadDelta, replay); delta != 0 {
				b.game.player.Load = max(0, b.game.player.Load+delta)
				logs = append(logs, fmt.Sprintf("system load %+d.", delta))
			}
			if delta := scaledSigned(opts.PendingLoadDelta, replay); delta != 0 {
				b.game.player.PendingLoad = max(0, b.game.player.PendingLoad+delta)
				logs = append(logs, fmt.Sprintf("pending load %+d.", delta))
			}
			if opts.PurgeGarbage {
				removed, ok := b.purgeOneGarbage()
				if ok {
					logs = append(logs, fmt.Sprintf("purged %s.", removed))
				} else {
					logs = append(logs, "no garbage card was available to purge.")
				}
			}
			if !replay && opts.PointerGain > 0 {
				b.game.player.PointerCharges += opts.PointerGain
				logs = append(logs, fmt.Sprintf("pointer charges +%d.", opts.PointerGain))
			}
			if !replay && opts.InjectCard != "" {
				b.injectCard(opts.InjectCard)
				logs = append(logs, fmt.Sprintf("injected %s into the deck flow.", opts.InjectCard))
			}

			b.normalizeEnemy()
			return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
		},
	}
}

func makeFFTScanCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		logs := []string{fmt.Sprintf("execute %s -> spectral scan online.", id)}
		if b.enemyArraySorted() {
			damage := scaledCount(7, replay)
			done := b.damageEnemy(damage, true)
			logs = append(logs, fmt.Sprintf("ordered waveform located, dealt %d direct damage.", done))
		} else {
			drop := scaledCount(3, replay)
			applyEnemyDelta(b, 5, -drop)
			b.draw(1)
			logs = append(logs, fmt.Sprintf("waveform still noisy, entropy -%d and drew 1 card.", drop))
		}
		b.normalizeEnemy()
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeLinkedSpliceCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		mutable := b.mutableBits()
		for i := 0; i+1 < len(mutable); i += 2 {
			mutable[i], mutable[i+1] = mutable[i+1], mutable[i]
		}
		b.setMutableBits(mutable)
		done := b.damageEnemy(scaledCount(3, replay), true)
		b.normalizeEnemy()
		logs := []string{
			fmt.Sprintf("execute %s -> rewired adjacent memory links.", id),
			fmt.Sprintf("rewrite damage %d applied.", done),
		}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeSegmentTreeCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		bestIndex := -1
		bestValue := -1
		for _, index := range []int{1, 2, 4} {
			if index < len(b.enemy.Bits) && b.enemy.Bits[index] > bestValue {
				bestIndex = index
				bestValue = b.enemy.Bits[index]
			}
		}
		if bestIndex == -1 {
			return CommandResult{Logs: []string{fmt.Sprintf("execute %s -> no tactical bit found.", id)}, Animation: [][]int{before}}
		}
		cut := max(2, bestValue/3)
		if replay {
			cut = max(1, cut-1)
		}
		b.enemy.Bits[bestIndex] = max(0, b.enemy.Bits[bestIndex]-cut)
		done := b.damageEnemy(scaledCount(2, replay), true)
		b.normalizeEnemy()
		logs := []string{
			fmt.Sprintf("execute %s -> segmented %s by %d.", id, enemyStatLabel(bestIndex), cut),
			fmt.Sprintf("direct damage %d applied.", done),
		}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeTopoSortCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		mutable := b.mutableBits()
		sort.Ints(mutable)
		b.setMutableBits(mutable)
		applyEnemyDelta(b, 3, scaledCount(1, replay))
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> dependency graph normalized.", id)}
		logs = append(logs, b.applySortCountermeasure(string(id))...)
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeDijkstraRouteCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		bestIndex := -1
		bestValue := 0
		for _, index := range []int{1, 2, 4, 5} {
			if index >= len(b.enemy.Bits) {
				continue
			}
			value := b.enemy.Bits[index]
			if value <= 0 {
				continue
			}
			if bestIndex == -1 || value < bestValue {
				bestIndex = index
				bestValue = value
			}
		}
		if bestIndex == -1 {
			b.draw(1)
			return CommandResult{Logs: []string{fmt.Sprintf("execute %s -> no weak node found, drew 1 card.", id)}, Animation: [][]int{before}}
		}
		damage := max(3, bestValue+1)
		if replay {
			damage = max(2, damage-1)
		}
		done := b.damageEnemy(damage, true)
		applyEnemyDelta(b, bestIndex, -1)
		applyEnemyDelta(b, 3, 1)
		b.normalizeEnemy()
		logs := []string{
			fmt.Sprintf("execute %s -> shortest path pierced %s.", id, enemyStatLabel(bestIndex)),
			fmt.Sprintf("direct damage %d applied.", done),
		}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeRollingHashCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		hash := 0
		for _, value := range b.mutableBits() {
			hash = (hash*33 + value + 7) % 32
		}
		logs := []string{fmt.Sprintf("execute %s -> rolling hash = %d.", id, hash)}
		if hash%2 == 0 {
			done := b.damageEnemy(scaledCount(4, replay), true)
			b.draw(1)
			logs = append(logs, fmt.Sprintf("even hash confirmed, dealt %d direct damage and drew 1 card.", done))
		} else {
			applyEnemyDelta(b, 2, -scaledCount(2, replay))
			applyEnemyDelta(b, 5, -scaledCount(2, replay))
			logs = append(logs, "odd hash detected, reduced armor and entropy.")
		}
		b.normalizeEnemy()
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeLruCacheCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		logs := []string{fmt.Sprintf("execute %s -> cache lookup started.", id)}
		bestIndex := -1
		bestCost := 1 << 30
		for i := len(b.state.Discard) - 1; i >= 0; i-- {
			card := b.state.Discard[i]
			if isGarbageCard(card) {
				continue
			}
			cost := allCards[card].Cost
			if cost < bestCost {
				bestCost = cost
				bestIndex = i
			}
		}
		if bestIndex >= 0 {
			card := b.state.Discard[bestIndex]
			b.state.Discard = append(b.state.Discard[:bestIndex], b.state.Discard[bestIndex+1:]...)
			addCardToHandOrTop(b, card)
			logs = append(logs, fmt.Sprintf("recovered %s from discard.", card))
		} else {
			b.draw(1)
			logs = append(logs, "discard cache empty, drew 1 card instead.")
		}
		gain := scaledCount(1, replay)
		b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+gain)
		logs = append(logs, fmt.Sprintf("operator recovered %d MP.", gain))
		return CommandResult{Logs: logs, Animation: [][]int{cloneInts(b.enemy.Bits)}}
	}}
}

func makeBitshiftGateCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		for _, index := range []int{1, 4} {
			if index < len(b.enemy.Bits) {
				b.enemy.Bits[index] /= 2
			}
		}
		applyEnemyDelta(b, 5, -1)
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> arithmetic right shift complete.", id)}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeBranchPredictorCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		logs := []string{fmt.Sprintf("execute %s -> branch predictor trained.", id)}
		if b.lastCard != "" && !isGarbageCard(b.lastCard) {
			addCardToHandOrTop(b, b.lastCard)
			logs = append(logs, fmt.Sprintf("cached a copy of %s.", b.lastCard))
		} else {
			logs = append(logs, "no previous script to clone.")
		}
		b.draw(1)
		logs = append(logs, "drew 1 card.")
		return CommandResult{Logs: logs, Animation: [][]int{cloneInts(b.enemy.Bits)}}
	}}
}

func makeTrieProbeCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		seen := map[int]bool{}
		duplicate := false
		for _, value := range b.mutableBits() {
			if seen[value] {
				duplicate = true
				break
			}
			seen[value] = true
		}
		logs := []string{fmt.Sprintf("execute %s -> prefix scan complete.", id)}
		if duplicate {
			done := b.damageEnemy(scaledCount(5, replay), true)
			applyEnemyDelta(b, 4, -1)
			logs = append(logs, fmt.Sprintf("duplicate prefix found, dealt %d direct damage.", done))
		} else {
			b.draw(1)
			logs = append(logs, "no duplicate prefix found, drew 1 card.")
		}
		b.normalizeEnemy()
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeCycleDetectCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		logs := []string{fmt.Sprintf("execute %s -> graph walk complete.", id)}
		if b.enemyArraySorted() {
			draws := scaledCount(2, replay)
			b.draw(draws)
			applyEnemyDelta(b, 5, -1)
			logs = append(logs, fmt.Sprintf("no cycle detected, drew %d card(s).", draws))
		} else {
			applyEnemyDelta(b, 1, -scaledCount(3, replay))
			applyEnemyDelta(b, 3, 1)
			logs = append(logs, "cycle detected, reduced attack and delayed the enemy.")
		}
		b.normalizeEnemy()
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeDeadlockBreakerCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		cut := b.enemyAttack()
		if replay {
			cut = min(cut, 4)
		}
		if len(b.enemy.Bits) > 1 {
			b.enemy.Bits[1] = max(0, b.enemy.Bits[1]-cut)
		}
		applyEnemyDelta(b, 3, scaledCount(2, replay))
		applyEnemyDelta(b, 4, -scaledCount(2, replay))
		b.normalizeEnemy()
		logs := []string{
			fmt.Sprintf("execute %s -> forced unlock succeeded.", id),
			fmt.Sprintf("enemy attack reduced by %d.", cut),
		}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeVectorizeCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		odd := 0
		for _, value := range b.mutableBits() {
			if value%2 != 0 {
				odd++
			}
		}
		damage := max(3, min(7, odd))
		if replay {
			damage = max(2, damage-1)
		}
		done := b.damageEnemy(damage, false)
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> SIMD lanes active, dealt %d damage.", id, done)}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeParityFlipCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		mutable := b.mutableBits()
		for i := range mutable {
			mutable[i] ^= 1
		}
		b.setMutableBits(mutable)
		applyEnemyDelta(b, 2, -1)
		b.draw(1)
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> parity flipped across mutable bits.", id)}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeShardBalanceCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		indices := []int{1, 2, 4}
		sum := 0
		count := 0
		for _, index := range indices {
			if index >= len(b.enemy.Bits) {
				continue
			}
			sum += b.enemy.Bits[index]
			count++
		}
		if count == 0 {
			return CommandResult{Logs: []string{fmt.Sprintf("execute %s -> no shard to rebalance.", id)}, Animation: [][]int{before}}
		}
		avg := sum / count
		for _, index := range indices {
			if index < len(b.enemy.Bits) {
				b.enemy.Bits[index] = avg
			}
		}
		applyEnemyDelta(b, 3, 1)
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> tactical shards balanced around %d.", id, avg)}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func makeTailCallCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		logs := []string{fmt.Sprintf("execute %s -> tail call optimized.", id)}
		if b.lastCard != "" && !isGarbageCard(b.lastCard) {
			b.state.DrawPile = append([]CardID{b.lastCard}, b.state.DrawPile...)
			logs = append(logs, fmt.Sprintf("queued %s on top of the draw pile.", b.lastCard))
		} else {
			logs = append(logs, "no previous script to queue.")
		}
		b.draw(1)
		logs = append(logs, "drew 1 card.")
		return CommandResult{Logs: logs, Animation: [][]int{cloneInts(b.enemy.Bits)}}
	}}
}

func makeDiffFuzzerCard(id CardID, cost int, complexity string) CardDef {
	return CardDef{ID: id, Name: string(id), Cost: cost, Complexity: complexity, Play: func(b *Battle, replay bool) CommandResult {
		before := cloneInts(b.enemy.Bits)
		mutable := b.mutableBits()
		shuffleInts(b.game.rng, mutable)
		b.setMutableBits(mutable)
		done := b.damageEnemy(scaledCount(4, replay), false)
		b.normalizeEnemy()
		logs := []string{fmt.Sprintf("execute %s -> randomized diff payload landed for %d damage.", id, done)}
		return CommandResult{Logs: logs, Animation: animationFrom(before, b.enemy.Bits)}
	}}
}

func scaledCount(value int, replay bool) int {
	if value <= 0 {
		return 0
	}
	if !replay {
		return value
	}
	if value == 1 {
		return 1
	}
	return value - 1
}

func scaledSigned(value int, replay bool) int {
	if !replay || value == 0 {
		return value
	}
	switch {
	case value > 1:
		return value - 1
	case value < -1:
		return value + 1
	default:
		return value
	}
}

func applyEnemyDelta(b *Battle, index int, delta int) {
	if delta == 0 || index < 0 || index >= len(b.enemy.Bits) {
		return
	}
	b.enemy.Bits[index] += delta
}

func addCardToHandOrTop(b *Battle, card CardID) {
	if len(b.state.Hand) < 7 {
		b.state.Hand = append(b.state.Hand, card)
		return
	}
	b.state.DrawPile = append([]CardID{card}, b.state.DrawPile...)
}

func animationFrom(before []int, after []int) [][]int {
	if equalIntSlices(before, after) {
		return [][]int{cloneInts(before)}
	}
	return [][]int{cloneInts(before), cloneInts(after)}
}

func equalIntSlices(left []int, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func enemyStatLabel(index int) string {
	switch index {
	case 1:
		return "attack"
	case 2:
		return "armor"
	case 3:
		return "countdown"
	case 4:
		return "script"
	case 5:
		return "entropy"
	default:
		return fmt.Sprintf("bit[%d]", index)
	}
}

func shuffleInts(rng interface{ Intn(int) int }, values []int) {
	for i := len(values) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		values[i], values[j] = values[j], values[i]
	}
}
