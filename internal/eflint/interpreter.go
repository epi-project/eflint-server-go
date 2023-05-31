package eflint

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
)

func print1(input interface{}) {
	bytes, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bytes))
}

func getFactName(name string) string {
	// If the name ends with quotation marks or digits, remove those
	name = strings.TrimRight(name, "'0123456789")

	if globalState["placeholders"][name] != nil {
		return getFactName(globalState["placeholders"][name].(string))
	}

	return name
}

// InterpretPhrases interprets the given phrases and returns the results
func InterpretPhrases(phrases []Phrase) {
	// Clean the global result and error state
	globalErrors = make([]Error, 0)
	globalResults = make([]interface{}, 0)

	// Initialise the global state if it is empty
	// TODO: This distinction is helpful for derivations, as you can
	//       then ignore non-instances of a fact when deriving. This
	//       way, only "unknown" facts are derived.
	globalState = make(map[string]map[string]interface{})
	globalState["facts"] = make(map[string]interface{})
	globalState["predicates"] = make(map[string]interface{})
	globalState["placeholders"] = make(map[string]interface{})
	globalState["instances"] = make(map[string]interface{})
	globalState["non-instances"] = make(map[string]interface{})

	for _, phrase := range phrases {
		if err := InterpretPhrase(phrase); err != nil {
			// TODO: Stop after first error? Or continue?
			log.Println(err, "oh no")
		}
	}

	//log.Println("Global state:")
	//glob, _ := json.MarshalIndent(globalState, "", "  ")
	//log.Println(string(glob))
}

func addViolation(reason string, violation Expression) {
	violations := globalState["violations"]

	if _, ok := violations[reason]; !ok {
		violations[reason] = make([]interface{}, 0)
	}

	violations[reason] = append(violations[reason].([]interface{}), violation)
}

func listViolations() {
	violations := globalState["violations"]

	if len(violations) == 0 {
		return
	}

	fmt.Println("violations:")

	for reason, violations := range globalState["violations"] {
		for _, violation := range violations.([]interface{}) {
			fmt.Print("  ")
			switch reason {

			case "action":
				fmt.Print("disabled action")
			case "duty":
				fmt.Print("violated duty!")
			case "invariant":
				fmt.Print("violated invariant!")
			}
			fmt.Println(":", formatExpression(violation.(Expression)))
		}
	}
}

func InterpretPhrase(phrase Phrase) error {
	globalState["violations"] = make(map[string]interface{})
	var err error = nil

	switch phrase.Kind {
	case "afact":
		err = handleAtomicFact(phrase)
	case "cfact":
		err = handleCompositeFact(phrase)
	case "placeholder":
		err = handlePlaceholder(phrase)
	case "create":
		err = handleCreate(*phrase.Operand, false)
	case "terminate":
		err = handleTerminate(*phrase.Operand)
	case "obfuscate":
		err = handleObfuscate(*phrase.Operand)
	case "bquery":
		err = handleBQuery(*phrase.Expression)
	case "iquery":
		err = handleIQuery(*phrase.Expression)
	case "predicate":
		err = handlePredicate(phrase)
	case "event":
		err = handleEvent(phrase)
	case "act":
		err = handleAct(phrase)
	case "duty":
		err = handleDuty(phrase)
	case "trigger":
		err = handleTrigger(*phrase.Operand)
	default:
		//err = fmt.Errorf("unknown phrase kind: %s", phrase.Kind)
	}

	//log.Println("instances after phrase but before derivations:")
	//log.Println(globalState["instances"])
	//log.Println(globalState["non-instances"])
	//log.Println("The end")

	DeriveFacts()

	listViolations()

	return err
}

func handlePredicate(phrase Phrase) error {
	// A predicate is a fact without parameters
	return handleAtomicFact(Phrase{
		Name:        phrase.Name,
		HoldsWhen:   []Expression{*phrase.Expression},
		IsInvariant: phrase.IsInvariant,
	})
}

func handleEvent(phrase Phrase) error {
	// An event is a composite fact
	return handleCompositeFact(Phrase{
		Name:          phrase.Name,
		IdentifiedBy:  phrase.RelatedTo,
		DerivedFrom:   phrase.DerivedFrom,
		HoldsWhen:     phrase.HoldsWhen,
		ConditionedBy: phrase.ConditionedBy,
		SyncsWith:     phrase.SyncsWith,
		Creates:       phrase.Creates,
		Terminates:    phrase.Terminates,
		Obfuscates:    phrase.Terminates,
		FactType:      EventType,
	})
}

func handleAct(phrase Phrase) error {
	return handleCompositeFact(Phrase{
		Name:          phrase.Name,
		IdentifiedBy:  append([]string{phrase.Actor}, phrase.RelatedTo...),
		DerivedFrom:   phrase.DerivedFrom,
		HoldsWhen:     phrase.HoldsWhen,
		ConditionedBy: phrase.ConditionedBy,
		SyncsWith:     phrase.SyncsWith,
		Creates:       phrase.Creates,
		Terminates:    phrase.Terminates,
		Obfuscates:    phrase.Terminates,
		FactType:      ActType,
	})
}

