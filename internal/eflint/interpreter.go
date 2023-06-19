package eflint

import (
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"log"
	"reflect"
	"strings"
)

var (
	verbose           = false
	customDerivation  = false
	derivationVersion = 3
)

func Println(a ...any) {
	if verbose {
		fmt.Println(a...)
	}
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
	globalResults = make([]PhraseResult, 0)
	globalState = make(map[string]map[string]interface{})
	globalState["facts"] = make(map[string]interface{})
	globalState["placeholders"] = make(map[string]interface{})
	globalInstances = make(map[string]*orderedmap.OrderedMap[uint64, Expression])
	globalNonInstances = make(map[string]*orderedmap.OrderedMap[uint64, Expression])

	initializeFacts()

	for _, phrase := range phrases {
		if err := InterpretPhrase(phrase); err != nil {
			// TODO: Stop after first error? Or continue?
			log.Println(err, "oh no")
		}
	}
}

func initializeFacts() {
	for factName, factType := range defaultFacts {
		handleAtomicFact(Phrase{
			Kind: "create",
			Name: factName,
			Type: factType,
		})
	}
}

func addViolation(reason string, violation Expression) {
	if _, ok := globalViolations[reason]; !ok {
		globalViolations[reason] = make([]Expression, 0)
	}

	globalViolations[reason] = append(globalViolations[reason], violation)
}

func listViolations() {
	if len(globalViolations) == 0 {
		return
	}

	index := len(globalResults) - 1

	globalResults[index].Violated = true

	Println("violations:")

	for reason, violations := range globalViolations {
		for _, violation := range violations {
			switch reason {

			case "act":
				Println("  disabled action:", formatExpression(violation))
			case "duty":
				Println("  violated duty!:", formatExpression(violation))
			case "invariant":
				Println("  violated invariant!:", formatExpression(violation))
			}

			if violation.Value != nil {
				globalResults[index].Violations = append(globalResults[index].Violations, Violation{
					Kind:       reason,
					Identifier: violation.Value.([]string)[0],
					Operands:   []Expression{}})
			} else {
				globalResults[index].Violations = append(globalResults[index].Violations, Violation{
					Kind:       reason,
					Identifier: violation.Identifier,
					Operands:   violation.Operands})
			}
		}
	}
}

func InterpretPhrase(phrase Phrase) error {
	globalViolations = make(map[string][]Expression)
	currentInstances := make(map[string]*orderedmap.OrderedMap[uint64, Expression])
	currentNonInstances := make(map[string]*orderedmap.OrderedMap[uint64, Expression])

	for factName, instances := range globalInstances {
		currentInstances[factName] = orderedmap.New[uint64, Expression]()
		for pair := instances.Oldest(); pair != nil; pair = pair.Next() {
			currentInstances[factName].Set(pair.Key, pair.Value)
		}
	}

	for factName, instances := range globalNonInstances {
		currentNonInstances[factName] = orderedmap.New[uint64, Expression]()
		for pair := instances.Oldest(); pair != nil; pair = pair.Next() {
			currentNonInstances[factName].Set(pair.Key, pair.Value)
		}
	}

	globalResults = append(globalResults, PhraseResult{Success: true, Changes: []Phrase{}, Triggers: []Trigger{}, Violations: []Violation{}})

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
		globalResults[len(globalResults)-1].IsBquery = true
		err = handleBQuery(*phrase.Expression)
	case "iquery":
		globalResults[len(globalResults)-1].IsIquery = true
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
	case "extend":
		err = handleExtend(phrase)
	default:
		//err = fmt.Errorf("unknown phrase kind: %s", phrase.Kind)
	}

	// Queries can never influence the state
	if phrase.Kind == "bquery" || phrase.Kind == "iquery" {
		return nil
	}

	index := len(globalResults) - 1

	if derivationVersion == 1 {
		DeriveFacts()
	} else if derivationVersion == 2 {
		DeriveFacts2()
	} else if derivationVersion == 3 {
		DeriveFacts3()
	} else {
		panic("unknown derivation version")
	}

	listViolations()

	for factName, instances := range currentInstances {
		for pair := instances.Oldest(); pair != nil; pair = pair.Next() {
			if _, ok := globalInstances[factName].Get(pair.Key); !ok {
				expr := copyExpression(pair.Value)
				if _, ok := globalNonInstances[factName].Get(pair.Key); !ok {
					Println("~" + formatExpression(pair.Value))
					globalResults[index].Changes = append(globalResults[index].Changes, Phrase{
						Kind:    "obfuscate",
						Operand: &expr,
					})
				} else {
					Println("-" + formatExpression(pair.Value))
					globalResults[index].Changes = append(globalResults[index].Changes, Phrase{
						Kind:    "terminate",
						Operand: &expr,
					})
				}
			}
		}
	}

	for factName, instances := range globalInstances {
		for pair := instances.Oldest(); pair != nil; pair = pair.Next() {
			// Check if currentInstances contains the factName
			if _, ok := currentInstances[factName]; ok {
				if _, ok := currentInstances[factName].Get(pair.Key); ok {
					continue
				}
			}

			Println("+" + formatExpression(pair.Value))
			expr := copyExpression(pair.Value)
			globalResults[index].Changes = append(globalResults[index].Changes, Phrase{
				Kind:    "create",
				Operand: &expr,
			})
		}
	}

	return err
}

