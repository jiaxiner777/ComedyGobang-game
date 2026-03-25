package game

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Randomizer interface {
	Intn(int) int
}

func NewWorld(rng Randomizer) *World {
	nodes := map[int]*Node{
		1:  newNode(1, "BOOT::Ingress", NodeStart, 0, "你的终端已经接入 The Kernel，内核树正在眼前展开。"),
		2:  newNode(2, "Trace Walker", NodeCombat, randomBalance(rng), "追踪进程会把你的每一步写进监控日志。"),
		3:  newNode(3, "Null Bazaar", NodeRest, randomBalance(rng), "黑市缓存与补丁商人愿意替你修补受损的运行时。"),
		4:  newNode(4, "Fork Bomb Warden", NodeCombat, -2, "递归复制体沿着总线蔓延，想把你的栈彻底淹没。"),
		5:  newNode(5, "Comment Archive", NodeTreasure, randomBalance(rng), "破损软盘和旧注释像遗物一样堆积成山。"),
		6:  newNode(6, "AVL Rotator", NodeCombat, randomBalance(rng), "平衡守卫不断旋转内存通道，让你难以站稳。"),
		7:  newNode(7, "Syscall Chapel", NodeRest, 2, "安静的系统调用礼拜堂里还回响着旧内核的低语。"),
		8:  newNode(8, "Heap Mirage", NodeTreasure, -2, "幻象堆池映出那些差一点就能编译成功的脚本。"),
		9:  newNode(9, "Diff Hound", NodeCombat, randomBalance(rng), "补丁猎犬会先闻出你的改动，再扑向真正的漏洞。"),
		10: newNode(10, "Modem Choir", NodeTreasure, randomBalance(rng), "古老拨号音在沉睡终端组成的唱诗班中回荡。"),
		11: newNode(11, "Red-Black Priest", NodeCombat, 2, "偏执的颜色祭司会惩罚每一条失衡的分支。"),
		12: newNode(12, "Cache Orchard", NodeRest, randomBalance(rng), "温热缓存行像果实一样挂在枝头，等待被摘取。"),
		13: newNode(13, "Dead Sector", NodeCombat, -2, "坏扇区映射里，损毁数据仍在顽强反击。"),
		14: newNode(14, "Archive Gate", NodeTreasure, randomBalance(rng), "Root 之前的最后一道档案库还保存着旧世界的注释。"),
		15: newNode(15, "ROOT::Sentinel", NodeBoss, 2, "最终守门程序只向能比它恢复得更快的人开放 Root。"),
	}

	nodes[1].Cleared = true
	return &World{
		Nodes:    nodes,
		Current:  1,
		Final:    15,
		Protocol: "preorder",
	}
}

func newNode(index int, name string, kind NodeKind, balance int, description string) *Node {
	return &Node{
		Index:       index,
		Name:        name,
		Kind:        kind,
		Balance:     balance,
		Description: description,
	}
}

func randomBalance(rng Randomizer) int {
	return rng.Intn(5) - 2
}

func (w *World) SetProtocol(mode string) {
	switch strings.ToLower(mode) {
	case "postorder":
		w.Protocol = "postorder"
	default:
		w.Protocol = "preorder"
	}
}

func (w *World) CurrentNode() *Node {
	return w.Nodes[w.Current]
}

func (w *World) NodeAt(index int) *Node {
	return w.Nodes[index]
}

func (w *World) AvailableMoves() map[string]int {
	moves := map[string]int{}
	if _, ok := w.Nodes[w.Current*2]; ok {
		moves["l"] = w.Current * 2
	}
	if _, ok := w.Nodes[w.Current*2+1]; ok {
		moves["r"] = w.Current*2 + 1
	}
	if w.Current > 1 {
		moves["b"] = w.Current / 2
	}
	return moves
}

func (w *World) Move(dir string) (*Node, error) {
	target, ok := w.AvailableMoves()[strings.ToLower(dir)]
	if !ok {
		return nil, fmt.Errorf("这个方向没有可达节点")
	}
	w.Current = target
	return w.CurrentNode(), nil
}

func (w *World) PathString() string {
	index := w.Current
	var labels []string
	for index >= 1 {
		labels = append([]string{w.Nodes[index].Name}, labels...)
		if index == 1 {
			break
		}
		index /= 2
	}
	return strings.Join(labels, " -> ")
}

func (w *World) TraversalSequence() []int {
	var order []int
	var walk func(int)
	walk = func(index int) {
		node := w.Nodes[index]
		if node == nil {
			return
		}
		left := index * 2
		right := index*2 + 1
		if w.Protocol == "postorder" {
			walk(left)
			walk(right)
			order = append(order, index)
			return
		}
		order = append(order, index)
		walk(left)
		walk(right)
	}
	walk(1)
	return order
}

func (w *World) ProtocolTarget() *Node {
	for _, index := range w.TraversalSequence() {
		node := w.Nodes[index]
		if node == nil || node.Cleared || node.Kind == NodeStart {
			continue
		}
		return node
	}
	return nil
}

func (w *World) Completion() string {
	total := 0
	cleared := 0
	for _, node := range w.Nodes {
		if node.Kind == NodeStart {
			continue
		}
		total++
		if node.Cleared {
			cleared++
		}
	}
	return fmt.Sprintf("%d/%d", cleared, total)
}

func (w *World) SortedIndexes() []int {
	indexes := make([]int, 0, len(w.Nodes))
	for index := range w.Nodes {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}

func (n *Node) Unstable() bool {
	return math.Abs(float64(n.Balance)) > 1
}
