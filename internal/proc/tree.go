package proc

import "sort"

// Node is a process with its direct children.
type Node struct {
	Info Info
	Kids []*Node
}

// Roots finds processes whose executable matches a known agent binary and
// nests each one together with its full descendant process tree (its session).
// selfPID is excluded (the rove process itself); the transient `ps` helper is
// excluded too.
func Roots(infos []Info, known map[string]bool, selfPID int) []*Node {
	nodes := map[int]*Node{}
	for i := range infos {
		in := infos[i]
		if in.PID == selfPID || in.PID <= 0 || BinOf(in) == "ps" {
			continue
		}
		if prev, ok := nodes[in.PID]; ok {
			// macOS ps can emit the same PID twice mid-snapshot (command
			// line changed). Prefer a row that matches a known agent.
			if known[BinOf(in)] && !known[BinOf(prev.Info)] {
				nodes[in.PID] = &Node{Info: in}
			}
			continue
		}
		nodes[in.PID] = &Node{Info: in}
	}

	byParent := map[int][]*Node{}
	for _, n := range nodes {
		if n.Info.PPID == n.Info.PID {
			continue
		}
		byParent[n.Info.PPID] = append(byParent[n.Info.PPID], n)
	}
	for _, n := range nodes {
		n.Kids = byParent[n.Info.PID]
	}

	var roots []*Node
	claimed := map[int]bool{}
	for _, pid := range sortedKeys(nodes) {
		n := nodes[pid]
		if !known[BinOf(n.Info)] {
			continue
		}
		if claimed[pid] {
			continue
		}
		roots = append(roots, n)
		claim(n, claimed)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Info.PID < roots[j].Info.PID })
	return roots
}

func claim(n *Node, claimed map[int]bool) {
	if claimed[n.Info.PID] {
		return
	}
	claimed[n.Info.PID] = true
	for _, k := range n.Kids {
		claim(k, claimed)
	}
}

func sortedKeys(m map[int]*Node) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// SessionCPU sums CPU% across a root and all its descendants.
func SessionCPU(n *Node) float64 {
	return session(n, true)
}

// SessionMem sums MEM% across a root and all its descendants.
func SessionMem(n *Node) float64 {
	return session(n, false)
}

func session(n *Node, cpu bool) float64 {
	seen := map[int]bool{}
	var walk func(*Node) float64
	walk = func(x *Node) float64 {
		if seen[x.Info.PID] {
			return 0
		}
		seen[x.Info.PID] = true
		v := x.Info.CPU
		if !cpu {
			v = x.Info.Mem
		}
		for _, k := range x.Kids {
			v += walk(k)
		}
		return v
	}
	return walk(n)
}