func handleExtend(phrase Phrase) error {
	name, ok := phrase.Name.(string)

	if !ok {
		panic("Error in name for extend")
	}
	if !factExists(name) {
		return fmt.Errorf("fact does not exist")
	}

	fact := globalState["facts"][name]

	if afact, ok := fact.(AtomicFact); ok {
		afact.DerivedFrom = append(afact.DerivedFrom, phrase.DerivedFrom...)
		afact.HoldsWhen = append(afact.HoldsWhen, phrase.HoldsWhen...)
		afact.ConditionedBy = append(afact.ConditionedBy, phrase.ConditionedBy...)

		globalState["facts"][name] = afact
	} else if cfact, ok := fact.(CompositeFact); ok {
		cfact.DerivedFrom = append(cfact.DerivedFrom, phrase.DerivedFrom...)
		cfact.HoldsWhen = append(cfact.HoldsWhen, phrase.HoldsWhen...)
		cfact.ConditionedBy = append(cfact.ConditionedBy, phrase.ConditionedBy...)

		if cfact.FactType == EventType || cfact.FactType == ActType {
			cfact.SyncsWith = append(cfact.SyncsWith, phrase.SyncsWith...)
			cfact.Creates = append(cfact.Creates, phrase.Creates...)
			cfact.Terminates = append(cfact.Terminates, phrase.Terminates...)
			cfact.Obfuscates = append(cfact.Obfuscates, phrase.Obfuscates...)
		}

		globalState["facts"][name] = cfact
	} else {
		panic("Error in fact type for extend")
	}

	return nil
}

func handlePredicate(phrase Phrase) error {
	// A predicate is a fact without parameters
	err := handleAtomicFact(Phrase{
		Name:        phrase.Name,
		HoldsWhen:   []Expression{*phrase.Expression},
		IsInvariant: phrase.IsInvariant,
	})

	if err != nil {
		return err
	}

	globalResults[len(globalResults)-1].Changes = []Phrase{phrase}

	return nil
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
		Obfuscates:    phrase.Obfuscates,
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
		Obfuscates:    phrase.Obfuscates,
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
			globalResults[len(globalResults)-1].Changes = []Phrase{phrase}
			return nil
		}
	} else {
		panic("placeholder name is not a list of strings")
	}
}

