package eflint

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (i *Input) UnmarshalJSON(data []byte) error {
	type Alias Input
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var tempMap map[string]interface{}
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return err
	}

	var phrasesExpected bool
	switch aux.Kind {
	case "phrases":
		phrasesExpected = true
	case "handshake":
		phrasesExpected = false
	case "ping":
		phrasesExpected = false
	default:
		return fmt.Errorf("unknown kind: %s", aux.Kind)
	}

	if phrasesExpected {
		if _, ok := tempMap["phrases"]; !ok {
			return fmt.Errorf("missing field: phrases")
		}
	} else {
		if _, ok := tempMap["phrases"]; ok {
			return fmt.Errorf("unexpected field: phrases")
		}
	}

	i.Version = aux.Version
	i.Kind = aux.Kind
	i.Updates = aux.Updates
	i.Phrases = aux.Phrases

	return nil
}

func (p *Phrase) UnmarshalJSON(data []byte) error {
	type Alias Phrase
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch aux.Kind {
	case "bquery":
		fallthrough
	case "iquery":
		var q Query
		if err := json.Unmarshal(data, &q); err != nil {
			return err
		}
		p.Expression = &q.Expression
		p.WhenTrue = q.WhenTrue
	case "create":
		fallthrough
	case "terminate":
		fallthrough
	case "obfuscate":
		fallthrough
	case "trigger":
		var s Statement
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		p.Operand = &s.Operand
	case "afact":
		var af AtomicFact
		if err := json.Unmarshal(data, &af); err != nil {
			return err
		}
		p.Name = af.Name
		p.Type = af.Type
		p.Range = af.Range
		p.DerivedFrom = af.DerivedFrom
		p.HoldsWhen = af.HoldsWhen
		p.ConditionedBy = af.ConditionedBy
	case "cfact":
		var cf CompositeFact
		if err := json.Unmarshal(data, &cf); err != nil {
			return err
		}
		p.Name = cf.Name
		p.IdentifiedBy = cf.IdentifiedBy
		p.DerivedFrom = cf.DerivedFrom
		p.HoldsWhen = cf.HoldsWhen
		p.ConditionedBy = cf.ConditionedBy
	case "placeholder":
		var ph Placeholder
		if err := json.Unmarshal(data, &ph); err != nil {
			return err
		}
		p.Name = ph.Name
		p.For = ph.For
	case "predicate":
		var pred Predicate
		if err := json.Unmarshal(data, &pred); err != nil {
			return err
		}
		p.Name = pred.Name
		p.IsInvariant = pred.IsInvariant
		p.Expression = &pred.Expression
	case "event":
		var ev Event
		if err := json.Unmarshal(data, &ev); err != nil {
			return err
		}
		p.Name = ev.Name
		p.RelatedTo = ev.RelatedTo
		p.DerivedFrom = ev.DerivedFrom
		p.HoldsWhen = ev.HoldsWhen
		p.ConditionedBy = ev.ConditionedBy
		p.SyncsWith = ev.SyncsWith
		p.Creates = ev.Creates
		p.Terminates = ev.Terminates
		p.Obfuscates = ev.Obfuscates
	case "act":
		var act Act
		if err := json.Unmarshal(data, &act); err != nil {
			return err
		}
		p.Name = act.Name
		p.Actor = act.Actor
		p.RelatedTo = act.RelatedTo
		p.DerivedFrom = act.DerivedFrom
		p.HoldsWhen = act.HoldsWhen
		p.ConditionedBy = act.ConditionedBy
		p.SyncsWith = act.SyncsWith
		p.Creates = act.Creates
		p.Terminates = act.Terminates
		p.Obfuscates = act.Obfuscates
	case "duty":
		var duty Duty
		if err := json.Unmarshal(data, &duty); err != nil {
			return err
		}
		p.Name = duty.Name
		p.Holder = duty.Holder
		p.Claimant = duty.Claimant
		p.RelatedTo = duty.RelatedTo
		p.DerivedFrom = duty.DerivedFrom
		p.HoldsWhen = duty.HoldsWhen
		p.ConditionedBy = duty.ConditionedBy
		p.ViolatedWhen = duty.ViolatedWhen
	case "extend":
		var ext Extend
		if err := json.Unmarshal(data, &ext); err != nil {
			return err
		}
		p.ParentKind = ext.ParentKind
		p.Name = ext.Name
		p.DerivedFrom = ext.DerivedFrom
		p.HoldsWhen = ext.HoldsWhen
		p.ConditionedBy = ext.ConditionedBy
		p.SyncsWith = ext.SyncsWith
		p.Creates = ext.Creates
		p.Terminates = ext.Terminates
		p.Obfuscates = ext.Obfuscates
	default:
		return fmt.Errorf("unknown kind: " + aux.Kind)
	}

	p.Kind = aux.Kind
	p.Stateless = aux.Stateless
	p.Updates = aux.Updates

	return nil
}

