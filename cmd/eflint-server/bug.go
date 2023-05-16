package main

import (
	"log"
)

type Expression struct {
	Value      interface{}   `json:"value,omitempty"`
	Operator   string        `json:"operator,omitempty"`
	Identifier string        `json:"identifier,omitempty"`
	Operands   []*Expression `json:"operands,omitempty"`
}

// Finds a variable within an expression, seems to work fine.
func findVariable(expression Expression) string {
	if expression.Value != nil {
		if ref, ok := expression.Value.([]string); ok {
			if len(ref) == 1 {
				return ref[0]
			}
		}
	} else if expression.Identifier != "" || expression.Operator != "" {
		for _, operand := range expression.Operands {
			if variable := findVariable(*operand); variable != "" {
				return variable
			}
		}
	}

	return ""
}

func findOccurrences(expression *Expression, variable string) []*Expression {
	if expression.Value != nil {
		if ref, ok := expression.Value.([]string); ok {
			if len(ref) == 1 && ref[0] == variable {
				// expression.Value = "TEST"
				return []*Expression{expression}
			}
		}
	} else if expression.Identifier != "" || expression.Operator != "" {
		var result []*Expression
		for _, operand := range expression.Operands {
			result = append(result, findOccurrences(operand, variable)...)
		}
		return result
	}

	return []*Expression{}
}

func handleExpression(expression Expression) {
	// Check if there are any variables in the expression
	ref := findVariable(expression)
	if ref != "" {
		// Find all occurrences of the variable
		occurrences := findOccurrences(&expression, ref)

		log.Println(occurrences, &expression)

		for i := 1; i <= 10; i++ {
			// Create a copy of the expression
			newExpression := expression

			// Replace all occurrences with the new value
			for _, occurrence := range occurrences {
				occurrence.Value = i
			}

			// Print the new expression
			log.Println("New expression:", newExpression, *newExpression.Operands[0])
		}

		log.Println("New expression:", expression, expression.Operands[0])
	}
}

func main() {
	// This is the example we tested with.
	test := Expression{
		Identifier: "x",
		Operands: []*Expression{
			{
				Value: []string{"ten"},
			},
		},
	}

	handleExpression(test)
}
