package eflint

type Input struct {
	Version string   `json:"version"`
	Kind    string   `json:"kind"`
	Phrases []Phrase `json:"phrases"`
	Updates bool     `json:"updates"`
}

// A phrase is one of 3 types:
// - A query
// - A statement
// - A definition

type Phrase struct {
	// General fields
	Kind      string `json:"kind"`
	Stateless bool   `json:"stateless,omitempty"`
	Updates   bool   `json:"updates,omitempty"`

	// Query fields
	Expression *Expression `json:"expression,omitempty"`

	// Statement fields
	Operand *Expression `json:"operand,omitempty"`

	// Definition fields
	Name          interface{}  `json:"name,omitempty"`
	Type          string       `json:"type,omitempty"`
	Range         []Expression `json:"range,omitempty"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
	IdentifiedBy  []string     `json:"identified-by,omitempty"`
	For           string       `json:"for,omitempty"`
	IsInvariant   bool         `json:"is-invariant,omitempty"`
	RelatedTo     []string     `json:"related-to,omitempty"`
	SyncsWith     []Expression `json:"syncs-with,omitempty"`
	Creates       []Expression `json:"creates,omitempty"`
	Terminates    []Expression `json:"terminates,omitempty"`
	Obfuscates    []Expression `json:"obfuscates,omitempty"`
	Actor         string       `json:"actor,omitempty"`
	Holder        string       `json:"holder,omitempty"`
	Claimant      string       `json:"claimant,omitempty"`
	ViolatedWhen  *Expression  `json:"violated-when,omitempty"`
	ParentKind    string       `json:"parent-kind,omitempty"`
}

type Query struct {
	Expression Expression `json:"expression"`
}

type Statement struct {
	Operand Expression `json:"operand"`
}

type AtomicFact struct {
	Name          string       `json:"name,omitempty"`
	Type          string       `json:"type,omitempty"`
	Range         []Expression `json:"range,omitempty"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
}

type CompositeFact struct {
	Name          string       `json:"name"`
	IdentifiedBy  []string     `json:"identified-by"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
}

type Placeholder struct {
	Name []string `json:"name"`
	For  string   `json:"for"`
}

type Predicate struct {
	Name        string     `json:"name"`
	IsInvariant bool       `json:"is-invariant,omitempty"`
	Expression  Expression `json:"expression"`
	Status      bool       `json:"status,omitempty"`
}

type Event struct {
	Name          string       `json:"name"`
	RelatedTo     []string     `json:"related-to,omitempty"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
	SyncsWith     []Expression `json:"syncs-with,omitempty"`
	Creates       []Expression `json:"creates,omitempty"`
	Terminates    []Expression `json:"terminates,omitempty"`
	Obfuscates    []Expression `json:"obfuscates,omitempty"`
}

type Act struct {
	Name          string       `json:"name"`
	Actor         string       `json:"actor,omitempty"`
	RelatedTo     []string     `json:"related-to,omitempty"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
	SyncsWith     []Expression `json:"syncs-with,omitempty"`
	Creates       []Expression `json:"creates,omitempty"`
	Terminates    []Expression `json:"terminates,omitempty"`
	Obfuscates    []Expression `json:"obfuscates,omitempty"`
}

type Duty struct {
	Name          string       `json:"name"`
	Holder        string       `json:"holder"`
	Claimant      string       `json:"claimant"`
	RelatedTo     []string     `json:"related-to,omitempty"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty"`
	ViolatedWhen  Expression   `json:"violated-when"`
}

type Extend struct {
	ParentKind string `json:"parent-kind"`
	Name       string `json:"name"`
}

// Expression is one of 5 types:
// - Primitives

type Expression struct {
	Value      interface{}  `json:"value,omitempty"`
	Operator   string       `json:"operator,omitempty"`
	Identifier string       `json:"identifier,omitempty"`
	Operands   []Expression `json:"operands,omitempty"`
	Iterator   string       `json:"iterator,omitempty"`
	Binds      []string     `json:"binds,omitempty"`
	Expression *Expression  `json:"expression,omitempty"`
}

type Primitive struct {
	Value interface{} `json:"value"`
}

// - Variable references

type VariableReference struct {
	Value []string `json:"value"`
}

// - Constructor applications

type ConstructorApplication struct {
	Identifier string       `json:"identifier"`
	Operands   []Expression `json:"operands"`
}

// - Operators

type Operator struct {
	Operator string       `json:"operator"`
	Operands []Expression `json:"operands"`
}

// - Iterators

type Iterator struct {
	Iterator   string     `json:"iterator"`
	Binds      []string   `json:"binds"`
	Expression Expression `json:"expression"`
}

// Triggers and Violations

type Trigger struct {
	Identifier string `json:"identifier"`
	Kind       string `json:"kind"`
	Parent     string `json:"parent"`
}

type Violation struct {
	Identifier string `json:"identifier"`
	Kind       string `json:"kind"`
}

type Output struct {
	Success bool          `json:"success"`
	Errors  []Error       `json:"errors,omitempty"`
	Results []interface{} `json:"results,omitempty"`
	Phrases []Phrase      `json:"phrases,omitempty"`
}

type Error struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type BQueryResult struct {
	Success bool    `json:"success"`
	Errors  []Error `json:"errors,omitempty"`
	Result  bool    `json:"result"`
}

type IQueryResult struct {
	Success bool         `json:"success"`
	Errors  []Error      `json:"errors,omitempty"`
	Result  []Expression `json:"result"`
}

type StateChanges struct {
	Success    bool        `json:"success"`
	Changes    []Phrase    `json:"changes,omitempty"`
	Triggers   []Trigger   `json:"triggers,omitempty"`
	Violated   bool        `json:"violated"`
	Violations []Violation `json:"violations,omitempty"`
}

type Result struct {
	Success bool    `json:"success"`
	Errors  []Error `json:"errors,omitempty"`
}

type Handshake struct {
	Success           bool     `json:"success"`
	SupportedVersions []string `json:"supported_versions"`
	Reasoner          string   `json:"reasoner"`
	ReasonerVersion   string   `json:"reasoner_version"`
	SharesUpdates     bool     `json:"shares_updates"`
	SharesTriggers    bool     `json:"shares_triggers"`
	SharesViolations  bool     `json:"shares_violations"`
}

type Value interface {
	int64 | string
}
