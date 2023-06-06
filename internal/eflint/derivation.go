package eflint

import (
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func DeriveFacts() {
	changed := true

	for changed {
		changed = deriveFactsOnce()
	}

	CheckViolations()

	//DerivePredicates()
}

func CheckViolations() {
	signal := make(chan struct{})
	defer close(signal)

	for factName, instances := range globalInstances {
		fact := globalState["facts"][factName]
		if cfact, ok := fact.(CompositeFact); ok && cfact.ViolatedWhen != nil {
			for pair := instances.Oldest(); pair != nil; pair = pair.Next() {
				clause := fillParameters(*cfact.ViolatedWhen, cfact.IdentifiedBy, pair.Value.Operands)
				expr, ok := <-handleExpression(clause, signal)
				if !ok {
					panic("Could not handle expression")
				}

				eval, err := evaluateInstance(expr)
				if err != nil {
					panic(err)
				}

				if eval {
					addViolation("duty", pair.Value)
				}
			}
		} else if afact, ok := fact.(AtomicFact); ok && afact.IsInvariant {
			if instances.Len() != 1 {
				addViolation("invariant", Expression{Value: []string{factName}})
			}
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
	var holdsWhen []Expression
	var derivedFrom []Expression
	var conditionedBy []Expression
	var name string

	if afact, ok := fact.(AtomicFact); ok {
		holdsWhen = afact.HoldsWhen
		derivedFrom = afact.DerivedFrom
		conditionedBy = afact.ConditionedBy
		name = afact.Name
	} else if cfact, ok := fact.(CompositeFact); ok {
		holdsWhen = cfact.HoldsWhen
		derivedFrom = cfact.DerivedFrom
		conditionedBy = cfact.ConditionedBy
		name = cfact.Name
	} else {
		panic("Fact is neither atomic nor composite")
	}

	rules := make([]Expression, 0, len(derivedFrom)+len(holdsWhen))
	instance := Expression{Value: []string{name}}

	if _, ok := globalState["facts"][name].(CompositeFact); ok {
		instance = Expression{Identifier: name}
	}

	if len(conditionedBy) > 0 {
		for _, derived := range derivedFrom {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: []Expression{derived, {
					Operator: "AND",
					Operands: append([]Expression{}, conditionedBy...),
				}},
			})
		}

		for _, holds := range holdsWhen {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: []Expression{instance, {
					Operator: "AND",
					Operands: append([]Expression{holds}, conditionedBy...),
				}},
			})
		}
	} else {
		rules = append(rules, derivedFrom...)

		for _, holds := range holdsWhen {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: []Expression{
					copyExpression(instance),
					holds,
				},
			})
		}
	}

	oldDerived := orderedmap.New[uint64, Expression]()

	for pair := globalInstances[name].Oldest(); pair != nil; {
		next := pair.Next()

		if pair.Value.IsDerived {
			oldDerived.Set(pair.Key, pair.Value)
			globalInstances[name].Delete(pair.Key)
		}

		pair = next
	}

	changed := true

	for changed {
		changed = false

		// Go over all the rules and derive the facts.
		for _, rule := range rules {
			// Go over all instances of the rule.
			signal := make(chan struct{}, 1)

			for expr := range handleExpression(rule, signal) {
				//log.Println("Derived", name, "with", expr)
				if expr.Identifier != name {
					expr = Expression{
						Identifier: name,
						Operands:   []Expression{expr},
					}
				}

				err := create(expr, true)

				if err != nil {
					//log.Println("Error deriving", name, "with", expr, ":", err)
					//panic(err)
				} else {
					changed = true
				}

				signal <- struct{}{}
			}

			close(signal)
		}
	}

	for pair := oldDerived.Oldest(); pair != nil; pair = pair.Next() {
		if _, ok := globalInstances[name].Get(pair.Key); !ok {
			return true
		}
	}

	for pair := globalInstances[name].Oldest(); pair != nil; pair = pair.Next() {
		//log.Printf("new: %v -> %v\n", pair.Key, pair.Value)
		if !pair.Value.IsDerived {
			continue
		}

		if _, ok := oldDerived.Get(pair.Key); !ok {
			return true
		}
	}

	return false
}
