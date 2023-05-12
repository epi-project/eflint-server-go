package eflint

func DeriveFacts() {
	changed := true

	for changed {
		changed = deriveFactsOnce()
	}
}

func deriveFactsOnce() bool {
	changed := false

	for _, fact := range globalState["facts"] {
		changed = deriveFact(fact) || changed
	}

	return changed
}

func deriveFact(fact interface{}) bool {
	changed := false

	if afact, ok := fact.(AtomicFact); ok {
		for _, derivation := range afact.DerivedFrom {
			//log.Println("Deriving fact", afact.Name, "from", derivation)
			for instance := range handleExpression(derivation) {
				//log.Println("Deriving fact", afact.Name, "from", derivation, "with instance", instance)
				err := handleCreate(Expression{
					Identifier: afact.Name,
					Operands: []Expression{
						instance,
					},
				})
				//if err != nil {
				//	log.Println(err)
				//}
				changed = changed || (err == nil)
			}
		}
	} else if _, ok := fact.(CompositeFact); ok {
		return false
	} else {
		panic("Unknown fact type")
	}

	return changed
}
