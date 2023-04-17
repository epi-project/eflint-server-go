package eflint

func DeriveFacts() {
	changed := true

	for changed {
		changed = deriveFactsOnce()
	}
}

func deriveFactsOnce() bool {
	return false
}
