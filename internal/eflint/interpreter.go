package eflint

import (
	"encoding/json"
	"fmt"
	"log"
)

// InterpretPhrases interprets the given phrases and returns the results
func InterpretPhrases(phrases []Phrase) {
	for _, phrase := range phrases {
		if err := InterpretPhrase(phrase); err != nil {
			// TODO: Stop after first error? Or continue?
			log.Println(err)
			break
		}
	}

	log.Println("Global state:")
	glob, _ := json.MarshalIndent(globalState, "", "  ")
	log.Println(string(glob))
}

func InterpretPhrase(phrase Phrase) error {
	var err error = nil

	switch phrase.Kind {
	case "afact":
		// TODO: Need to keep a global state that is accessible. Maybe a global interface{} array?
		err = handleAtomicFact(phrase)
	case "cfact":
		err = handleCompositeFact(phrase)
	case "create":
		err = handleCreate(*phrase.Operand)
	case "bquery":
		fallthrough
	case "iquery":
		err = handleQuery(*phrase.Expression)
	default:
		//err = fmt.Errorf("unknown phrase kind: %s", phrase.Kind)
	}

	return err
}

func handleAtomicFact(fact Phrase) error {
	// Add the fact to the global state under the given "fact" key
	// If "fact" key is not present, add it
	// If "fact" key is present, append to
	afact := AtomicFact{
		Name:          fact.Name.(string),
		Type:          fact.Type,
		Range:         fact.Range,
		DerivedFrom:   fact.DerivedFrom,
		HoldsWhen:     fact.HoldsWhen,
		ConditionedBy: fact.ConditionedBy,
	}

	if _, ok := globalState["fact"]; !ok {
		globalState["fact"] = []interface{}{afact}
	} else {
		globalState["fact"] = append(globalState["fact"].([]interface{}), afact)
	}

	return nil
}

func handleCompositeFact(fact Phrase) error {
	// Add the fact to the global state under the given "fact" key
	// If "fact" key is not present, add it
	// If "fact" key is present, append to it
	cfact := CompositeFact{
		Name:          fact.Name.(string),
		IdentifiedBy:  fact.IdentifiedBy,
		DerivedFrom:   fact.DerivedFrom,
		HoldsWhen:     fact.HoldsWhen,
		ConditionedBy: fact.ConditionedBy,
	}

	if _, ok := globalState["fact"]; !ok {
		globalState["fact"] = []interface{}{cfact}
	} else {
		globalState["fact"] = append(globalState["fact"].([]interface{}), cfact)
	}

	return nil
}

func handleCreate(operand Expression) error {
	// Get rid of stuff that is not yet supported
	if operand.Identifier == "" {
		return nil
	}
	if len(operand.Operands) != 1 {
		return nil
	}
	if _, ok := operand.Operands[0].Value.(string); !ok {
		return nil
	}

	// Make sure the fact exists
	found := false
	facts := globalState["fact"].([]interface{})
	for _, fact := range facts {
		switch fact.(type) {
		case AtomicFact:
			if fact.(AtomicFact).Name == operand.Identifier {
				found = true
			}
		case CompositeFact:
			if fact.(CompositeFact).Name == operand.Identifier {
				found = true
			}
		}
	}

	if !found {
		return fmt.Errorf("fact %s not found", operand.Identifier)
	}

	if _, ok := globalState[operand.Identifier]; !ok {
		globalState[operand.Identifier] = []interface{}{operand.Operands[0].Value.(string)}
	} else {
		globalState[operand.Identifier] = append(globalState[operand.Identifier].([]interface{}), operand.Operands[0].Value.(string))
	}

	return nil

}

func handleQuery(expression Expression) error {
	// Assume a bquery with a single operand
	if expression.Identifier == "" {
		return nil
	}
	if len(expression.Operands) != 1 {
		return nil
	}
	if _, ok := expression.Operands[0].Value.(string); !ok {
		return nil
	}

	for _, instance := range globalState[expression.Identifier].([]interface{}) {
		if instance == expression.Operands[0].Value.(string) {
			// add to global results
			globalResults = append(globalResults, Result{
				Success: true,
			})
			return nil
		}
	}

	// add to global results
	globalResults = append(globalResults, Result{
		Success: false,
	})
	return nil
}