func fillParameters(expression Expression, params []string, values []Expression) Expression {
	newExpression := copyExpression(expression)
	err := TypeCheckExpression(&newExpression)
	if err != nil {
		panic(err)
	}

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
	for _, expr := range gatherExpressions(operand) {
		if expr.Identifier == "" {
			log.Println("Skipping non-identifier expression in trigger")
			continue
		}

		expr, err := convertInstance(expr)
		if err != nil {
			log.Println("Error in converting trigger instance")
			continue
		}

		Println("executed transition:")

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
						Println(formatExpression(expr), "(DISABLED)")
						addViolation("act", copyExpression(expr))
					} else {
						Println(formatExpression(expr), "(ENABLED)")
					}
				} else if cfact.FactType == EventType {
					Println(formatExpression(expr))
				} else if cfact.FactType == DutyType {
					Println("Triggering duty", cfact.Name)
				} else {
					log.Println("Fact is not triggerable")
					break
				}

				syncsWith := make([]Expression, 0)
				obfuscates := make([]Expression, 0)
				terminates := make([]Expression, 0)
				creates := make([]Expression, 0)

				for _, sync := range cfact.SyncsWith {
					syncsWith = append(syncsWith, gatherExpressions(fillParameters(sync, cfact.IdentifiedBy, expr.Operands))...)
				}

				for _, obfuscate := range cfact.Obfuscates {
					obfuscates = append(obfuscates, gatherExpressions(fillParameters(obfuscate, cfact.IdentifiedBy, expr.Operands))...)
				}

				for _, terminate := range cfact.Terminates {
					terminates = append(terminates, gatherExpressions(fillParameters(terminate, cfact.IdentifiedBy, expr.Operands))...)
				}

				for _, create1 := range cfact.Creates {
					creates = append(creates, gatherExpressions(fillParameters(create1, cfact.IdentifiedBy, expr.Operands))...)
				}

				for _, sync := range syncsWith {
					handleTrigger(sync)
				}

				for _, obfuscate := range obfuscates {
					handleObfuscate(obfuscate)
				}

				for _, terminate := range terminates {
					handleTerminate(terminate)
				}

				for _, create1 := range creates {
					create(create1, false)
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
	globalInstances[afact.Name] = orderedmap.New[uint64, Expression]()
	globalNonInstances[afact.Name] = orderedmap.New[uint64, Expression]()

	index := len(globalResults) - 1
	if index >= 0 {
		globalResults[index].Changes = []Phrase{fact}
		Println("New type", afact.Name)
	}

	return nil
}

