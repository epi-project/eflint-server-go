package eflint

import "log"

func DeriveFacts() {
	changed := true

	for changed {
		changed = deriveFactsOnce()
	}

	DerivePredicates()
}

// TODO: You want to be able to query predicates, this is not possible right now.

func DerivePredicates() {
	for _, predicate := range globalState["predicates"] {
		predicate, ok := predicate.(Predicate)
		if !ok {
			panic("Predicate is not a predicate")
		}

		expr := <-handleExpression(predicate.Expression)
		eval, err := evaluateInstance(expr)
		if err != nil {
			panic(err)
		}

		if eval != predicate.Status {
			if predicate.Status {
				log.Println("~", predicate.Name, "()")
			} else {
				log.Println("+", predicate.Name, "()")
			}

			predicate.Status = eval
		}
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

	// TODO: Clean this up.
	if afact, ok := fact.(AtomicFact); ok {
		rules := make([]Expression, 0, len(afact.DerivedFrom)+len(afact.HoldsWhen))

		if len(afact.ConditionedBy) > 0 {
			for _, derived := range afact.DerivedFrom {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: append([]Expression{derived}, afact.ConditionedBy...),
				})
			}

			for _, holds := range afact.HoldsWhen {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: append([]Expression{holds, {Value: []string{afact.Name}}}, afact.ConditionedBy...),
				})
			}
		} else {
			rules = append(rules, afact.DerivedFrom...)

			for _, holds := range afact.HoldsWhen {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: []Expression{
						{Value: []string{afact.Name}},
						holds,
					},
				})
			}
		}

		for _, rule := range rules {
			for instance := range handleExpression(rule) {
				err := handleCreate(instance)
				//if err != nil {
				//	log.Println(err)
				//}
				changed = changed || (err == nil)
			}
		}
	} else if cfact, ok := fact.(CompositeFact); ok {
		rules := make([]Expression, 0, len(cfact.DerivedFrom)+len(cfact.HoldsWhen))

		if len(cfact.ConditionedBy) > 0 {
			for _, derived := range cfact.DerivedFrom {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: append([]Expression{derived}, cfact.ConditionedBy...),
				})
			}

			for _, holds := range cfact.HoldsWhen {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: append([]Expression{holds, {Value: []string{cfact.Name}}}, cfact.ConditionedBy...),
				})
			}
		} else {
			rules = append(rules, cfact.DerivedFrom...)

			for _, holds := range cfact.HoldsWhen {
				rules = append(rules, Expression{
					Operator: "WHEN",
					Operands: []Expression{
						{Value: []string{cfact.Name}},
						holds,
					},
				})
			}
		}

		for _, rule := range rules {
			for instance := range handleExpression(rule) {
				err := handleCreate(instance)
				//if err != nil {
				//	log.Println(err)
				//}
				changed = changed || (err == nil)
			}
		}
	} else {
		panic("Unknown fact type")
	}

	return changed
}
