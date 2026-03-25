package game

import (
	"fmt"
	"sort"
)

var allCards = map[CardID]CardDef{
	CardReverse: {
		ID:         CardReverse,
		Name:       "reverse_list.py",
		Cost:       2,
		Complexity: "O(n)",
		Text:       "翻转敌方可移动内存区。若敌人锁定了前缀位，这些位不会被翻动。",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			start := b.lockedPrefix()
			for left, right := start, len(b.enemy.Bits)-1; left < right; left, right = left+1, right-1 {
				b.enemy.Bits[left], b.enemy.Bits[right] = b.enemy.Bits[right], b.enemy.Bits[left]
			}
			b.normalizeEnemy()
			logs := []string{
				"执行 reverse_list.py -> 可移动内存区已翻转。",
				fmt.Sprintf("瀹堟姢绋嬪簭褰撳墠瑙ｇ爜涓?%v", b.enemy.Bits),
			}
			if start > 0 {
				logs = append(logs, fmt.Sprintf("前 %d 个 bit 已锁定，未被翻转。", start))
			}
			return CommandResult{
				Logs:      logs,
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardBubble: {
		ID:         CardBubble,
		Name:       "bubble_sort.sh",
		Cost:       4,
		Complexity: "O(n^2)",
		Text:       "????? 4 ???? bit ????????????????? 2 ???",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			bits := cloneInts(b.enemy.Bits)
			frames := [][]int{cloneInts(bits)}
			start := b.lockedPrefix()
			window := min(4, len(bits)-start)
			for i := 0; i < window; i++ {
				swapped := false
				for j := start; j < start+window-1-i; j++ {
					if bits[j] > bits[j+1] {
						bits[j], bits[j+1] = bits[j+1], bits[j]
						frames = append(frames, cloneInts(bits))
						swapped = true
					}
				}
				if !swapped {
					break
				}
			}
			b.enemy.Bits = bits
			if len(b.enemy.Bits) > 3 {
				b.enemy.Bits[3] += 2
			}
			b.normalizeEnemy()
			logs := []string{
				"执行 bubble_sort.sh -> 只整理了前 4 个可移动 bit。",
				fmt.Sprintf("敌方倒计时已延后至 %d。", b.enemyCountdown()),
			}
			logs = append(logs, b.applySortCountermeasure("bubble_sort.sh")...)
			return CommandResult{Logs: logs, Animation: frames}
		},
	},
	CardBinary: {
		ID:         CardBinary,
		Name:       "binary_search.go",
		Cost:       2,
		Complexity: "O(log n)",
		Text:       "若敌方可移动 bits 已有序，则清空当前攻击签名；否则抽 1 张牌。",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			if b.enemyArraySorted() {
				if b.lockedPrefix() <= 1 && len(b.enemy.Bits) > 1 {
					target := b.enemy.Bits[1]
					b.enemy.Bits[1] = 0
					if replay && len(b.enemy.Bits) > 2 {
						b.enemy.Bits[2] = max(0, b.enemy.Bits[2]-1)
					}
					b.normalizeEnemy()
					return CommandResult{
						Logs: []string{
							fmt.Sprintf("执行 binary_search.go -> 攻击签名 %d 已被清空。", target),
							fmt.Sprintf("瀹堟姢绋嬪簭褰撳墠瑙ｇ爜涓?%v", b.enemy.Bits),
						},
						Animation: [][]int{before, cloneInts(b.enemy.Bits)},
					}
				}

				start := b.lockedPrefix()
				mutable := cloneInts(b.enemy.Bits[start:])
				target := b.enemyAttack()
				index := sort.SearchInts(mutable, target)
				if index < len(mutable) && mutable[index] == target {
					mutable[index] = 0
					b.setMutableBits(mutable)
					b.normalizeEnemy()
					return CommandResult{
						Logs: []string{
							fmt.Sprintf("执行 binary_search.go -> 在可移动区命中攻击签名 %d。", target),
						},
						Animation: [][]int{before, cloneInts(b.enemy.Bits)},
					}
				}
			}

			b.draw(1)
			logs := []string{
				"执行 binary_search.go -> 目标区间失序，改为补充手牌。",
				fmt.Sprintf("当前手牌数为 %d。", len(b.state.Hand)),
			}
			if b.enemy.UnsortedAttackGain > 0 && !b.enemyArraySorted() {
				logs = append(logs, fmt.Sprintf("敌方被动生效：失序状态下攻击额外 +%d。", b.enemy.UnsortedAttackGain))
			}
			return CommandResult{Logs: logs, Animation: [][]int{before}}
		},
	},
	CardQuick: {
		ID:         CardQuick,
		Name:       "quick_sort.rs",
		Cost:       5,
		Complexity: "O(n log n)",
		Text:       "???????????????????????????? 1 ???",
		Play: func(b *Battle, replay bool) CommandResult {
			bits := cloneInts(b.enemy.Bits)
			frames := [][]int{cloneInts(bits)}
			start := b.lockedPrefix()
			var quick func(int, int)
			quick = func(low, high int) {
				if low >= high {
					return
				}
				pivot := bits[high]
				i := low
				for j := low; j < high; j++ {
					if bits[j] <= pivot {
						bits[i], bits[j] = bits[j], bits[i]
						frames = append(frames, cloneInts(bits))
						i++
					}
				}
				bits[i], bits[high] = bits[high], bits[i]
				frames = append(frames, cloneInts(bits))
				quick(low, i-1)
				quick(i+1, high)
			}
			if start < len(bits) {
				quick(start, len(bits)-1)
			}
			b.enemy.Bits = bits
			if len(b.enemy.Bits) > 3 {
				b.enemy.Bits[3]++
			}
			b.normalizeEnemy()
			logs := []string{
				"执行 quick_sort.rs -> 可移动区已完成快速排序。",
				fmt.Sprintf("敌方倒计时已延后至 %d。", b.enemyCountdown()),
			}
			logs = append(logs, b.applySortCountermeasure("quick_sort.rs")...)
			return CommandResult{Logs: logs, Animation: frames}
		},
	},
	CardBacktrack: {
		ID:         CardBacktrack,
		Name:       "backtrack.ts",
		Cost:       3,
		Complexity: "O(branch^depth)",
		Text:       "???????????????????????MP?????? bits????????????????",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			if !b.restoreSystemSnapshot() {
				return CommandResult{Err: fmt.Errorf("褰撳墠娌℃湁鍙仮澶嶇殑蹇収")}
			}
			return CommandResult{
				Logs: []string{
					"执行 backtrack.ts -> 快照恢复完成。",
					fmt.Sprintf("内存已回滚至第 %d 回合。", b.turn),
				},
				Animation: [][]int{cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardHeap: {
		ID:         CardHeap,
		Name:       "heap_alloc.cpp",
		Cost:       1,
		Complexity: "O(1)",
		Text:       "??????? 2 ? MP??? 1 ???",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			recover := 2
			b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+recover)
			b.draw(1)
			return CommandResult{
				Logs: []string{
					fmt.Sprintf("?? heap_alloc.cpp -> ?? %d ? MP?", recover),
					fmt.Sprintf("?? MP ? %d?", b.game.player.MP),
				},
			}
		},
	},
	CardNullPointer: {
		ID:         CardNullPointer,
		Name:       "null_pointer.exe",
		Cost:       2,
		Complexity: "O(1)",
		Text:       "若敌方仍有护甲则将其归零，否则直接造成 5 点完整度伤害。",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			if b.enemyArmor() > 0 {
				b.enemy.Bits[2] = 0
				b.normalizeEnemy()
				return CommandResult{
					Logs: []string{
						"执行 null_pointer.exe -> 护甲寄存器已清空。",
						"护甲被改写为 0。",
					},
					Animation: [][]int{before, cloneInts(b.enemy.Bits)},
				}
			}
			damage := 5
			if replay {
				damage = 3
			}
			done := b.damageEnemy(damage, true)
			return CommandResult{
				Logs: []string{
					fmt.Sprintf("执行 null_pointer.exe -> 直接造成 %d 点完整度伤害。", done),
				},
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardMerge: {
		ID:         CardMerge,
		Name:       "merge_patch.diff",
		Cost:       3,
		Complexity: "O(n)",
		Text:       "交错合并敌方可移动内存区，并直接造成 3 点重写伤害。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			start := b.lockedPrefix()
			mutable := cloneInts(b.enemy.Bits[start:])
			left := cloneInts(mutable[:len(mutable)/2])
			right := cloneInts(mutable[len(mutable)/2:])
			merged := make([]int, 0, len(mutable))
			for len(left) > 0 || len(right) > 0 {
				if len(left) > 0 {
					merged = append(merged, left[0])
					left = left[1:]
				}
				if len(right) > 0 {
					merged = append(merged, right[0])
					right = right[1:]
				}
			}
			b.setMutableBits(merged)
			damage := 3
			if replay {
				damage = 2
			}
			done := b.damageEnemy(damage, true)
			b.normalizeEnemy()
			return CommandResult{
				Logs: []string{
					"执行 merge_patch.diff -> 可移动内存区已交错合并。",
					fmt.Sprintf("直接改写了 %d 点完整度。", done),
				},
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardRadix: {
		ID:         CardRadix,
		Name:       "radix_sort.java",
		Cost:       4,
		Complexity: "O(n*k)",
		Text:       "用稳定的基数排序整理敌方可移动区，降低 3 点熵值并抽 1 张牌。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			start := b.lockedPrefix()
			bits := cloneInts(b.enemy.Bits)
			frames := [][]int{before}
			if start >= len(bits) {
				return CommandResult{
					Logs:      []string{"执行 radix_sort.java -> 敌方当前没有可整理的可移动区。"},
					Animation: frames,
				}
			}
			mutable := cloneInts(bits[start:])
			maxValue := 0
			for _, value := range mutable {
				if value > maxValue {
					maxValue = value
				}
			}
			for exp := 1; maxValue/exp > 0; exp *= 10 {
				count := make([]int, 10)
				output := make([]int, len(mutable))
				for _, value := range mutable {
					count[(value/exp)%10]++
				}
				for i := 1; i < len(count); i++ {
					count[i] += count[i-1]
				}
				for i := len(mutable) - 1; i >= 0; i-- {
					digit := (mutable[i] / exp) % 10
					output[count[digit]-1] = mutable[i]
					count[digit]--
				}
				copy(mutable, output)
				copy(bits[start:], mutable)
				frames = append(frames, cloneInts(bits))
			}
			b.enemy.Bits = bits
			if len(b.enemy.Bits) > 5 {
				b.enemy.Bits[5] = max(0, b.enemy.Bits[5]-3)
			}
			b.draw(1)
			b.normalizeEnemy()
			logs := []string{
				"执行 radix_sort.java -> 稳定排序完成。",
				fmt.Sprintf("敌方熵值降至 %d。", b.enemyEntropy()),
			}
			logs = append(logs, b.applySortCountermeasure("radix_sort.java")...)
			return CommandResult{Logs: logs, Animation: frames}
		},
	},
	CardXor: {
		ID:         CardXor,
		Name:       "xor_mask.asm",
		Cost:       2,
		Complexity: "O(n)",
		Text:       "用掩码 3 对敌方攻击、护甲、脚本与熵值做 XOR，打乱战术槽位。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			for _, index := range []int{1, 2, 4, 5} {
				if index < len(b.enemy.Bits) && index >= b.lockedPrefix() {
					b.enemy.Bits[index] ^= 3
				}
			}
			if replay && len(b.enemy.Bits) > 3 {
				b.enemy.Bits[3]++
			}
			b.normalizeEnemy()
			return CommandResult{
				Logs: []string{
					"执行 xor_mask.asm -> 战术槽位已被打乱。",
					fmt.Sprintf("鏀诲嚮 %d | 鎶ょ敳 %d | 鑴氭湰 %d", b.enemyAttack(), b.enemyArmor(), b.enemyScript()),
				},
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardMemoize: {
		ID:         CardMemoize,
		Name:       "memoize.lua",
		Cost:       1,
		Complexity: "O(1)",
		Text:       "将弃牌堆里最近一张可用脚本复制回手牌。",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			for i := len(b.state.Discard) - 1; i >= 0; i-- {
				card := b.state.Discard[i]
				if card == CardMemoize || isGarbageCard(card) {
					continue
				}
				if len(b.state.Hand) < 7 {
					b.state.Hand = append(b.state.Hand, card)
				} else {
					b.state.DrawPile = append([]CardID{card}, b.state.DrawPile...)
				}
				if replay {
					b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+1)
				}
				return CommandResult{
					Logs: []string{
						fmt.Sprintf("执行 memoize.lua -> 已缓存 %s。", string(card)),
					},
				}
			}
			b.draw(1)
			return CommandResult{
				Logs: []string{
					"执行 memoize.lua -> 没有可缓存脚本，改为抽 1 张牌。",
				},
			}
		},
	},
	CardChecksum: {
		ID:         CardChecksum,
		Name:       "checksum.py",
		Cost:       1,
		Complexity: "O(n)",
		Text:       "计算 bits 校验和。若总和为偶数则直伤 4 点，否则抽 2 张牌并写入 1 张 overflow_error.log。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			total := 0
			for _, value := range b.enemy.Bits {
				total += value
			}
			if total%2 == 0 {
				damage := 4
				if replay {
					damage = 2
				}
				done := b.damageEnemy(damage, true)
				return CommandResult{
					Logs: []string{
						fmt.Sprintf("执行 checksum.py -> 校验和为偶数，直接造成 %d 点伤害。", done),
					},
					Animation: [][]int{before, cloneInts(b.enemy.Bits)},
				}
			}
			draws := 2
			if replay {
				draws = 1
			}
			b.draw(draws)
			b.injectCard(CardOverflow)
			return CommandResult{
				Logs: []string{
					fmt.Sprintf("执行 checksum.py -> 校验和为奇数，抽了 %d 张牌。", draws),
					"副作用：写入了 1 张 overflow_error.log。",
				},
				Animation: [][]int{before},
			}
		},
	},
	CardPivot: {
		ID:         CardPivot,
		Name:       "pivot_swap.rb",
		Cost:       2,
		Complexity: "O(1)",
		Text:       "交换攻击槽与护甲槽。若敌人正在蓄力，则额外延后 1 个计时。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			if b.lockedPrefix() > 1 {
				return CommandResult{
					Logs: []string{
						"执行 pivot_swap.rb -> 攻击槽已被锁定，交换失败。",
					},
					Animation: [][]int{before},
				}
			}
			if len(b.enemy.Bits) > 2 {
				b.enemy.Bits[1], b.enemy.Bits[2] = b.enemy.Bits[2], b.enemy.Bits[1]
			}
			if b.enemyCountdown() > 0 {
				b.enemy.Bits[3]++
			}
			if replay && len(b.enemy.Bits) > 4 {
				b.enemy.Bits[4] = max(0, b.enemy.Bits[4]-1)
			}
			b.normalizeEnemy()
			return CommandResult{
				Logs: []string{
					"执行 pivot_swap.rb -> 已交换攻击与护甲槽。",
					fmt.Sprintf("鏀诲嚮 %d | 鎶ょ敳 %d", b.enemyAttack(), b.enemyArmor()),
				},
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardDefrag: {
		ID:         CardDefrag,
		Name:       "defrag_gc.cs",
		Cost:       2,
		Complexity: "O(n)",
		Text:       "稀有清理牌：清除 1 张垃圾卡，当前负载 -2，并恢复 2 点生命。",
		Play: func(b *Battle, replay bool) CommandResult {
			heal := 2
			loadDrop := 2
			if replay {
				heal = 1
				loadDrop = 1
			}
			b.game.player.HP = min(b.game.player.MaxHP, b.game.player.HP+heal)
			b.game.player.Load = max(0, b.game.player.Load-loadDrop)
			b.game.player.PendingLoad = max(0, b.game.player.PendingLoad-1)
			removed, ok := b.purgeOneGarbage()
			logs := []string{
				fmt.Sprintf("执行 defrag_gc.cs -> 恢复 %d 点生命，当前负载降低 %d。", heal, loadDrop),
			}
			if ok {
				logs = append(logs, fmt.Sprintf("已清除垃圾卡：%s。", removed))
			} else {
				logs = append(logs, "当前没有可清除的垃圾卡。")
			}
			return CommandResult{Logs: logs, Animation: [][]int{cloneInts(b.enemy.Bits)}}
		},
	},
	CardGreedy: {
		ID:         CardGreedy,
		Name:       "greedy_path.kt",
		Cost:       3,
		Complexity: "O(n)",
		Text:       "锁定敌方最大非生命槽位并抽取其数值，造成一半的直接伤害。",
		Play: func(b *Battle, replay bool) CommandResult {
			before := cloneInts(b.enemy.Bits)
			bestIndex := 1
			bestValue := 0
			for i := 1; i < len(b.enemy.Bits); i++ {
				if b.enemy.Bits[i] > bestValue {
					bestValue = b.enemy.Bits[i]
					bestIndex = i
				}
			}
			damage := max(2, bestValue/2)
			if replay {
				damage = max(1, damage-1)
			}
			done := b.damageEnemy(damage, true)
			b.enemy.Bits[bestIndex] = max(0, b.enemy.Bits[bestIndex]-2)
			b.normalizeEnemy()
			return CommandResult{
				Logs: []string{
					fmt.Sprintf("执行 greedy_path.kt -> 锁定槽位 %d，直接造成 %d 点伤害。", bestIndex, done),
				},
				Animation: [][]int{before, cloneInts(b.enemy.Bits)},
			}
		},
	},
	CardMemoryLeak: {
		ID:         CardMemoryLeak,
		Name:       "memory_leak.bin",
		Cost:       0,
		Complexity: "O(1)",
		Text:       "诅咒卡。无法打出；只要留在手里，每回合都会额外增加 1 点系统负载。",
		Unplayable: true,
		Play: func(b *Battle, replay bool) CommandResult {
			return CommandResult{Err: fmt.Errorf("memory_leak.bin 不能被直接执行")}
		},
	},
	CardOverflow: {
		ID:         CardOverflow,
		Name:       "overflow_error.log",
		Cost:       0,
		Complexity: "O(1)",
		Text:       "垃圾卡。打出后抽 1 张牌，但会额外增加 1 点待结算负载。",
		Play: func(b *Battle, replay bool) CommandResult {
			b.draw(1)
			b.game.player.PendingLoad++
			return CommandResult{
				Logs: []string{
					"执行 overflow_error.log -> 抽 1 张牌，但下回合会额外增加 1 点负载。",
				},
			}
		},
	},
	CardNoop: {
		ID:         CardNoop,
		Name:       "noop.log",
		Cost:       0,
		Complexity: "O(0)",
		Text:       "废卡，只会在日志里留下一个没有意义的 Success。",
		Starter:    true,
		Play: func(b *Battle, replay bool) CommandResult {
			return CommandResult{
				Logs: []string{
					"鎵ц noop.log -> Success",
					"什么都没有发生。某个 90 年代程序员隔着噪音在发笑。",
				},
			}
		},
	},
}

func isGarbageCard(card CardID) bool {
	switch card {
	case CardNoop, CardMemoryLeak, CardOverflow:
		return true
	default:
		return false
	}
}