func handleCompositeFact(fact Phrase) error {
	cfact := CompositeFact{
		Name:          fact.Name.(string),
		IdentifiedBy:  fact.IdentifiedBy,
		DerivedFrom:   fact.DerivedFrom,
		HoldsWhen:     fact.HoldsWhen,
		ConditionedBy: fact.ConditionedBy,
		SyncsWith:     fact.SyncsWith,
		Creates:       fact.Creates,
		Terminates:    fact.Terminates,
		Obfuscates:    fact.Obfuscates,
		ViolatedWhen:  fact.ViolatedWhen,
		FactType:      fact.FactType,
	}

	globalState["facts"][cfact.Name] = cfact

	// Initialise instances and non-instances for the composite fact
	globalInstances[cfact.Name] = orderedmap.New[uint64, Expression]()
	globalNonInstances[cfact.Name] = orderedmap.New[uint64, Expression]()

	globalResults[len(globalResults)-1].Changes = []Phrase{fact}
	Println("New type", cfact.Name)

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

func create(op Expression, derived bool) error {
	op, err := convertInstance(op)
	if err != nil {
		return err
	}

	op.IsDerived = derived

	hash, err := hashstructure.Hash(op, hashstructure.FormatV2, nil)

	if err != nil {
		panic(err)
	}

	if _, present := globalNonInstances[op.Identifier].Get(hash); present {
		if derived {
			return fmt.Errorf("cannot derive a non-instance")
		}

		globalNonInstances[op.Identifier].Delete(hash)
	}

	// Check if the instance already exists
	if instance, present := globalInstances[op.Identifier].Get(hash); present {
		if !derived {
			// Set the derived field to this instance to false, as it is now postulated.
			newExpr := instance
			newExpr.IsDerived = false
			globalInstances[op.Identifier].Set(hash, newExpr)

			return nil
		} else {
			return fmt.Errorf("instance already exists")
		}
	}

	globalInstances[op.Identifier].Set(hash, op)

	return nil
}

// handleCreate explicitly sets a given expression to true,
// by moving it from the non-instances to the instances list.
func handleCreate(operand Expression, derived bool) error {
	for _, op := range gatherExpressions(operand) {
		err := create(op, derived)
		if err != nil {
			// TODO: Handle error
		}
	}

	return nil
}

// handleTerminate explicitly sets a given expression to false,
// by moving it from the instances to the non-instances
// list.
func handleTerminate(operand Expression) error {
	for _, op := range gatherExpressions(operand) {
		op, err := convertInstance(op)
		if err != nil {
			return err
		}

		hash, err := hashstructure.Hash(op, hashstructure.FormatV2, nil)

		if err != nil {
			panic(err)
		}

		// If there is an instance for this expression, remove it
		if _, present := globalInstances[op.Identifier].Get(hash); present {
			globalInstances[op.Identifier].Delete(hash)
		}

		if _, present := globalNonInstances[op.Identifier].Get(hash); present {
			return fmt.Errorf("non-instance %s already exists", formatExpression(op))
		}

		globalNonInstances[op.Identifier].Set(hash, op)
	}

	return nil
}

// handleObfuscate implicitly sets a given expression to false,
// by removing it from both the instances and non-instances list.
func handleObfuscate(operand Expression) error {
	for _, op := range gatherExpressions(operand) {
		if op.Identifier == "" {
			log.Println("Skipping non-identifier expression", formatExpression(op))
			continue
		}

		op, err := convertInstance(op)
		if err != nil {
			return err
		}

		hash, err := hashstructure.Hash(op, hashstructure.FormatV2, nil)

		if err != nil {
			panic(err)
		}

		// If there is an instance for this expression, remove it
		if _, present := globalInstances[op.Identifier].Get(hash); present {
			globalInstances[op.Identifier].Delete(hash)
		}

		// If there is a non-instance for this expression, remove it
		if _, present := globalNonInstances[op.Identifier].Get(hash); present {
			globalNonInstances[op.Identifier].Delete(hash)
		}
	}

	return nil
}

func handleBQuery(expression Expression) error {
	instances := gatherExpressions(expression)

	if len(instances) != 1 {
		panic("multiple instances in handleBQuery")
	}

	instance := instances[0]
	result, err := evaluateInstance(instance)

	if err != nil {
		// TODO: Based on the kind of error, handle it differently
		panic(err)
	}

	globalResults[len(globalResults)-1].Result = result

	if result {
		Println("query successful")
	} else {
		Println("query failed")
	}

	return nil
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
		//log.Println("Infinite fact")
		go func() {
			for pair := globalInstances[factName].Oldest(); pair != nil; pair = pair.Next() {
				c <- ConstructorApplication{
					Identifier: pair.Value.Identifier,
					Operands:   pair.Value.Operands,
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
				result += ","
			}
			result += formatExpression(operand)
		}
		result += ")"
		return result
	} else if expression.Operator != "" {
		expr1 := formatExpression(expression.Operands[0])
		if expression.Operator == "NOT" {
			return "!" + expr1
		}
		expr2 := formatExpression(expression.Operands[1])

		switch expression.Operator {
		case "AND":
			return expr1 + " && " + expr2
		case "OR":
			return expr1 + " || " + expr2
		case "ADD":
			return expr1 + " + " + expr2
		case "SUB":
			return expr1 + " - " + expr2
		case "MUL":
			return expr1 + " * " + expr2
		case "DIV":
			return expr1 + " / " + expr2
		case "MOD":
			return expr1 + " % " + expr2
		case "LT":
			return expr1 + " < " + expr2
		case "GT":
			return expr1 + " > " + expr2
		case "GTE":
			return expr1 + " >= " + expr2
		case "LTE":
			return expr1 + " <= " + expr2
		case "EQ":
			return expr1 + " == " + expr2
		case "NEQ":
			return expr1 + " != " + expr2
		case "WHEN":
			return expr1 + " When " + expr2
		}
	} else if expression.Iterator != "" {
		keyword := expression.Iterator[:1] + strings.ToLower(expression.Iterator[1:])
		if expression.Iterator == "EXISTS" || expression.Iterator == "FOREACH" {
			return keyword + expression.Binds[0] + " : " + formatExpression(expression.Operands[0])
		}

		return keyword + "(" + formatExpression(expression.Operands[0]) + ")"
	} else if expression.Parameter != "" {
		return formatExpression(*expression.Operand) + "." + expression.Parameter
	}

	return ""
}

func handleIQuery(expression Expression) error {
	Println("?-" + formatExpression(expression))

	signal := make(chan struct{})

	results := make([]Expression, 0)
	errors := make([]Error, 0)

	for instance := range handleExpression(expression, signal) {
		if instance.Identifier == "" {
			panic("invalid instance in iquery result")
		}

		Println(formatExpression(instance))

		results = append(results, instance)

		signal <- struct{}{}
	}

	if len(errors) > 0 {
		globalResults[len(globalResults)-1].Success = false
		globalResults[len(globalResults)-1].Errors = errors
	} else {
		globalResults[len(globalResults)-1].Results = results
	}

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
			signal := make(chan struct{})
			defer close(signal)

			return handleExpression(instance, signal) != nil, nil
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
		//log.Println("Evaluating", formatExpression(instance))
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
		hash, err := hashstructure.Hash(instance, hashstructure.FormatV2, nil)
		if err != nil {
			panic(err)
		}
		if _, present := globalInstances[instance.Identifier].Get(hash); present {
			return true, nil
		}

		// Check if the instance is known to not exist
		if _, present := globalNonInstances[instance.Identifier].Get(hash); present {
			return false, nil
		}
	} else {
		log.Println("Don't know what to do with instance:", instance)
	}

	return false, nil
}

