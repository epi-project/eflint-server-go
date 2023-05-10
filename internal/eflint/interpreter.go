package eflint

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
)

func print1(input interface{}) {
	bytes, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bytes))
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

	//log.Println("Global state:")
	//glob, _ := json.MarshalIndent(globalState, "", "  ")
	//log.Println(string(glob))
}

func InterpretPhrase(phrase Phrase) error {
	var err error = nil

	switch phrase.Kind {
	case "afact":
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
		err = handleBQuery(*phrase.Expression)
	case "iquery":
		err = handleIQuery(*phrase.Expression)
	default:
		//err = fmt.Errorf("unknown phrase kind: %s", phrase.Kind)
	}

	DeriveFacts()

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

func checkConversion(operand Expression, fact interface{}) Expression {
	if operand.Value != nil {
		// Nothing to convert
		return operand
	}

	if operand.Identifier == "" || operand.Operands == nil {
		return operand
	}

	if !factExists(operand.Identifier) {
		return operand
	}

	fromFact := globalState["facts"][operand.Identifier]

	if afactTo, ok := fact.(AtomicFact); ok {
		if afactFrom, ok := fromFact.(AtomicFact); ok {
			if afactTo.Type == afactFrom.Type {
				return Expression{
					Identifier: operand.Identifier,
					Operands:   []Expression{operand.Operands[0]},
				}
			}
		}
	}

	return operand
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
			if _, ok := value.(string); ok {
				value = fmt.Sprintf("\"%s\"", value)
			} else if _, ok := value.(bool); ok {
				value = fmt.Sprintf("%t", value)
			} else if _, ok := value.(int64); ok {
				value = fmt.Sprintf("%d", value)
			}
			return fmt.Errorf("value %s is not in the range of fact %s", value, operand.Identifier)
		}
	}

	// If it is a composite fact, check if the values are of the correct type
	// by checking if we can recursively create them.

	return nil
}

// handleCreate explicitly sets a given expression to true,
// by moving it from the non-instances to the instances list.
func handleCreate(operand Expression) error {
	// Get rid of stuff that is not yet supported
	if operand.Identifier == "" {
		return fmt.Errorf("not implemented yet")
	}
	if len(operand.Operands) != 1 {
		return fmt.Errorf("not implemented yet")
	}
	//if _, ok := operand.Operands[0].Value.(string); !ok {
	//	return nil
	//}

	operand.Operands[0] = checkConversion(operand.Operands[0], globalState["facts"][operand.Identifier])

	if err := canCreate(operand); err != nil {
		// TODO: Do something with the error
		return err
	}

	// If there is a non-instance for this expression, remove it
	if noninstances, ok := globalState["non-instances"][operand.Identifier]; ok {
		for i, noninstance := range noninstances.([]interface{}) {
			if noninstance == operand.Operands[0].Value {
				// Remove the non-instance
				globalState["non-instances"][operand.Identifier] = append(noninstances.([]interface{})[:i], noninstances.([]interface{})[i+1:]...)
			}
		}
	}

	// Loop through the instances and make sure the instance does not already exist
	for _, instance := range globalState["instances"][operand.Identifier].([]interface{}) {
		if instance == operand.Operands[0].Value {
			return fmt.Errorf("instance %s already exists", operand.Operands[0].Value)
		}
	}

	// Add the instance to the global state
	globalState["instances"][operand.Identifier] = append(globalState["instances"][operand.Identifier].([]interface{}), operand.Operands[0].Value)
	globalResults = append(globalResults, StateChanges{
		Success:    true,
		Changes:    []Phrase{{Kind: "create", Operand: &operand}},
		Triggers:   nil,
		Violated:   false,
		Violations: nil,
	})

	fields := ""
	if len(operand.Operands) > 0 {
		value := operand.Operands[0].Value
		if _, ok := value.(string); ok {
			value = fmt.Sprintf("\"%s\"", value)
		} else if _, ok := value.(int64); ok {
			value = fmt.Sprintf("%d", value)
		}
		fields = "(" + value.(string) + ")"
	}
	log.Println("+" + operand.Identifier + fields)

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

	fields := ""
	if len(operand.Operands) > 0 {
		fields = "(" + operand.Operands[0].Value.(string) + ")"
	}
	log.Println("-" + operand.Identifier + fields)

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

	fields := ""
	if len(operand.Operands) > 0 {
		fields = "(" + operand.Operands[0].Value.(string) + ")"
	}
	log.Println("x" + operand.Identifier + fields)

	return nil
}

