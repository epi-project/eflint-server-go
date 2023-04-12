package eflint

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

func mapParseField(input map[string]interface{}, field string, expected reflect.Type, required bool, template interface{}) (interface{}, error) {
	// Check if the field is present
	if _, ok := input[field]; !ok {
		if required {
			return nil, errors.New("field " + field + " is required")
		} else {
			return template, nil
		}
	}

	// Check if the field can be converted to the expected type
	if reflect.TypeOf(input[field]).ConvertibleTo(expected) {
		return input[field], nil
	} else {
		return nil, errors.New("field " + field + " is not of type " + expected.String())
	}
}

// ParseInput parses the given io.Reader as JSON into the Input struct. If
// it fails, it returns an error
func ParseInput(r io.Reader) (Input, error) {
	var input Input
	err := json.NewDecoder(r).Decode(&input)
	if err != nil {
		return Input{}, err
	}

	switch input.Kind {
	case "ping":
	case "handshake":
		// Ensure no extra fields are present
		if input.Phrases != nil || input.Updates {
			return Input{}, errors.New("handshake should not contain phrases or updates")
		}
		break
	case "phrases":
		// Ensure phrases are present
		if input.Phrases == nil {
			return Input{}, errors.New("phrases should contain phrases")
		}

		// Set updates to false if it is not present
		if !input.Updates {
			input.Updates = false
		}

		phrases, err := ParsePhrases(input.Phrases)

		if err != nil {
			return Input{}, err
		}

		input.Phrases = phrases
	case "inspect":
		return Input{}, errors.New("inspect is not supported yet")
	default:
		return Input{}, errors.New("unknown kind")
	}

	return input, nil
}

func ParsePhrases(rawPhrases interface{}) ([]Phrase, error) {
	if rawPhrases == nil {
		return nil, errors.New("phrases should contain phrases")
	} else if _, ok := rawPhrases.([]interface{}); !ok {
		return nil, errors.New("phrases should be an array")
	}
	var phrases []Phrase

	// Step 1: Parse each phrase as a Phrase struct
	for _, phrase := range rawPhrases.([]interface{}) {
		var p Phrase

		if _, ok := phrase.(map[string]interface{}); !ok {
			return nil, errors.New("phrases should be an array of objects")
		}

		phrase := phrase.(map[string]interface{})

		// Try to parse the kind string (required)
		kind, err := mapParseField(phrase, "kind", reflect.TypeOf(""), true, nil)
		if err != nil {
			return nil, err
		}
		p.Kind = kind.(string)

		// Try to parse the stateless bool (optional)
		stateless, err := mapParseField(phrase, "stateless", reflect.TypeOf(true), false, false)
		if err != nil {
			return nil, err
		}
		p.Stateless = stateless.(bool)

		// Try to parse the updates bool (optional)
		updates, err := mapParseField(phrase, "updates", reflect.TypeOf(true), false, false)
		if err != nil {
			return nil, err
		}
		p.Updates = updates.(bool)

		// Parse the phrase itself
		newPhrase, err := ParsePhrase(p.Kind, phrase)

		if err != nil {
			return nil, err
		}

		p.Phrase = newPhrase
		phrases = append(phrases, p)
	}

	return phrases, nil
}

func ParsePhrase(kind string, phrase map[string]interface{}) (interface{}, error) {
	switch kind {
	case "bquery":
		fallthrough
	case "iquery":
		return ParseQuery(phrase)
	case "create":
		fallthrough
	case "terminate":
		fallthrough
	case "obfuscate":
		fallthrough
	case "trigger":
		return parseStatement(phrase)
	case "afact":
		return parseFact(phrase)
	default:
		return nil, errors.New("unknown kind")
	}
}

func ParseQuery(phrase map[string]interface{}) (interface{}, error) {
	// Ensure that the phrase contains an expression
	expression, err := mapParseField(phrase, "expression", objectType, true, nil)
	if err != nil {
		return nil, err
	}
	return parseExpression(expression.(map[string]interface{}))
}