func (e *Expression) UnmarshalJSON(data []byte) error {
	var Primitive Primitive
	if err := json.Unmarshal(data, &Primitive); err == nil {
		//log.Println("Primitive", Primitive)
		e.Value = Primitive.Value
		return nil
	}

	var VariableReference []string
	if err := json.Unmarshal(data, &VariableReference); err == nil {
		//log.Println("VariableReference", VariableReference)
		if len(VariableReference) == 1 {
			e.Value = VariableReference
			return nil
		}
	}

	var ConstructorApplication ConstructorApplication
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ConstructorApplication); err == nil {
		//log.Println("ConstructorApplication", ConstructorApplication)
		e.Identifier = ConstructorApplication.Identifier
		e.Operands = ConstructorApplication.Operands
		return nil
	}

	var Operator Operator
	dec = json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&Operator); err == nil {
		//log.Println("Operator", Operator)
		e.Operator = Operator.Operator
		e.Operands = Operator.Operands
		return nil
	}

	var Iterator Iterator
	dec = json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&Iterator); err == nil {
		//log.Println("Iterator", Iterator)
		e.Iterator = Iterator.Iterator
		e.Binds = Iterator.Binds
		e.Expression = &Iterator.Expression
		return nil
	}

	var Projection Projection
	dec = json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&Projection); err == nil {
		//log.Println("Projection", Projection.Operand)
		e.Parameter = Projection.Parameter
		e.Operand = Projection.Operand
		return nil
	}

	return fmt.Errorf("unknown expression type")
}

func (p *Primitive) UnmarshalJSON(data []byte) error {
	var String string
	if err := json.Unmarshal(data, &String); err == nil {
		p.Value = String
		return nil
	}

	var Number float64
	if err := json.Unmarshal(data, &Number); err == nil {
		if Number == float64(int64(Number)) {
			p.Value = int64(Number)
			return nil
		} else {
			return fmt.Errorf("float64 not supported")
		}
	}

	var Boolean bool
	if err := json.Unmarshal(data, &Boolean); err == nil {
		p.Value = Boolean
		return nil
	}

	return fmt.Errorf("unknown primitive type")
}

func GenerateHandshake() ([]byte, error) {
	return json.Marshal(Handshake{
		Success:           true,
		SupportedVersions: SupportedVersions,
		Reasoner:          Reasoner,
		ReasonerVersion:   ReasonerVersion,
		SharesUpdates:     true,
		SharesTriggers:    true,
		SharesViolations:  false,
	})
}

func (e Expression) MarshalJSON() ([]byte, error) {
	if e.Value != nil {
		return json.Marshal(e.Value)
	}

	type Alias Expression
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&e),
	})
}

func (p PhraseResult) MarshalJSON() ([]byte, error) {

	if p.IsBquery {
		return json.Marshal(&BQueryResult{
			Success: p.Success,
			Errors:  p.Errors,
			Result:  p.Result,
		})
	} else if p.IsIquery {
		return json.Marshal(&IQueryResult{
			Success: p.Success,
			Errors:  p.Errors,
			Result:  p.Results,
		})
	}

	return json.Marshal(&StateChanges{
		Success:    p.Success,
		Changes:    p.Changes,
		Triggers:   p.Triggers,
		Violated:   p.Violated,
		Violations: p.Violations,
	})
}

// GenerateJSON generates JSON from the given struct
// If it fails, it returns an error
func GenerateJSON(output Output) ([]byte, error) {
	if len(globalErrors) > 0 {
		output.Success = false
		output.Errors = globalErrors
	} else {
		output.Results = globalResults
	}

	result, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}
