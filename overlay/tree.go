package overlay

func (t *Tree) getChildren(parent string) []string {
	n := len(t.GetAddresses())

	for i, addr := range t.GetAddresses() {
		if parent == addr {
			firstIndex := int(t.GetK())*i + 1
			if firstIndex >= n {
				return []string{}
			}
			lastIndex := firstIndex + int(t.GetK())
			if lastIndex >= n {
				lastIndex = n
			}

			return t.GetAddresses()[firstIndex:lastIndex]
		}
	}

	return []string{}
}
