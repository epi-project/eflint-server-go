package eflint

import "log"

func DeriveFacts() {
	changed := true

	for changed {
		changed = deriveFactsOnce()
	}

	CheckViolations()

	//DerivePredicates()
}

func CheckViolations() {
	for factName, instances := range globalState["instances"] {
		fact := globalState["facts"][factName]
		if cfact, ok := fact.(CompositeFact); ok && cfact.ViolatedWhen != nil {
			for _, instance := range instances.([]interface{}) {
				clause := fillParameters(*cfact.ViolatedWhen, cfact.IdentifiedBy, instance.(Expression).Operands)
				expr, ok := <-handleExpression(clause)
				if !ok {
					panic("Could not handle expression")
				}

				eval, err := evaluateInstance(expr)
				if err != nil {
					panic(err)
				}

				if eval {
					addViolation("duty", instance.(Expression))
				}
			}
		} else if afact, ok := fact.(AtomicFact); ok && afact.IsInvariant {
			if len(instances.([]interface{})) != 1 {
				addViolation("invariant", Expression{Value: []string{factName}})
			}
		}
	}
}

func containsInstance(instances []interface{}, instance Expression) bool {
	for _, i := range instances {
		if equalInstances(i.(Expression), instance) {
			return true
		}
	}

	return false
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

	for i := range derivedFrom {
		if derivedFrom[i].Identifier != name {
			derivedFrom[i] = Expression{
				Identifier: name,
				Operands:   []Expression{derivedFrom[i]},
			}
		}
	}

	rules := make([]Expression, 0, len(derivedFrom)+len(holdsWhen))

	if len(conditionedBy) > 0 {
		for _, derived := range derivedFrom {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: append([]Expression{derived}, conditionedBy...),
			})
		}

		for _, holds := range holdsWhen {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: append([]Expression{holds, {Value: []string{name}}}, conditionedBy...),
			})
		}
	} else {
		rules = append(rules, derivedFrom...)

		for _, holds := range holdsWhen {
			rules = append(rules, Expression{
				Operator: "WHEN",
				Operands: []Expression{
					{Value: []string{name}},
					holds,
				},
			})
		}
	}

	result := make([]interface{}, 0, len(rules))

	oldDerived := make([]interface{}, 0, len(globalState["instances"][name].([]interface{})))
	oldPostulated := make([]interface{}, 0, len(globalState["instances"][name].([]interface{})))

	for _, old := range globalState["instances"][name].([]interface{}) {
		if oldExpr, ok := old.(Expression); ok {
			if oldExpr.IsDerived {
				oldDerived = append(oldDerived, oldExpr)
			} else {
				oldPostulated = append(oldPostulated, oldExpr)
			}
		}
	}

	//log.Println(globalState["instances"][name])
	//log.Println("old derived", oldDerived)
	//log.Println("old postulated", oldPostulated)

	// Only keep the postulated facts.
	globalState["instances"][name] = append([]interface{}{}, oldPostulated...)

	// Go over all the rules and derive the facts.
	for _, rule := range rules {
		// Go over all instances of the rule.
		for expr := range handleExpression(rule) {
			converted, err := convertInstance(expr)
			if err != nil {
				panic(err)
			}
			//log.Println("got +", converted)
			result = append(result, converted)
		}
	}

	//log.Println("result", result)
	//log.Println("old derived", oldDerived)
	//log.Println("old postulated", oldPostulated)

	for _, expr := range oldDerived {
		if !containsInstance(result, expr.(Expression)) {
			changed = true
			log.Println("~" + formatExpression(expr.(Expression)))
		}
	}

	// TODO: BROKEN AS FUCK
	for _, res := range result {
		if expr, ok := res.(Expression); ok {
			if !containsInstance(oldDerived, expr) {
				err := handleCreate(expr, true)
				//if err != nil {
				//	log.Println("Derivation create error:", err)
				//}
				changed = changed || err == nil
			} else {
				expr.IsDerived = true
				globalState["instances"][name] = append(globalState["instances"][name].([]interface{}), expr)
			}
		}
	}

	//log.Println("Done with deriving fact", name, globalState["instances"][name])
	return changed
}
