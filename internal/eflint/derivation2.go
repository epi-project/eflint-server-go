package eflint

import (
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func findReferences(expr Expression) []string {
	references := make([]string, 0, len(expr.Operands))

	if expr.Identifier != "" {
		references = append(references, expr.Identifier)
	}

	for _, operand := range expr.Operands {
		references = append(references, findReferences(operand)...)
	}

	if expr.Expression != nil {
		references = append(references, findReferences(*expr.Expression)...)
	}

	if expr.Operand != nil {
		references = append(references, findReferences(*expr.Operand)...)
	}

	return references
}

func DeriveFacts2() {
	dependencies := make(map[string]map[string]struct{})

	for _, fact := range globalState["facts"] {
		name, rules := generateDerivationRules(fact)
		dependencies[name] = make(map[string]struct{})

		for _, rule := range rules {
			for _, reference := range findReferences(rule) {
				// Whenever the reference changes, the fact needs to be re-derived.
				if _, ok := dependencies[reference]; !ok {
					dependencies[reference] = make(map[string]struct{})
				}

				dependencies[reference][name] = struct{}{}
			}
		}
	}

	queue := make([]string, len(dependencies))
	i := 0

	for name := range dependencies {
		queue[i] = name
		i++
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if !deriveFact2(globalState["facts"][name]) {
			continue
		}

		for dependency := range dependencies[name] {
			queue = append(queue, dependency)
		}
	}

	CheckViolations()
}

func deriveFact2(fact interface{}) bool {
	name, rules := generateDerivationRules(fact)
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
