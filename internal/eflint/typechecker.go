package eflint

func isSupportedVersion(version string) bool {
	for _, supportedVersion := range SupportedVersions {
		if version == supportedVersion {
			return true
		}
	}
	return false
}

// Typecheck checks that the input is valid.
func Typecheck(input Input) error {
	// Check if the input version is supported
	if !isSupportedVersion(input.Version) {
		return ErrUnsupportedVersion
	}

	switch input.Kind {
	case "phrases":
		return TypecheckPhrases(input.Phrases)
	case "ping":
		fallthrough
	case "handshake":
		// Check if the input is empty
		if len(input.Phrases) != 0 || input.Updates {
			return ErrUnsupportedFields
		}
		return nil
	// TODO: Add the "inspect" kind
	default:
		return ErrUnknownKind
	}
}

// TypecheckPhrases goes over all the phrases in the input and checks that
// the types of the expressions are correct.
func TypecheckPhrases(phrases []Phrase) error {
	for _, phrase := range phrases {
		err := TypecheckPhrase(phrase)
		if err != nil {
			return err
		}
	}
	return nil
}

// TypecheckPhrase checks that the types of the expressions in the phrase are
// correct.
func TypecheckPhrase(phrase Phrase) error {
	switch phrase.Kind {
	case "bquery":
		return TypecheckBquery(phrase)
	case "iquery":
		return TypecheckIquery(phrase)
	case "create":
		return TypecheckCreate(phrase)
	case "terminate":
		return TypecheckTerminate(phrase)
	case "obfuscate":
		return TypecheckObfuscate(phrase)
	case "trigger":
		return TypecheckTrigger(phrase)
	case "afact":
		return TypecheckAfact(phrase)
	case "cfact":
		return TypecheckCfact(phrase)
	case "placeholder":
		return TypecheckPlaceholder(phrase)
	case "predicate":
		return TypecheckPredicate(phrase)
	case "event":
		return TypecheckEvent(phrase)
	case "act":
		return TypecheckAct(phrase)
	case "duty":
		return TypecheckDuty(phrase)
	case "extend":
		return TypecheckExtend(phrase)
	default:
		return ErrUnknownKind
	}
}

// TypecheckBquery checks that the types of the expressions in the bquery are
// correct.
func TypecheckBquery(phrase Phrase) error {
	return nil
}

// TypecheckIquery checks that the types of the expressions in the iquery are
// correct.
func TypecheckIquery(phrase Phrase) error {
	return nil
}

// TypecheckCreate checks that the types of the expressions in the create are
// correct.
func TypecheckCreate(phrase Phrase) error {
	return nil
}

// TypecheckTerminate checks that the types of the expressions in the terminate
// are correct.
func TypecheckTerminate(phrase Phrase) error {
	return nil
}

// TypecheckObfuscate checks that the types of the expressions in the obfuscate
// are correct.
func TypecheckObfuscate(phrase Phrase) error {
	return nil
}

// TypecheckTrigger checks that the types of the expressions in the trigger are
// correct.
func TypecheckTrigger(phrase Phrase) error {
	return nil
}

// TypecheckAfact checks that the types of the expressions in the afact are
// correct.
func TypecheckAfact(phrase Phrase) error {
	// Default the name to "String"
	if phrase.Name.(string) == "" {
		phrase.Name = "String"
	}

	// Check if range is given. If so, check its type.
	//if phrase.Range != nil {
	//	for _, expr := range phrase.Range {
	//		if phrase.Name.(string) == "String" {
	//			if _, ok := expr.Value.(string); !ok {
	//				return fmt.Errorf("range of atomic fact %s must be string", phrase.Name.(string))
	//			}
	//		} else if phrase.Name.(string) == "Int" {
	//			if _, ok := expr.Value.(int); !ok {
	//				return fmt.Errorf("range of atomic fact %s must be int", phrase.Name.(string))
	//			}
	//		} else {
	//			// TODO: This should be handled somewhere else
	//			return fmt.Errorf("unknown type %s", phrase.Name.(string))
	//		}
	//	}
	//}

	return nil
}

// TypecheckCfact checks that the types of the expressions in the cfact are
// correct.
func TypecheckCfact(phrase Phrase) error {
	return nil
}

// TypecheckPlaceholder checks that the types of the expressions in the
// placeholder are correct.
func TypecheckPlaceholder(phrase Phrase) error {
	return nil
}

// TypecheckPredicate checks that the types of the expressions in the predicate
// are correct.
func TypecheckPredicate(phrase Phrase) error {
	return nil
}

// TypecheckEvent checks that the types of the expressions in the event are
// correct.
func TypecheckEvent(phrase Phrase) error {
	return nil
}

// TypecheckAct checks that the types of the expressions in the act are correct.
func TypecheckAct(phrase Phrase) error {
	return nil
}

// TypecheckDuty checks that the types of the expressions in the duty are
// correct.
func TypecheckDuty(phrase Phrase) error {
	return nil
}

// TypecheckExtend checks that the types of the expressions in the extend are
// correct.
func TypecheckExtend(phrase Phrase) error {
	return nil
}

func TypeCheckExpressions(expressions *[]Expression) error {
	for i := range *expressions {
		err := TypeCheckExpression(&(*expressions)[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func TypeCheckExpression(expression *Expression) error {
	if len(expression.Operands) > 0 {
		err := TypeCheckExpressions(&expression.Operands)
		if err != nil {
			return err
		}
	}

	if expression.Identifier != "" && len(expression.Operands) == 0 {
		if !factExists(expression.Identifier) {
			//log.Println(expression)
			panic("Fact does not exist in typecheck")
		}

		fact := globalState["facts"][expression.Identifier]

		if cfact, ok := fact.(CompositeFact); ok {
			for _, param := range cfact.IdentifiedBy {
				expression.Operands = append(expression.Operands, Expression{Value: []string{param}})
			}
		}
	}

	return nil
}