func handleDuty(phrase Phrase) error {
	return handleCompositeFact(Phrase{
		Name:          phrase.Name,
		IdentifiedBy:  append([]string{phrase.Holder, phrase.Claimant}, phrase.RelatedTo...),
		DerivedFrom:   phrase.DerivedFrom,
		HoldsWhen:     phrase.HoldsWhen,
		ConditionedBy: phrase.ConditionedBy,
		ViolatedWhen:  phrase.ViolatedWhen,
		FactType:      ActType,
	})
}

func handlePlaceholder(phrase Phrase) error {
	if names, ok := phrase.Name.([]string); ok {
		name := names[0]
		if _, ok := globalState["placeholders"][name]; ok {
			return fmt.Errorf("placeholder %s already exists", name)
		} else {
			globalState["placeholders"][name] = phrase.For
			log.Println("New placeholder:", phrase.Name, phrase.For)
			return nil
		}
	} else {
		panic("placeholder name is not a list of strings")
	}
}

func fillParameters(expression Expression, params []string, values []Expression) Expression {
	newExpression := copyExpression(expression)

	for i, param := range params {
		occurrences := findOccurrences(&newExpression, param)
		for _, occurrence := range occurrences {
			*occurrence = values[i]
		}
	}

	return newExpression
}

func handleTrigger(operand Expression) error {
	// A trigger can trigger an Event

	// Iterate over the given operand
	for expr := range handleExpression(operand) {
		if expr.Identifier == "" {
			log.Println("Skipping non-identifier expression in trigger")
			continue
		}

		// Check if the given identifier is a fact which is triggerable
		if fact, ok := globalState["facts"][expr.Identifier]; ok {
			if cfact, ok := fact.(CompositeFact); ok {
				if cfact.FactType == ActType {
					// Need to check if the fact is triggerable by checking if it holds true
					eval, err := evaluateInstance(expr)
					if err != nil {
						log.Println("Error in evaluating Act")
						continue
					}

					if !eval {
						// TODO: Non-true act can still be enabled if its conditioned-by fields are okay.
						//log.Println("Triggering disabled act")
						addViolation("action", copyExpression(expr))
					} else {
						log.Println("Triggering act", cfact.Name)
					}
				} else if cfact.FactType == EventType {
					log.Println("Triggering event", cfact.Name)
				} else if cfact.FactType == DutyType {
					log.Println("Triggering duty", cfact.Name)
				} else {
					log.Println("Fact is not triggerable")
					break
				}

				for _, create := range cfact.Creates {
					handleCreate(fillParameters(create, cfact.IdentifiedBy, expr.Operands), false)
				}

				for _, terminate := range cfact.Terminates {
					handleTerminate(fillParameters(terminate, cfact.IdentifiedBy, expr.Operands))
				}

				for _, obfuscate := range cfact.Obfuscates {
					handleObfuscate(fillParameters(obfuscate, cfact.IdentifiedBy, expr.Operands))
				}

				for _, sync := range cfact.SyncsWith {
					handleTrigger(fillParameters(sync, cfact.IdentifiedBy, expr.Operands))
				}

			} else {
				log.Println("Fact is not triggerable")
			}
		} else {
			log.Println("Fact not found in trigger")
		}
	}

	return nil
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
		IsInvariant:   fact.IsInvariant,
	}

	globalState["facts"][afact.Name] = afact

	// Initialise instances and non-instances for the atomic fact
	globalState["instances"][afact.Name] = make([]interface{}, 0)
	globalState["non-instances"][afact.Name] = make([]interface{}, 0)
	globalResults = append(globalResults, StateChanges{
		Success:    true,
		Changes:    []Phrase{fact},
		Triggers:   nil,
		Violated:   false,
		Violations: nil,
	})

	log.Println("New type", afact.Name)

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
		SyncsWith:     fact.SyncsWith,
		Creates:       fact.Creates,
		Terminates:    fact.Terminates,
		Obfuscates:    fact.Terminates,
		ViolatedWhen:  fact.ViolatedWhen,
		FactType:      fact.FactType,
	}

	globalState["facts"][cfact.Name] = cfact

	// Initialise instances and non-instances for the composite fact
	globalState["instances"][cfact.Name] = make([]interface{}, 0)
	globalState["non-instances"][cfact.Name] = make([]interface{}, 0)
	globalResults = append(globalResults, StateChanges{
		Success:    true,
		Changes:    []Phrase{fact},
		Triggers:   nil,
		Violated:   false,
		Violations: nil,
	})

	log.Println("New type", cfact.Name)

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

func factExists(identifier string) bool {
	for factname := range globalState["facts"] {
		if factname == identifier {
			return true
		}
	}

	return false
}

func formatValue(v interface{}) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case []string:
		return fmt.Sprintf("%s", v.([]string)[0])
	default:
		return fmt.Sprintf("%v", v)
	}
}