func handleBQuery(expression Expression) error {
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
		globalResults = append(globalResults, BQueryResult{
			Success: false,
			Errors: []Error{
				{
					Id:      "undeclared-type",
					Message: "Undeclared type or placeholder: " + expression.Identifier,
				},
			},
		})

		return nil
	}

	result := BQueryResult{
		Success: true,
		Errors:  nil,
		Result:  false,
	}

	for _, instance := range globalState["instances"][expression.Identifier].([]interface{}) {
		if instance == expression.Operands[0].Value.(string) {
			// add to global results
			result.Result = true
			break
		}
	}

	// add to global results
	fields := ""
	if len(expression.Operands) > 0 {
		fields = "(" + expression.Operands[0].Value.(string) + ")"
	}
	log.Println("?" + expression.Identifier + fields + " = " + strconv.FormatBool(result.Result))
	globalResults = append(globalResults, result)

	return nil
}

func isFiniteFact(factName string) bool {
	if fact, ok := globalState["facts"][factName]; ok {
		if afact, ok := fact.(AtomicFact); ok {
			return len(afact.Range) > 0
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

	result := ConstructorApplication{
		Identifier: factName,
		Operands:   nil,
	}

	if isFiniteFact(factName) {
		// Iterate over all possible instances for finite facts
		//log.Println("finite fact")
		go func() {
			if fact, ok := globalState["facts"][factName].(AtomicFact); ok {
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

				combinations := cartesianProduct2(instances...)
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
		//log.Println("infinite fact")
		go func() {
			for _, instance := range globalState["instances"][factName].([]interface{}) {
				switch instance.(type) {
				case string:
					result.Operands = []Expression{
						{
							Value: instance.(string),
						},
					}
				case int64:
					result.Operands = []Expression{
						{
							Value: instance.(int64),
						},
					}
				case Expression:
					result.Operands = []Expression{
						instance.(Expression),
					}
				}
				c <- result
			}
			close(c)
		}()
	}

	return c
}

func cartesianProduct2(params ...[]interface{}) (result [][]interface{}) {
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

func cartesianProduct(params ...[]interface{}) (result [][]interface{}) {
	if len(params) == 0 {
		return [][]interface{}{nil}
	}

	remainder := cartesianProduct(params[1:]...)
	for _, param := range params[0] {
		for _, r := range remainder {
			result = append(result, append([]interface{}{param}, r...))
		}
	}

	return
}

func handleIQuery(expression Expression) error {
	if reference, ok := expression.Value.([]string); ok {
		if len(reference) != 1 {
			return nil
		}

		value := reference[0]

		if !factExists(value) {
			return fmt.Errorf("fact %s does not exist", value)
		}

		result := IQueryResult{
			Success: true,
			Errors:  nil,
			Result:  []Expression{},
		}

		for instance := range iterateFact(value) {
			log.Println(instance)
			result.Result = append(result.Result, Expression{
				Identifier: instance.Identifier,
				Operands:   instance.Operands,
			})
		}

		// add to global results
		globalResults = append(globalResults, result)
		//log.Printf("Got %d results\n", len(result.Result))
	}

	return nil
}

// TODO: This can return any expression
func handleExpression(expression Expression) <-chan ConstructorApplication {
	c := make(chan ConstructorApplication)

	if ref, ok := expression.Value.([]string); ok {
		if len(ref) != 1 {
			close(c)
			return c
		}

		return iterateFact(ref[0])
	}

	close(c)
	return c
}

//func handleOperator(expression Expression) bool {
//	switch expression.Operator {
//	// Logical operators
//	case "AND":
//		return handleAnd(expression.Operands)
//	case "OR":
//		return handleOr(expression.Operands)
//	case "NOT":
//		return handleNot(expression.Operands)
//
//	// Comparison operators
//	case "EQ":
//		return handleEQ(expression.Operands)
//	case "NE":
//		return !handleEQ(expression.Operands)
//	case "GT":
//		return handleGT(expression.Operands)
//	}
//
//	return false
//}
//
//// TODO: This is short-circuiting, need to check if this is correct
//func handleAnd(operands []Expression) bool {
//	for _, operand := range operands {
//		if !handleExpression(operand) {
//			return false
//		}
//	}
//
//	return true
//}
//
//// TODO: This is short-circuiting, need to check if this is correct
//func handleOr(operands []Expression) bool {
//	for _, operand := range operands {
//		if handleExpression(operand) {
//			return true
//		}
//	}
//
//	return false
//}
//
//func handleNot(operands []Expression) bool {
//	return !handleExpression(operands[0])
//}
//
//func handleEQ(operands []Expression) bool {
//	return handleExpression(operands[0]) == handleExpression(operands[1])
//}
//
//func handleGT(operands []Expression) bool {
//	// TODO: handleExpression can also return a String / Int
//	return false
//}
