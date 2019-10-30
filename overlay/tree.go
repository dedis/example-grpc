package overlay

func (t Tree) ChildrenOf(pos int) []int {
	nodes := make([]int, t.GetN())
	for i := range nodes {
		if i == 0 {
			nodes[i] = int(t.GetRoot())
		} else if i < int(t.GetRoot()) {
			nodes[i] = i - 1
		} else {
			nodes[i] = i
		}
	}

	for i, j := range nodes[:len(nodes)-1] {
		if j == pos {
			firstIndex := int(t.GetK())*i + 1
			if firstIndex >= len(nodes) {
				return []int{}
			}
			lastIndex := firstIndex + int(t.GetK())
			if lastIndex >= len(nodes) {
				lastIndex = firstIndex + len(nodes) - firstIndex
			}

			return nodes[firstIndex:lastIndex]
		}
	}

	return []int{}
}