func gatherExpressions(expression Expression) []Expression {
	result := make([]Expression, 0)
	signal := make(chan struct{}, 1)
	defer close(signal)

	for instance := range handleExpression(expression, signal) {
		result = append(result, instance)

		signal <- struct{}{}
	}

	return result
}

// TODO: This can return any expression
func handleExpression(expression Expression, signal <-chan struct{}) <-chan Expression {
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
			signal2 := make(chan struct{}, 1)

			for instance := range iterateFact(ref) {
				// Replace all occurrences of the variable with the instance
				for _, occurrence := range occurrences {
					*occurrence = Expression{
						Identifier: instance.Identifier,
						Operands:   instance.Operands,
					}
				}

				for result := range handleExpression(copyExpression(expression), signal2) {
					c <- copyExpression(result)

					<-signal
					signal2 <- struct{}{}
				}
			}

			// Put the original expression back
			for _, occurrence := range occurrences {
				*occurrence = Expression{
					Value: []string{ref},
				}
			}

			close(signal2)
			close(c)
			select {
			case _, _ = <-signal:
			default:
				break
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

				<-signal
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
	} else if val, ok := expression.Value.(bool); ok {
		go func() {
			c <- Expression{
				Value: val,
			}
			close(c)
		}()
	} else if expression.Operator != "" {
		go func() {
			signal2 := make(chan struct{}, 1)

			for operand := range handleOperator(expression, signal2) {
				c <- operand

				<-signal
				signal2 <- struct{}{}
			}

			close(signal2)
			close(c)
		}()
	} else if expression.Identifier != "" {
		// TODO: Get all instances for the operands and return them

		//if len(expression.Operands) == 0 {
		//	panic("No operands for expression")
		//}

		signal2 := make(chan struct{}, 1)

		for i := range expression.Operands {
			// TODO: CHeck if this is correct (It is not!)
			expression.Operands[i], ok = <-handleExpression(expression.Operands[i], signal2)
			if !ok {
				close(signal2)
				close(c)
				return c
			}
		}

		go func() {
			// TODO: This is needed as we cannot always evaluate instances to true/false (citizen(Bob))
			c <- expression

			//log.Println("Sent expression")

			close(signal2)
			close(c)

			//log.Println("Closed channel")
			_, _ = <-signal
			//log.Println("Received optional signal")
		}()
	} else if expression.Iterator != "" {
		go func() {
			signal2 := make(chan struct{}, 1)

			for expr := range handleIterator(expression, signal2) {
				c <- expr

				<-signal
				signal2 <- struct{}{}
			}

			close(signal2)
			close(c)
		}()
	} else if expression.Parameter != "" {
		go func() {
			signal2 := make(chan struct{}, 1)

			for expr := range handleProjection(expression, signal2) {
				c <- expr

				<-signal
				signal2 <- struct{}{}
			}

			close(signal2)
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

func handleOperator(expression Expression, signal <-chan struct{}) <-chan Expression {
	c := make(chan Expression)

	if expression.Operator == "ADD" || expression.Operator == "SUB" || expression.Operator == "MUL" || expression.Operator == "DIV" || expression.Operator == "MOD" ||
		expression.Operator == "LT" || expression.Operator == "GT" || expression.Operator == "LTE" || expression.Operator == "GTE" {
		go func() {
			// TODO: Check if exactly two operands

			signal1 := make(chan struct{}, 1)
			defer close(signal1)

			expression1 := <-handleExpression(expression.Operands[0], signal1)
			expression2 := <-handleExpression(expression.Operands[1], signal1)

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
		signal1 := make(chan struct{})
		defer close(signal1)

		expr1 := <-handleExpression(expression.Operands[0], signal1)
		expr2 := <-handleExpression(expression.Operands[1], signal1)

		go func() {
			value := equalInstanceContents(expr1, expr2)
			if expression.Operator == "NEQ" {
				value = !value
			}

			c <- Expression{
				Value: value,
			}

			close(c)
		}()
	} else if expression.Operator == "AND" {
		go func() {
			signal1 := make(chan struct{})
			defer close(signal1)

			result := true
			for _, operand := range expression.Operands {
				expr := <-handleExpression(operand, signal1)
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
	} else if expression.Operator == "OR" {
		go func() {
			signal1 := make(chan struct{})
			defer close(signal1)

			result := false
			for _, operand := range expression.Operands {
				expr := <-handleExpression(operand, signal1)
				if eval, err := evaluateInstance(expr); err == nil {
					result = result || eval
				} else {
					panic(err)
				}

				if result {
					break
				}
			}

			c <- Expression{
				Value: result,
			}
		}()
	} else if expression.Operator == "NOT" {
		signal1 := make(chan struct{})
		defer close(signal1)

		expr := <-handleExpression(expression.Operands[0], signal1)

		go func() {
			if eval, err := evaluateInstance(expr); err == nil {
				if customDerivation {
					hash1, err := hashstructure.Hash(expr, hashstructure.FormatV2, nil)
					if err != nil {
						panic(err)
					}

					hash2, err := hashstructure.Hash(expr, hashstructure.FormatV2, nil)
					if err != nil {
						panic(err)
					}

					if hash1 == hash2 {
						//log.Println("Negating literal", formatExpression(expr))
						if _, present := globalNonInstances[expr.Identifier].Get(hash2); !present {
							//log.Println("Assuming negated literal", formatExpression(expr), "is false")
							// We assume that the instance does not exist
							//log.Println("Assuming that", formatExpression(expr), "does not exist")
							tempAssumptions = append(tempAssumptions, &Assumptions{
								Expression:  hash2,
								Knowledge:   copyKnowledge(),
								Assumptions: copyAssumptions(),
								Queue:       copyQueue(),
							})
						}
					}
				}

				c <- Expression{
					Value: !eval,
				}
			} else {
				panic(err)
			}

			close(c)
		}()
	} else if expression.Operator == "COUNT" {
		signal1 := make(chan struct{}, 1)

		go func() {
			length := int64(0)

			for range handleExpression(expression.Operands[0], signal1) {
				length++

				signal1 <- struct{}{}
			}

			c <- Expression{
				Value: length,
			}

			close(c)
			close(signal1)
		}()
	} else if expression.Operator == "WHEN" {
		signal1 := make(chan struct{})

		expr := <-handleExpression(expression.Operands[1], signal1)

		//log.Println("WHEN", formatExpression(expression.Operands[0]), expr)

		if eval, err := evaluateInstance(expr); err == nil && eval {
			go func() {
				for expr := range handleExpression(expression.Operands[0], signal1) {
					c <- expr

					<-signal
					signal1 <- struct{}{}
				}

				close(signal1)
				close(c)
			}()
		} else {
			//log.Println("When is false")
			close(c)
			close(signal1)
		}
	} else if expression.Operator == "MAX" || expression.Operator == "MIN" || expression.Operator == "SUM" {
		signal1 := make(chan struct{})
		go func() {
			value := int64(0)
			first := true

			for expr := range handleExpression(expression.Operands[0], signal1) {
				numb := instanceToInt(expr)

				if numb.Value == nil || reflect.TypeOf(numb.Value) != intType {
					panic("Cannot convert to int")
				}

				if expression.Operator == "MAX" && numb.Value.(int64) > value {
					value = numb.Value.(int64)
				} else if expression.Operator == "MIN" && (first || numb.Value.(int64) < value) {
					first = false
					value = numb.Value.(int64)
				} else if expression.Operator == "SUM" {
					value += numb.Value.(int64)
				}

				signal1 <- struct{}{}
			}

			c <- Expression{
				Value: value,
			}

			close(c)
			close(signal1)
		}()
	} else if expression.Operator == "HOLDS" {
		signal1 := make(chan struct{})
		defer close(signal1)
		expr1 := <-handleExpression(expression.Operands[0], signal1)

		go func() {
			if expr1.Identifier == "" {
				panic("Holds(t) requires t to evaluate to a an instance, not a literal")
			}
			eval, err := evaluateInstance(expr1)
			if err != nil {
				panic(err)
			}

			c <- Expression{
				Value: eval,
			}

			close(c)
		}()
	} else if expression.Operator == "ENABLED" {
		expr := expression.Operands[0]
		conditions := make([]Expression, 0)
		fact := globalState["facts"][expression.Operands[0].Identifier]
		if afact, ok := fact.(AtomicFact); ok {
			for _, condition := range afact.ConditionedBy {
				conditions = append(conditions, fillParameters(condition, []string{afact.Name}, []Expression{expr}))
			}
		} else if cfact, ok := fact.(CompositeFact); ok {
			for _, condition := range cfact.ConditionedBy {
				conditions = append(conditions, fillParameters(condition, cfact.IdentifiedBy, expr.Operands))
			}
		} else {
			panic("Unknown fact type")
		}

		signal1 := make(chan struct{})
		defer close(signal1)
		expr = <-handleExpression(Expression{
			Operator: "AND",
			Operands: append([]Expression{
				{
					Operator: "HOLDS",
					Operands: []Expression{expression.Operands[0]},
				},
			}, conditions...),
		}, signal1)

		go func() {
			eval, err := evaluateInstance(expr)
			if err != nil {
				panic(err)
			}

			c <- Expression{
				Value: eval,
			}

			close(c)
		}()
	} else {
		log.Println("Unknown operator", expression)
		panic("Unknown operator")
	}

	return c
}

func handleIterator(expression Expression, signal <-chan struct{}) <-chan Expression {
	c := make(chan Expression)

	if expression.Iterator == "FOREACH" {
		go func() {
			signal1 := make(chan struct{})
			defer close(signal1)

			for expr := range handleExpression(*expression.Expression, signal1) {
				c <- copyExpression(expr)

				<-signal
				signal1 <- struct{}{}
			}

			close(c)
		}()
	} else if expression.Iterator == "EXISTS" {
		go func() {
			signal1 := make(chan struct{})
			defer close(signal1)

			for expr := range handleExpression(*expression.Expression, signal1) {
				if eval, err := evaluateInstance(expr); err == nil {
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

				<-signal
				signal1 <- struct{}{}
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

func handleProjection(expression Expression, signal <-chan struct{}) <-chan Expression {
	//log.Println("Projection", expression.Parameter, expression.Operand)
	c := make(chan Expression)

	go func() {
		signal1 := make(chan struct{})
		defer close(signal1)

		for expr := range handleExpression(*expression.Operand, signal1) {
			if expr.Identifier == "" {
				panic("Cannot project non-identifier")
			}

			if !factExists(expr.Identifier) {
				panic("Cannot project non-existing fact")
			}

			fact := globalState["facts"][expr.Identifier]

			if cfact, ok := fact.(CompositeFact); ok {
				found := false

				for i, param := range cfact.IdentifiedBy {
					if param == expression.Parameter {
						c <- expr.Operands[i]

						<-signal
						signal1 <- struct{}{}

						found = true
						break
					}
				}

				if !found {
					close(c)
					panic("Expression has no parameter " + expression.Parameter)
				}
			} else {
				panic("Cannot project atomic fact")
			}
		}
	}()

	return c
}
