package eflint

import (
	"github.com/mitchellh/hashstructure/v2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Assumptions struct {
	Expression  uint64
	Knowledge   map[string]*orderedmap.OrderedMap[uint64, Expression]
	Assumptions map[uint64]*Assumptions
	Queue       []string
}

var tempAssumptions []*Assumptions
var globalAssumptions map[uint64]*Assumptions
var globalQueue []string

func copyAssumptions() map[uint64]*Assumptions {
	newAssumptions := make(map[uint64]*Assumptions)

	for name, assumptions := range globalAssumptions {
		newAssumptions[name] = assumptions
	}

	return newAssumptions
}

func copyQueue() []string {
	newQueue := make([]string, len(globalQueue))
	copy(newQueue, globalQueue)
	return newQueue
}

func copyKnowledge() map[string]*orderedmap.OrderedMap[uint64, Expression] {
	newKnowledge := make(map[string]*orderedmap.OrderedMap[uint64, Expression])

	for name, knowledge := range globalInstances {
		newKnowledge[name] = orderedmap.New[uint64, Expression]()
		for pair := knowledge.Oldest(); pair != nil; pair = pair.Next() {
			newKnowledge[name].Set(pair.Key, pair.Value)
		}
	}

	return newKnowledge
}

func DeriveFacts3() {
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

	globalQueue = make([]string, len(dependencies))
	i := 0

	for name := range dependencies {
		globalQueue[i] = name
		i++
	}

	globalAssumptions = make(map[uint64]*Assumptions)
	customDerivation = true

	for len(globalQueue) > 0 {
		name := globalQueue[0]
		globalQueue = globalQueue[1:]

		if !deriveFact3(globalState["facts"][name]) {
			continue
		}

		for dependency := range dependencies[name] {
			globalQueue = append(globalQueue, dependency)
		}
	}

	CheckViolations()

	customDerivation = false
}

func deriveFactsOnce3() bool {
	changed := false

	for _, fact := range globalState["facts"] {
		changed = deriveFact(fact) || changed
	}

	return changed
}

func deriveFact3(fact interface{}) bool {
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
			tempAssumptions = make([]*Assumptions, 0)

			for expr := range handleExpression(rule, signal) {
				if expr.Identifier != name {
					expr = Expression{
						Identifier: name,
						Operands:   []Expression{expr},
					}
				}

				//log.Println("Derived", name, "with", expr)

				for _, assumptions := range tempAssumptions {
					if _, ok := globalAssumptions[assumptions.Expression]; !ok {
						globalAssumptions[assumptions.Expression] = assumptions
					}
				}

				hash, err := hashstructure.Hash(expr, hashstructure.FormatV2, nil)

				if err != nil {
					panic(err)
				}

				if assumed, ok := globalAssumptions[hash]; ok {
					// Need to revert the state
					//log.Println("Reverting", name)
					globalInstances = assumed.Knowledge
					globalQueue = assumed.Queue
					globalAssumptions = assumed.Assumptions

					return changed
				}

				err = create(expr, true)

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