func parseStatement(phrase map[string]interface{}) (interface{}, error) {
	// Ensure that the phrase contains an operand
	if operand, ok := phrase["operand"]; ok {
		return parseExpression(operand)
	} else {
		return nil, errors.New("operand is required for create")
	}

	// TODO: Should check the JSON contains nothing else
}

// TODO: Abstract this into parsers for each field, as they will be reused
func parseFact(phrase map[string]interface{}) (interface{}, error) {
	var fact Fact

	// Fact has a name field (required)
	if name, ok := phrase["name"]; ok {
		if _, ok := name.(string); !ok {
			return nil, errors.New("name should be a string")
		}
		fact.Name = name.(string)
	} else {
		return nil, errors.New("name is required")
	}

	// Fact has a type field (optional, defaults to string)
	if factType, ok := phrase["type"]; ok {
		switch factType {
		case "String":
			fallthrough
		case "Int":
			fact.Type = factType.(string)
		default:
			return nil, errors.New("type should be one of String or Int")
		}
	} else {
		fact.Type = "String"
	}

	// Fact has a range field (optional)
	if range_, ok := phrase["range"]; ok {
		if _, ok := range_.([]interface{}); !ok {
			return nil, errors.New("range should be an array")
		}

		for _, value := range range_.([]interface{}) {
			expression, err := parseExpression(value)

			if err != nil {
				return nil, err
			} else if _, ok := expression.(Primitive); !ok {
				return nil, errors.New("range can only contain primitives")
			}

			fact.Range = append(fact.Range, expression)
		}
	}

	// Fact has a derived-from field (optional)
	if derivedFrom, ok := phrase["derived-from"]; ok {
		// Make sure the derived-from is a map
		if _, ok := derivedFrom.([]interface{}); !ok {
			return nil, errors.New("derived-from should be an array")
		}

		for _, value := range derivedFrom.([]interface{}) {
			expression, err := parseExpression(value)

			if err != nil {
				return nil, err
			}

			fact.DerivedFrom = append(fact.DerivedFrom, expression)
		}
	} else {
		fact.DerivedFrom = []interface{}{}
	}

	// Fact has a holds-when field (optional)
	if holdsWhen, ok := phrase["holds-when"]; ok {
		// Make sure holds-when is an array
		if _, ok := holdsWhen.([]interface{}); !ok {
			return nil, errors.New("holds-when should be an array")
		}

		for _, value := range holdsWhen.([]interface{}) {
			expression, err := parseExpression(value)

			// TODO: Check that the expression is a boolean operator
			// For now, just check that it's an operator
			if err != nil {
				return nil, err
			} else if _, ok := expression.(Operator); !ok {
				return nil, errors.New("holds-when can only contain operators")
			}

			fact.HoldsWhen = append(fact.HoldsWhen, expression)
		}
	}

	// Fact has a conditioned-by field (optional)
	if conditionedBy, ok := phrase["conditioned-by"]; ok {
		// Make sure conditioned-by is an array
		if _, ok := conditionedBy.([]interface{}); !ok {
			return nil, errors.New("conditioned-by should be an array")
		}

		for _, value := range conditionedBy.([]interface{}) {
			expression, err := parseExpression(value)

			// TODO: Check that the expression is a boolean operator
			// For now, just check that it's an operator
			if err != nil {
				return nil, err
			} else if _, ok := expression.(Operator); !ok {
				return nil, errors.New("conditioned-by can only contain operators")
			}

			fact.ConditionedBy = append(fact.ConditionedBy, expression)
		}
	} else {
		fact.ConditionedBy = []interface{}{}
	}

	return fact, nil
}

func parseName(phrase map[string]interface{}) (interface{}, error) {
	// Ensure that the phrase contains a name
	if name, ok := phrase["name"]; ok {
		return parseExpression(name)
	} else {
		return nil, errors.New("name is required")
	}
}

