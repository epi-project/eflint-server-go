package eflint

type Input struct {
	Version string      `json:"version"`
	Kind    string      `json:"kind"`
	Phrases interface{} `json:"phrases"`
	Updates bool        `json:"updates"`
}

// A phrase is one of 3 types:
// - A query
// - A statement
// - A definition

type Phrase struct {
	Kind      string      `json:"kind"`
	Stateless bool        `json:"stateless"`
	Updates   bool        `json:"updates"`
	Phrase    interface{} `json:"phrase"`
}

type Query struct {
	Expression interface{} `json:"expression"`
}

type Fact struct {
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Range         []interface{} `json:"range"`
	DerivedFrom   []interface{} `json:"derived-from"`
	HoldsWhen     []interface{} `json:"holds-when"`
	ConditionedBy []interface{} `json:"conditioned-by"`
}

// Expression is one of 5 types:
// - Primitives

type Primitive struct {
	Value interface{} `json:"value"`
}

// - Variable references

type VariableReference struct {
	Value interface{} `json:"value"`
}

// - Constructor applications

type ConstructorApplication struct {
	Identifier string        `json:"identifier"`
	Operands   []interface{} `json:"operands"`
}

// - Operators

type Operator struct {
	Operator string        `json:"operator"`
	Operands []interface{} `json:"operands"`
}

// - Iterators

type Iterator struct {
	Iterator  string      `json:"iterator"`
	Binds     []string    `json:"binds"`
	Predicate interface{} `json:"expression"`
}

type Output struct {
	Success bool          `json:"success"`
	Phrases []interface{} `json:"phrases,omitempty"`
}

type Handshake struct {
	SupportedVersions []string `json:"supported_versions"`
	Reasoner          string   `json:"reasoner,omitempty"`
	ReasonerVersion   string   `json:"reasoner_version,omitempty"`
	SharesUpdates     bool     `json:"shares_updates,omitempty"`
	SharesTriggers    bool     `json:"shares_triggers,omitempty"`
	SharesViolations  bool     `json:"shares_violations,omitempty"`
}

// Possible information for a phrase (besides the shared fields)
// - Expression
// - Operand (which is an expression?)
// - Name
// - Identifier
