package eflint

import (
	"encoding/json"
	"fmt"
	"log"
)

// InterpretPhrases interprets the given phrases and returns the results
func InterpretPhrases(phrases []Phrase) {
	// Clean the global result and error state
	globalErrors = make([]Error, 0)
	globalResults = make([]Result, 0)

	// Initialise the global state if it is empty
	if len(globalState) == 0 {
		globalState = make(map[string]map[string]interface{})
		globalState["facts"] = make(map[string]interface{})
		globalState["instances"] = make(map[string]interface{})
		globalState["non-instances"] = make(map[string]interface{})
	}

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
	case "terminate":
		err = handleTerminate(*phrase.Operand)
	case "obfuscate":
		err = handleObfuscate(*phrase.Operand)
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

	globalState["facts"][afact.Name] = afact

	// Initialise instances and non-instances for the atomic fact
	globalState["instances"][afact.Name] = make([]interface{}, 0)
	globalState["non-instances"][afact.Name] = make([]interface{}, 0)

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

	globalState["facts"][cfact.Name] = cfact

	// Initialise instances and non-instances for the composite fact
	globalState["instances"][cfact.Name] = make([]interface{}, 0)
	globalState["non-instances"][cfact.Name] = make([]interface{}, 0)

	return nil
}

func checkRange(value interface{}, fact interface{}) bool {
	// First check if the fact is an atomic fact
	if _, ok := fact.(AtomicFact); !ok {
		// Composite facts do not have a range
		return true
	}

	if fact.(AtomicFact).Range == nil {
		return true
	}

	// Check if the value is in the range
	for _, expr := range fact.(AtomicFact).Range {
		if value == expr.Value {
			return true
		}
	}

	return false
}

// handleCreate explicitly sets a given expression to true,
// by moving it from the non-instances to the instances list.
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
	for factname := range globalState["facts"] {
		if factname == operand.Identifier {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("fact %s not found", operand.Identifier)
	}

	if !checkRange(operand.Operands[0].Value, globalState["facts"][operand.Identifier]) {
		return fmt.Errorf("value %s not in range of fact %s", operand.Operands[0].Value, operand.Identifier)
	}

	// If there is a non-instance for this expression, remove it
	if noninstances, ok := globalState["non-instances"][operand.Identifier]; ok {
		for i, noninstance := range noninstances.([]interface{}) {
			if noninstance == operand.Operands[0].Value.(string) {
				// Remove the non-instance
				globalState["non-instances"][operand.Identifier] = append(noninstances.([]interface{})[:i], noninstances.([]interface{})[i+1:]...)
			}
		}
	}

	// Loop through the instances and make sure the instance does not already exist
	for _, instance := range globalState["instances"][operand.Identifier].([]interface{}) {
		if instance == operand.Operands[0].Value.(string) {
			return fmt.Errorf("instance %s already exists", operand.Operands[0].Value.(string))
		}
	}

	// Add the instance to the global state
	globalState["instances"][operand.Identifier] = append(globalState["instances"][operand.Identifier].([]interface{}), operand.Operands[0].Value.(string))

	return nil
}

// handleTerminate explicitly sets a given expression to false,
// by moving it from the instances to the non-instances
// list.
func handleTerminate(operand Expression) error {
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
	facts := globalState["facts"]
	for factname := range facts {
		if factname == operand.Identifier {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("fact %s not found", operand.Identifier)
	}

	if !checkRange(operand.Operands[0].Value, globalState["facts"][operand.Identifier]) {
		return fmt.Errorf("value %s not in range of fact %s", operand.Operands[0].Value, operand.Identifier)
	}

	// If there is an instance for this expression, remove it
	if instances, ok := globalState["instances"][operand.Identifier]; ok {
		for i, instance := range instances.([]interface{}) {
			if instance == operand.Operands[0].Value.(string) {
				// Remove the instance
				globalState["instances"][operand.Identifier] = append(instances.([]interface{})[:i], instances.([]interface{})[i+1:]...)
			}
		}
	}

	// Loop through the non-instances and make sure the expression does not already exist
	for _, noninstance := range globalState["non-instances"][operand.Identifier].([]interface{}) {
		if noninstance == operand.Operands[0].Value.(string) {
			return fmt.Errorf("expression %s already exists", operand.Operands[0].Value.(string))
		}
	}

	globalState["non-instances"][operand.Identifier] = append(globalState["non-instances"][operand.Identifier].([]interface{}), operand.Operands[0].Value.(string))

	return nil
}

// handleObfuscate implicitly sets a given expression to false,
// by removing it from both the instances and non-instances list.
func handleObfuscate(operand Expression) error {
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
	for factname := range globalState["facts"] {
		if factname == operand.Identifier {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("fact %s not found", operand.Identifier)
	}

	// If there is an instance for this expression, remove it
	if instances, ok := globalState["instances"][operand.Identifier]; ok {
		for i, instance := range instances.([]interface{}) {
			if instance == operand.Operands[0].Value.(string) {
				// Remove the instance
				globalState["instances"][operand.Identifier] = append(instances.([]interface{})[:i], instances.([]interface{})[i+1:]...)
			}
		}
	}

	// If there is a non-instance for this expression, remove it
	if noninstances, ok := globalState["non-instances"][operand.Identifier]; ok {
		for i, noninstance := range noninstances.([]interface{}) {
			if noninstance == operand.Operands[0].Value.(string) {
				// Remove the non-instance
				globalState["non-instances"][operand.Identifier] = append(noninstances.([]interface{})[:i], noninstances.([]interface{})[i+1:]...)
			}
		}
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

	if _, ok := globalState["instances"][expression.Identifier]; !ok {
		// add to global results
		globalResults = append(globalResults, Result{
			Success: false,
		})
		return nil
	}

	for _, instance := range globalState["instances"][expression.Identifier].([]interface{}) {
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