func canCreate(operand Expression) error {
	// First check if the fact exists
	if !factExists(operand.Identifier) {
		return fmt.Errorf("fact %s does not exist", operand.Identifier)
	}

	// If it is an atomic fact, check if the value is of the correct type
	// and in the range of the fact.
	if _, ok := globalState["facts"][operand.Identifier].(AtomicFact); ok {
		if !checkRange(operand.Operands[0].Value, globalState["facts"][operand.Identifier]) {
			value := operand.Operands[0].Value
			return fmt.Errorf("value %s is not in the range of fact %s", formatValue(value), operand.Identifier)
		}
	}

	// If it is a composite fact, check if the values are of the correct type
	// by checking if we can recursively create them.
	if _, ok := globalState["facts"][operand.Identifier].(CompositeFact); ok {
		for _, expr := range operand.Operands {
			err := canCreate(expr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func convertAtomic(operand Expression, target string) Expression {
	if operand.Value != nil {
		// Primitive value, check if we can convert it
		if reflect.TypeOf(operand.Value) == intType && target == "Int" {
			return operand
		} else if reflect.TypeOf(operand.Value) == stringType && target == "String" {
			return operand
		} else {
			// Try to convert the value
			if !factExists(target) {
				panic("Conversion target does not exist")
			}
			if afact, ok := globalState["facts"][target].(AtomicFact); ok {
				newOperand := convertAtomic(operand, afact.Type)
				if newOperand.Value != nil {
					return Expression{
						Identifier: target,
						Operands: []Expression{
							newOperand,
						},
					}
				}
			}

			panic("Cannot convert primitive value to composite fact")
		}
	} else if operand.Identifier != "" {
		if !factExists(operand.Identifier) {
			panic("Fact does not exist")
		}

		if afact, ok := globalState["facts"][operand.Identifier].(AtomicFact); ok {
			if afact.Type == target {
				return convertAtomic(operand.Operands[0], target)
			} else if afact.Name == target {
				return operand
			} else {
				panic("Don't know how to convert " + afact.Type + " to " + target)
			}
		} else {
			panic("Cannot convert composite fact to atomic fact")
		}
	} else {
		panic("Unknown error in convertAtomic")
	}
}

func convertComposite(operands []Expression, targets []string) []Expression {
	for i := range operands {
		// Find target[i] in the global state
		target := getFactName(targets[i])
		if !factExists(target) {
			panic("Fact does not exist")
		}
		if _, ok := globalState["facts"][target].(AtomicFact); ok {
			operands[i] = convertAtomic(operands[i], target)
		} else {
			operands[i].Operands = convertComposite(operands[i].Operands, globalState["facts"][target].(CompositeFact).IdentifiedBy)
		}
	}

	return operands
}

func equalInstances(instance1 Expression, instance2 Expression) bool {
	if instance1.Value != nil {
		return instance1.Value == instance2.Value
	}

	if instance1.Identifier != instance2.Identifier {
		return false
	}

	if len(instance1.Operands) != len(instance2.Operands) {
		return false
	}

	for i := range instance1.Operands {
		if !equalInstances(instance1.Operands[i], instance2.Operands[i]) {
			return false
		}
	}

	//log.Println("equalInstances: true")
	//hash1, err1 := hashstructure.Hash(instance1, hashstructure.FormatV2, nil)
	//hash2, err2 := hashstructure.Hash(instance2, hashstructure.FormatV2, nil)
	//if err1 != nil || err2 != nil {
	//	panic("hashstructure error")
	//}
	//log.Println("Hash equalInstances: ", hash1 == hash2)

	return true
}

func convertInstance(operand Expression) (Expression, error) {
	if !factExists(operand.Identifier) {
		return operand, fmt.Errorf("fact %s does not exist", operand.Identifier)
	}

	fact := globalState["facts"][operand.Identifier]
	if afact, ok := fact.(AtomicFact); ok {
		if len(operand.Operands) == 0 && afact.Type == "" {
			return operand, nil
		}

		if len(operand.Operands) != 1 {
			return operand, fmt.Errorf("atomic fact operands mismatch")
		}

		operand.Operands[0] = convertAtomic(operand.Operands[0], afact.Type)

	} else if cfact, ok := fact.(CompositeFact); ok {
		if len(operand.Operands) != len(cfact.IdentifiedBy) {
			return operand, fmt.Errorf("composite fact operands mismatch")
		}
		operand.Operands = convertComposite(operand.Operands, cfact.IdentifiedBy)
	}

	return operand, canCreate(operand)
}

func equalInstanceContents(instance1 Expression, instance2 Expression) bool {
	if instance1.Value != nil && instance2.Value != nil {
		return instance1.Value == instance2.Value
	} else if instance1.Value != nil {
		if instance2.Identifier != "" {
			if _, ok := globalState["facts"][instance2.Identifier].(AtomicFact); ok {
				return instance2.Operands[0].Value == instance1.Value
			} else {
				return false
			}
		}
	} else if instance2.Value != nil {
		if instance1.Identifier != "" {
			if _, ok := globalState["facts"][instance1.Identifier].(AtomicFact); ok {
				return instance2.Value == instance1.Operands[0].Value
			} else {
				return false
			}
		}
	}

	if len(instance1.Operands) != len(instance2.Operands) {
		return false
	}

	if !factExists(instance1.Identifier) || !factExists(instance2.Identifier) {
		log.Println("One of the facts does not exist")
		return false
	}

	fact1 := globalState["facts"][instance1.Identifier]
	fact2 := globalState["facts"][instance2.Identifier]

	afact1, aok1 := fact1.(AtomicFact)
	afact2, aok2 := fact2.(AtomicFact)

	if aok1 && aok2 && len(instance1.Operands) == 1 {
		if afact1.Type != afact2.Type {
			return false
		}

		return equalInstanceContents(instance1.Operands[0], instance2.Operands[0])
	}

	cfact1, cok1 := fact1.(CompositeFact)
	cfact2, cok2 := fact2.(CompositeFact)

	if cok1 && cok2 {
		for i := range cfact1.IdentifiedBy {
			if cfact1.IdentifiedBy[i] != cfact2.IdentifiedBy[i] {
				return false
			}

			if !equalInstanceContents(instance1.Operands[i], instance2.Operands[i]) {
				return false
			}
		}
	}

	return true
}

// handleCreate explicitly sets a given expression to true,
// by moving it from the non-instances to the instances list.
func handleCreate(operand Expression, derived bool) error {
	for op := range handleExpression(operand) {
		//if op.Identifier == "" {
		//	log.Println("Skipping non-identifier expression", formatExpression(op))
		//	continue
		//}

		op, err := convertInstance(op)
		if err != nil {
			return err
		}

		op.IsDerived = derived

		// If there is a non-instance for this expression, remove it
		if noninstances, ok := globalState["non-instances"][op.Identifier]; ok {
			for i, noninstance := range noninstances.([]interface{}) {
				if equalInstances(noninstance.(Expression), op) {
					if derived {
						return fmt.Errorf("cannot derive a non-instance")
					}

					globalState["non-instances"][op.Identifier] = append(noninstances.([]interface{})[:i], noninstances.([]interface{})[i+1:]...)
					break
				}
			}
		}

		instances := globalState["instances"][op.Identifier].([]interface{})

		// Loop through the instances and make sure the instance does not already exist
		for i, instance := range globalState["instances"][op.Identifier].([]interface{}) {
			if equalInstances(instance.(Expression), op) {
				if !derived {
					// Set the derived field to this instance to false, as it is now postulated.
					newExpr := instance.(Expression)
					newExpr.IsDerived = false
					instances[i] = newExpr

					return nil
				}
				return fmt.Errorf("instance %s already exists", formatExpression(op))
			}
		}

		// Add the instance to the global state
		globalState["instances"][op.Identifier] = append(globalState["instances"][op.Identifier].([]interface{}), op)
		globalResults = append(globalResults, StateChanges{
			Success:    true,
			Changes:    []Phrase{{Kind: "create", Operand: &op}},
			Triggers:   nil,
			Violated:   false,
			Violations: nil,
		})

		log.Println("+" + formatExpression(op))
	}

	return nil
}

// handleTerminate explicitly sets a given expression to false,
// by moving it from the instances to the non-instances
// list.
func handleTerminate(operand Expression) error {
	//log.Println("Terminate", operand)
	for op := range handleExpression(operand) {
		//log.Println("Terminating", op)
		if op.Identifier == "" {
			log.Println("Skipping non-identifier expression", formatExpression(op))
			continue
		}

		op, err := convertInstance(op)
		if err != nil {
			return err
		}

		// If there is an instance for this expression, remove it
		if instances, ok := globalState["instances"][op.Identifier]; ok {
			for i, instance := range instances.([]interface{}) {
				if equalInstances(instance.(Expression), op) {
					globalState["instances"][op.Identifier] = append(instances.([]interface{})[:i], instances.([]interface{})[i+1:]...)
					break
				}
			}
		}

		// Loop through the non-instances and make sure the non-instance does not already exist
		for _, noninstance := range globalState["non-instances"][op.Identifier].([]interface{}) {
			if equalInstances(noninstance.(Expression), op) {
				return fmt.Errorf("non-instance %s already exists", formatExpression(op))
			}
		}

		// Add the non-instance to the global state
		globalState["non-instances"][op.Identifier] = append(globalState["non-instances"][op.Identifier].([]interface{}), op)
		globalResults = append(globalResults, StateChanges{
			Success:    true,
			Changes:    []Phrase{{Kind: "terminate", Operand: &op}},
			Triggers:   nil,
			Violated:   false,
			Violations: nil,
		})

		log.Println("-" + formatExpression(op))
	}

	return nil
}

// handleObfuscate implicitly sets a given expression to false,
// by removing it from both the instances and non-instances list.
func handleObfuscate(operand Expression) error {
	for op := range handleExpression(operand) {
		if op.Identifier == "" {
			log.Println("Skipping non-identifier expression", formatExpression(op))
			continue
		}

		op, err := convertInstance(op)
		if err != nil {
			return err
		}

		// If there is an instance for this expression, remove it
		if instances, ok := globalState["instances"][op.Identifier]; ok {
			for i, instance := range instances.([]interface{}) {
				if equalInstances(instance.(Expression), op) {
					globalState["instances"][op.Identifier] = append(instances.([]interface{})[:i], instances.([]interface{})[i+1:]...)
					break
				}
			}
		}

		// If there is a non-instance for this expression, remove it
		if noninstances, ok := globalState["non-instances"][op.Identifier]; ok {
			for i, noninstance := range noninstances.([]interface{}) {
				if equalInstances(noninstance.(Expression), op) {
					globalState["non-instances"][op.Identifier] = append(noninstances.([]interface{})[:i], noninstances.([]interface{})[i+1:]...)
					break
				}
			}
		}

		globalResults = append(globalResults, StateChanges{
			Success:    true,
			Changes:    []Phrase{{Kind: "obfuscate", Operand: &op}},
			Triggers:   nil,
			Violated:   false,
			Violations: nil,
		})

		log.Println("~" + formatExpression(op))
	}

	return nil
}

func handleBQuery(expression Expression) error {
	instances := handleExpression(expression)
	if instances == nil {
		panic("empty handleExpression result")
	}

	instance := <-instances

	// Check if there is more than one result
	if _, ok := <-instances; ok {
		panic("multiple results from handleExpression")
	}

	result, err := evaluateInstance(instance)

	if err != nil {
		// TODO: Based on the kind of error, handle it differently
		panic(err)
	}

	globalResults = append(globalResults, BQueryResult{
		Success: true,
		Errors:  nil,
		Result:  result,
	})

	log.Println("?" + formatExpression(expression) + " = " + strconv.FormatBool(result))

	return nil

	// Assume a bquery with a single operand
	//if expression.Identifier == "" {
	//	return nil
	//}
	//if len(expression.Operands) != 1 {
	//	return nil
	//}
	//if _, ok := expression.Operands[0].Value.(string); !ok {
	//	return nil
	//}
	//
	//if _, ok := globalState["instances"][expression.Identifier]; !ok {
	//	// add to global results
	//	globalResults = append(globalResults, BQueryResult{
	//		Success: false,
	//		Errors: []Error{
	//			{
	//				Id:      "undeclared-type",
	//				Message: "Undeclared type or placeholder: " + expression.Identifier,
	//			},
	//		},
	//	})
	//
	//	return nil
	//}
	//
	//result := BQueryResult{
	//	Success: true,
	//	Errors:  nil,
	//	Result:  false,
	//}
	//
	//for _, instance := range globalState["instances"][expression.Identifier].([]interface{}) {
	//	if instance == expression.Operands[0].Value.(string) {
	//		// add to global results
	//		result.Result = true
	//		break
	//	}
	//}
	//
	//// add to global results
	//fields := ""
	//if len(expression.Operands) > 0 {
	//	fields = "(" + expression.Operands[0].Value.(string) + ")"
	//}
	//log.Println("?" + expression.Identifier + fields + " = " + strconv.FormatBool(result.Result))
	//globalResults = append(globalResults, result)
	//
	//return nil
}

func isFiniteFact(factName string) bool {
	factName = getFactName(factName)

	if fact, ok := globalState["facts"][factName]; ok {
		if afact, ok := fact.(AtomicFact); ok {
			return len(afact.Range) > 0 || afact.Type == ""
		} else if cfact, ok := fact.(CompositeFact); ok {
			for _, param := range cfact.IdentifiedBy {
				if !isFiniteFact(param) {
					return false
				}
			}
			return true
		}
	}

	panic("fact does not exist or something went wrong")
	return false
}

func iterateFact(factName string) <-chan ConstructorApplication {
	c := make(chan ConstructorApplication)

	factName = getFactName(factName)

	result := ConstructorApplication{
		Identifier: factName,
		Operands:   nil,
	}

	if isFiniteFact(factName) {
		// Iterate over all possible instances for finite facts
		go func() {
			if fact, ok := globalState["facts"][factName].(AtomicFact); ok {
				if len(fact.Range) == 0 {
					c <- result
				}

				for _, instance := range fact.Range {
					result.Operands = []Expression{
						instance,
					}
					c <- result
				}
			} else if fact, ok := globalState["facts"][factName].(CompositeFact); ok {
				var instances [][]interface{}
				for _, param := range fact.IdentifiedBy {
					pInstances := make([]interface{}, 0)
					for instance := range iterateFact(param) {
						pInstances = append(pInstances, instance)
					}
					instances = append(instances, pInstances)
				}

				combinations := cartesianProduct(instances...)
				for _, combination := range combinations {
					result.Operands = make([]Expression, 0)
					for _, param := range combination {
						result.Operands = append(result.Operands, Expression{
							Identifier: param.(ConstructorApplication).Identifier,
							Operands:   param.(ConstructorApplication).Operands,
						})
					}
					c <- result
				}
			}
			close(c)
		}()
	} else {
		// Iterate over all known instances for infinite facts
		go func() {
			for _, instance := range globalState["instances"][factName].([]interface{}) {
				switch instance.(type) {
				case string:
					result.Operands = []Expression{
						{
							Value: instance.(string),
						},
					}
					c <- result
				case int64:
					result.Operands = []Expression{
						{
							Value: instance.(int64),
						},
					}
					c <- result
				case Expression:
					c <- ConstructorApplication{
						Identifier: instance.(Expression).Identifier,
						Operands:   instance.(Expression).Operands,
					}
				}
			}
			close(c)
		}()
	}

	return c
}

func cartesianProduct(params ...[]interface{}) (result [][]interface{}) {
	c := 1
	for _, param := range params {
		c *= len(param)
	}

	if c == 0 {
		return [][]interface{}{nil}
	}

	p := make([][]interface{}, c)
	b := make([]interface{}, c*len(params))
	n := make([]int, len(params))
	s := 0

	for i := range p {
		e := s + len(params)
		pi := b[s:e]
		p[i] = pi
		s = e
		for j, n := range n {
			pi[j] = params[j][n]
		}
		for j := len(n) - 1; j >= 0; j-- {
			n[j]++
			if n[j] < len(params[j]) {
				break
			}
			n[j] = 0
		}
	}

	return p
}

func formatExpression(expression Expression) string {
	if expression.Value != nil {
		return formatValue(expression.Value)
	} else if expression.Identifier != "" {
		result := expression.Identifier + "("
		for i, operand := range expression.Operands {
			if i > 0 {
				result += ", "
			}
			result += formatExpression(operand)
		}
		result += ")"
		return result
	} else if expression.Operator != "" {
		switch expression.Operator {
		case "ADD":
			return formatExpression(expression.Operands[0]) + " + " + formatExpression(expression.Operands[1])
		case "NOT":
			return "!" + formatExpression(expression.Operands[0])
		case "LT":
			return formatExpression(expression.Operands[0]) + " < " + formatExpression(expression.Operands[1])
		case "GT":
			return formatExpression(expression.Operands[0]) + " > " + formatExpression(expression.Operands[1])
		case "GTE":
			return formatExpression(expression.Operands[0]) + " >= " + formatExpression(expression.Operands[1])
		case "LTE":
			return formatExpression(expression.Operands[0]) + " <= " + formatExpression(expression.Operands[1])
		case "EQ":
			return formatExpression(expression.Operands[0]) + " == " + formatExpression(expression.Operands[1])
		case "NEQ":
			return formatExpression(expression.Operands[0]) + " != " + formatExpression(expression.Operands[1])
		}
	} else if expression.Parameter != "" {
		return formatExpression(*expression.Operand) + "." + expression.Parameter
	}

	return ""
}

func printExpression(expression Expression, newline bool) {
	fmt.Print(formatExpression(expression))
	if newline {
		fmt.Println()
	}
}

func handleIQuery(expression Expression) error {
	log.Println("?-" + formatExpression(expression))

	for instance := range handleExpression(expression) {
		if instance.Identifier == "" {
			panic("invalid instance in iquery result")
		}

		printExpression(instance, true)
	}

	//if reference, ok := expression.Value.([]string); ok {
	//	if len(reference) != 1 {
	//		return nil
	//	}
	//
	//	value := reference[0]
	//
	//	if !factExists(value) {
	//		return fmt.Errorf("fact %s does not exist", value)
	//	}
	//
	//	result := IQueryResult{
	//		Success: true,
	//		Errors:  nil,
	//		Result:  []Expression{},
	//	}
	//
	//	for instance := range iterateFact(value) {
	//		printExpression(Expression{
	//			Identifier: instance.Identifier,
	//			Operands:   instance.Operands,
	//		}, true)
	//
	//		result.Result = append(result.Result, Expression{
	//			Identifier: instance.Identifier,
	//			Operands:   instance.Operands,
	//		})
	//	}
	//
	//	// add to global results
	//	globalResults = append(globalResults, result)
	//	//log.Printf("Got %d results\n", len(result.Result))
	//} else {
	//	for instance := range handleExpression(expression) {
	//		log.Println(formatExpression(instance))
	//	}
	//}

	return nil
}

func findVariable(expression Expression) string {
	if expression.Value != nil {
		if ref, ok := expression.Value.([]string); ok {
			if len(ref) == 1 {
				return ref[0]
			}
		}
	} else if expression.Identifier != "" || expression.Operator != "" {
		for _, operand := range expression.Operands {
			if variable := findVariable(operand); variable != "" {
				return variable
			}
		}
	}

	return ""
}

// TODO: When an iterator is found, the variable that the iterator binds should not be replaced
func findOccurrences(expression *Expression, variable string) []*Expression {
	if expression.Value != nil {
		if ref, ok := expression.Value.([]string); ok {
			if len(ref) == 1 && ref[0] == variable {
				return []*Expression{expression}
			}
		}
	} else if expression.Identifier != "" || expression.Operator != "" {
		var result []*Expression
		for i := range expression.Operands {
			result = append(result, findOccurrences(&expression.Operands[i], variable)...)
		}
		return result
	}

	return []*Expression{}
}

func copyExpression(expression Expression) Expression {
	result := Expression{
		Identifier: expression.Identifier,
		Operator:   expression.Operator,
		Value:      expression.Value,
		Operands:   []Expression{},
		Binds:      []string{},
		Iterator:   expression.Iterator,
		IsDerived:  expression.IsDerived,
	}

	if expression.Expression != nil {
		copyNested := copyExpression(*expression.Expression)
		result.Expression = &copyNested
	}

	if expression.Operand != nil {
		copyNested := copyExpression(*expression.Operand)
		result.Operand = &copyNested
	}

	for _, operand := range expression.Operands {
		result.Operands = append(result.Operands, copyExpression(operand))
	}

	for _, bind := range expression.Binds {
		result.Binds = append(result.Binds, bind)
	}

	return result
}

func evaluateInstance(instance Expression) (bool, error) {
	if instance.Value != nil {
		switch instance.Value.(type) {
		case []string:
			return handleExpression(instance) != nil, nil
		case bool:
			return instance.Value.(bool), nil
		case string:
			return instance.Value.(string) != "", nil
		case int64:
			return instance.Value.(int64) > 0, nil
		default:
			panic("invalid type")
			//return false, fmt.Errorf("invalid type %T", instance.Value)
		}
	} else if instance.Identifier != "" {
		log.Println("Evaluating", formatExpression(instance))
		if findVariable(instance) != "" {
			panic("instance contains variables")
		}

		instance, err := convertInstance(instance)
		if err != nil {
			// TODO: TEMPORARY
			return false, nil
			panic(err)
		}

		// Search for the fact
		if !factExists(instance.Identifier) {
			return false, ErrUnknownType
		}

		// Check if the instance is already known
		log.Println(globalState["instances"][instance.Identifier])
		for _, knownInstance := range globalState["instances"][instance.Identifier].([]interface{}) {
			if instanceExpr, ok := knownInstance.(Expression); ok {
				if equalInstances(instanceExpr, instance) {
					return true, nil
				}
			}
		}
	} else {
		log.Println("Don't know what to do with instance:", instance)
	}

	return false, nil
}

// TODO: This can return any expression
func handleExpression(expression Expression) <-chan Expression {
	c := make(chan Expression)

	if err := TypeCheckExpression(&expression); err != nil {
		panic(err)
	}

	// Check if there are any variables in the expression
	ref := findVariable(expression)
	if ref != "" {
		// Find all occurrences of the variable
		occurrences := findOccurrences(&expression, ref)

		go func() {
			// Iterate over all instances of the variable
			for instance := range iterateFact(ref) {

				// Replace all occurrences of the variable with the instance
				for _, occurrence := range occurrences {
					*occurrence = Expression{
						Identifier: instance.Identifier,
						Operands:   instance.Operands,
					}
				}

				//log.Println("New expr:", expression)

				for result := range handleExpression(copyExpression(expression)) {
					//log.Println("Got result:", formatExpression(result))
					c <- copyExpression(result)
				}
			}

			close(c)

			// Put the original expression back
			for _, occurrence := range occurrences {
				*occurrence = Expression{
					Value: []string{ref},
				}
			}
		}()

		return c
	}

	if ref, ok := expression.Value.([]string); ok {
		if len(ref) != 1 {
			close(c)
			return c
		}

		go func() {
			for instance := range iterateFact(ref[0]) {
				c <- Expression{
					Identifier: instance.Identifier,
					Operands:   instance.Operands,
				}
			}
			close(c)
		}()
	} else if val, ok := expression.Value.(int64); ok {
		go func() {
			c <- Expression{
				Value: val,
			}
			close(c)
		}()
	} else if val, ok := expression.Value.(string); ok {
		go func() {
			c <- Expression{
				Value: val,
			}
			close(c)
		}()
	} else if expression.Operator != "" {
		go func() {
			for operand := range handleOperator(expression) {
				c <- operand
			}
			close(c)
		}()
	} else if expression.Identifier != "" {
		// TODO: Get all instances for the operands and return them

		//if len(expression.Operands) == 0 {
		//	panic("No operands for expression")
		//}

		for i := range expression.Operands {
			// TODO: CHeck if this is correct (It is not!)
			expression.Operands[i], ok = <-handleExpression(expression.Operands[i])
			if !ok {
				close(c)
				return c
			}
		}

		go func() {
			// TODO: This is needed as we cannot always evaluate instances to true/false (citizen(Bob))
			c <- expression
			close(c)
		}()
	} else if expression.Iterator != "" {
		go func() {
			for expr := range handleIterator(expression) {
				c <- expr
			}
			close(c)
		}()
	} else if expression.Parameter != "" {
		go func() {
			for expr := range handleProjection(expression) {
				c <- expr
			}
			close(c)
		}()
	} else {
		log.Println("Unknown expression type", expression.Parameter, ":(")
		panic("Unknown expression type")
		close(c)
	}

	return c
}

func handleArithmeticOperator(operator string, operand1 int64, operand2 int64) interface{} {
	switch operator {
	case "ADD":
		return operand1 + operand2
	case "SUB":
		return operand1 - operand2
	case "MUL":
		return operand1 * operand2
	case "DIV":
		return operand1 / operand2
	case "MOD":
		return operand1 % operand2
	case "GT":
		return operand1 > operand2
	case "LT":
		return operand1 < operand2
	case "GTE":
		return operand1 >= operand2
	case "LTE":
		return operand1 <= operand2
	default:
		panic("Unknown operator")
	}
}

func instanceToInt(expression Expression) Expression {
	if !factExists(expression.Identifier) || len(expression.Operands) == 0 {
		return expression
	}

	fact := globalState["facts"][expression.Identifier]

	if afact, ok := fact.(AtomicFact); ok {
		if afact.Type == "Int" {
			return expression.Operands[0]
		}
	}

	return expression
}

func handleOperator(expression Expression) <-chan Expression {
	c := make(chan Expression)

	if expression.Operator == "ADD" || expression.Operator == "SUB" || expression.Operator == "MUL" || expression.Operator == "DIV" || expression.Operator == "MOD" ||
		expression.Operator == "LT" || expression.Operator == "GT" || expression.Operator == "LTE" || expression.Operator == "GTE" {
		go func() {
			// TODO: Check if exactly two operands
			expression1 := <-handleExpression(expression.Operands[0])
			expression2 := <-handleExpression(expression.Operands[1])

			expression1 = instanceToInt(expression1)
			expression2 = instanceToInt(expression2)

			if expression1.Value == nil || expression2.Value == nil {
				panic("nil value")
			}

			if reflect.TypeOf(expression1.Value) != intType {
				panic("Cannot convert to expression1 to int")
			}

			if reflect.TypeOf(expression2.Value) != intType {
				panic("Cannot convert to expression2 to int")
			}

			c <- Expression{
				Value: handleArithmeticOperator(expression.Operator, expression1.Value.(int64), expression2.Value.(int64)),
			}

			close(c)
		}()
	} else if expression.Operator == "EQ" || expression.Operator == "NEQ" {
		expr1 := <-handleExpression(expression.Operands[0])
		expr2 := <-handleExpression(expression.Operands[1])
		//log.Println("EQ", expr1, expr2)
		go func() {
			value := equalInstanceContents(expr1, expr2)
			if expression.Operator == "NEQ" {
				value = !value
			}

			c <- Expression{
				Value: equalInstanceContents(expr1, expr2),
			}

		}()
	} else if expression.Operator == "AND" {
		go func() {
			result := true
			for _, operand := range expression.Operands {
				expr := <-handleExpression(operand)
				//log.Println("AND", expr, operand)
				if eval, err := evaluateInstance(expr); err == nil {
					result = result && eval
				} else {
					panic(err)
				}

				if !result {
					break
				}
			}

			c <- Expression{
				Value: result,
			}
		}()
	} else if expression.Operator == "NOT" {
		//log.Println("NOT", expression)
		expr := <-handleExpression(expression.Operands[0])
		go func() {
			//log.Println("NOT", expr)
			if eval, err := evaluateInstance(expr); err == nil {
				c <- Expression{
					Value: !eval,
				}
			} else {
				panic(err)
			}

			close(c)
		}()
	} else if expression.Operator == "COUNT" {
		//log.Println("COUNT", expression)
		go func() {
			length := int64(0)

			for range handleExpression(expression.Operands[0]) {
				length++
			}

			c <- Expression{
				Value: length,
			}

			close(c)
		}()
	} else if expression.Operator == "WHEN" {
		// TODO: Procedure: Evaluate the second operand
		expr := <-handleExpression(expression.Operands[1])
		//log.Println(formatExpression(expression.Operands[0]), "WHEN", formatExpression(expr))
		if eval, err := evaluateInstance(expr); err == nil && eval {
			//log.Println("When is true")
			go func() {
				for expr := range handleExpression(expression.Operands[0]) {
					c <- expr
				}
				close(c)
			}()
		} else {
			//log.Println("When is false")
			close(c)
		}
	} else {
		log.Println("Unknown operator", expression)
		panic("Unknown operator")
	}

	return c
}

func handleIterator(expression Expression) <-chan Expression {
	c := make(chan Expression)

	if expression.Iterator == "FOREACH" {
		if len(expression.Binds) != 1 {
			log.Println(expression.Binds)
			panic("FOREACH can currently only bind one variable")
		}

		go func() {
			expr := *expression.Expression
			bind := expression.Binds[0]
			occurrences := findOccurrences(&expr, bind)

			for instance := range iterateFact(bind) {
				// Replace all occurrences of the variable with the instance
				for _, occurrence := range occurrences {
					*occurrence = Expression{
						Identifier: instance.Identifier,
						Operands:   instance.Operands,
					}
				}

				for result := range handleExpression(copyExpression(expr)) {
					c <- copyExpression(result)
				}
			}

			close(c)
		}()
	} else if expression.Iterator == "EXISTS" {
		if len(expression.Binds) != 1 {
			panic("EXISTS can currently only bind one variable")
		}

		log.Println("EXISTS", expression.Binds[0], *expression.Expression)

		go func() {
			expr := *expression.Expression
			bind := expression.Binds[0]
			occurrences := findOccurrences(&expr, bind)

			for instance := range iterateFact(bind) {
				// Replace all occurrences of the variable with the instance
				for _, occurrence := range occurrences {
					*occurrence = Expression{
						Identifier: instance.Identifier,
						Operands:   instance.Operands,
					}
				}

				log.Println("EXXISTS", expr, expression.Binds[0])
				exprResult := <-handleExpression(copyExpression(expr))

				if eval, err := evaluateInstance(exprResult); err == nil {
					if eval {
						c <- Expression{
							Value: true,
						}
						close(c)
						return
					}
				} else {
					panic(err)
				}
			}

			c <- Expression{
				Value: false,
			}
			close(c)
		}()
	} else {
		log.Println("Unknown iterator", expression)
		panic("Unknown iterator")
	}

	return c
}

func handleProjection(expression Expression) <-chan Expression {
	//log.Println("Projection", expression.Parameter, expression.Operand)
	c := make(chan Expression)

	go func() {
		for expr := range handleExpression(*expression.Operand) {
			if expr.Identifier == "" {
				panic("Cannot project non-identifier")
			}

			if !factExists(expr.Identifier) {
				panic("Cannot project non-existing fact")
			}

			fact := globalState["facts"][expr.Identifier]

			if cfact, ok := fact.(CompositeFact); ok {
				for i, param := range cfact.IdentifiedBy {
					if param == expression.Parameter {
						c <- expr.Operands[i]
						close(c)
						return
					}
				}
			} else {
				panic("Cannot project atomic fact")
			}

			close(c)
			log.Println("Expression has no parameter", expression.Parameter)

		}
	}()

	return c
}
