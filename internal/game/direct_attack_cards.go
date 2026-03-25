package game

func init() {
	registerDirectAttackCards()
}

func registerDirectAttackCards() {
	for id, def := range directAttackCards() {
		allCards[id] = def
	}
}

func directAttackCards() map[CardID]CardDef {
	return map[CardID]CardDef{
		CardID("ping_break.c"):       makeStatScript(CardID("ping_break.c"), 1, "O(1)", statScriptOptions{Damage: 3}),
		CardID("packet_burst.lua"):   makeStatScript(CardID("packet_burst.lua"), 2, "O(n)", statScriptOptions{Damage: 5}),
		CardID("breach_charge.rs"):   makeStatScript(CardID("breach_charge.rs"), 2, "O(1)", statScriptOptions{Damage: 4, ArmorDelta: -1}),
		CardID("railgun.asm"):        makeStatScript(CardID("railgun.asm"), 3, "O(1)", statScriptOptions{Damage: 6, IgnoreArmor: true}),
		CardID("recursion_spike.go"): makeStatScript(CardID("recursion_spike.go"), 4, "O(n log n)", statScriptOptions{Damage: 8, IgnoreArmor: true}),
		CardID("kernel_panic.sys"):   makeStatScript(CardID("kernel_panic.sys"), 5, "O(n^2)", statScriptOptions{Damage: 10, IgnoreArmor: true, InjectCard: CardOverflow}),
	}
}