//func parseIdentifiedBy(phrase map[string]interface{}) (interface{}, error) {
//	// Check if the phrase has an identified-by field
//	if identifiedBy, ok := phrase["identified-by"]; ok {
//		// Make sure identified-by is an array
//		if _, ok := identifiedBy.([]interface{}); !ok {
//			return nil, errors.New("identified-by should be an array")
//		}
//
//		for _, value := range identifiedBy.([]interface{}) {
//			expression, err := parseExpression(value)
//
//			if err != nil {
//				return nil, err
//			}
//
//		}
//	}
//}

func parseExpression(expression interface{}) (interface{}, error) {
	// Primitives and references can only occur in Operands

	// Check if the expression is a primitive
	switch expression.(type) {
	case string:
		return Primitive{Value: expression.(string)}, nil
	case bool:
		return Primitive{Value: expression.(bool)}, nil
	case float64:
		// Check if the float is an integer
		if expression != float64(int64(expression.(float64))) {
			return nil, errors.New("floats are not supported")
		}

		return Primitive{Value: int64(expression.(float64))}, nil
	case []string:
		{
			if len(expression.([]string)) != 1 {
				return nil, errors.New("string reference can only contain one value")
			}

			return VariableReference{Value: expression.([]string)[0]}, nil
		}
	case []int:
		{
			if len(expression.([]int)) != 1 {
				return nil, errors.New("integer reference can only contain one value")
			}

			return VariableReference{Value: expression.([]int)[0]}, nil
		}
	case []bool:
		{
			if len(expression.([]bool)) != 1 {
				return nil, errors.New("boolean reference can only contain one value")
			}

			return VariableReference{Value: expression.([]bool)[0]}, nil
		}
	case map[string]interface{}:
		break
	default:
		return nil, errors.New("unknown expression type")
	}

	expressionMap := expression.(map[string]interface{})

	// Check if the expression is a constructor application
	// It is one if it contains an "identifier" field
	identifier, err := mapParseField(expressionMap, "identifier", stringType, false, nil)
	if err != nil {
		return nil, err
	}
	if identifier != nil {
		operands, err := mapParseField(expressionMap, "operands", arrayType, true, nil)
		if err != nil {
			return nil, err
		}

		// Parse the operands
		parsedOperands, err := parseOperands(operands.([]interface{}))

		if err != nil {
			return nil, err
		}

		return ConstructorApplication{Identifier: identifier.(string), Operands: parsedOperands}, nil
	}

	// Check if the expression is an operator
	operator, err := mapParseField(expressionMap, "operator", stringType, false, nil)
	if err != nil {
		return nil, err
	}
	if operator != nil {
		operands, err := mapParseField(expressionMap, "operands", arrayType, true, nil)
		if err != nil {
			return nil, err
		}

		// Parse the operands
		parsedOperands, err := parseOperands(operands.([]interface{}))

		if err != nil {
			return nil, err
		}

		return Operator{Operator: operator.(string), Operands: parsedOperands}, nil
	}

	// Check if the expression is an iterator
	iterator, err := mapParseField(expressionMap, "iterator", stringType, false, nil)
	if err != nil {
		return nil, err
	}
	if iterator != nil {
		binds, err := mapParseField(expressionMap, "binds", arrayType, true, nil)
		if err != nil {
			return nil, err
		}

		predicate, err := mapParseField(expressionMap, "predicate", nil, true, nil)
		if err != nil {
			return nil, err
		}

		// Parse the predicate
		parsedPredicate, err := parseExpression(predicate)

		if err != nil {
			return nil, err
		}

		return Iterator{Iterator: iterator.(string), Binds: binds.([]string), Predicate: parsedPredicate}, nil
	}

	return nil, errors.New("unknown expression type")
}

func parseOperands(expression []interface{}) ([]interface{}, error) {
	operands := make([]interface{}, 0)

	for _, operand := range expression {
		newOperand, err := parseExpression(operand)

		if err != nil {
			return nil, err
		}

		operands = append(operands, newOperand)
	}

	return operands, nil
}

// GenerateJSON generates JSON from the given struct
// If it fails, it returns an error
func GenerateJSON(output Output) ([]byte, error) {
	// Filter out empty fields in the output.Phrases

	result, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}
