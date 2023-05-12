package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"os"
)

var (
	eflintLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{`FactID`, `[a-z][a-z_-]*`},
		{`Fact`, `Fact`},
		{`StringType`, `String`},
		{`IntType`, `Int`},

		{`IdentifiedBy`, `Identified by`},
		{`DerivedFrom`, `Derived from`},
		{`HoldsWhen`, `Holds when`},
		{`ConditionedBy`, `Conditioned by`},
		{`Placeholder`, `Placeholder`},
		{`Predicate`, `Predicate`},
		{`Invariant`, `Invariant`},
		{`Event`, `Event`},
		{`Duty`, `Duty`},
		{`RelatedTo`, `Related to`},
		{`SyncsWith`, `Syncs with`},
		{`Creates`, `Creates`},
		{`Terminates`, `Terminates`},
		{`Obfuscates`, `Obfuscates`},
		{`Actor`, `Actor`},
		{`Act`, `Act`},
		{`Recipient`, `Recipient`},
		{`Extend`, `Extend`},
		{`Holder`, `Holder`},
		{`Claimant`, `Claimant`},

		{`For`, `For`},
		{`When`, `When`},

		// Iterators
		{`Count`, `Count`},
		{`Sum`, `Sum`},
		{`Max`, `Max`},
		{`Min`, `Min`},

		{`Not`, `Not`},

		{`True`, `True`},
		{`False`, `False`},
		{`OR`, `\|\|`},
		{`AND`, `&&`},
		{`NOT`, `NOT`},
		{`Range`, `\.\.`},

		{`Int`, `[0-9]+`},
		{`String`, `[A-Z][a-z0-9]*`},

		// Statements
		{`Iquery`, `\?-`},
		{`Bquery`, `\?`},
		{`Create`, `\+`},
		{`Obfuscate`, `~`},
		{`Terminate`, `-`},

		{`Comma`, `,`},
		{`Plus`, `\+`},
		{`Star`, `\*`},
		{`Div`, `/`},
		{`Mod`, `%`},
		{`Range`, `\.\.`},
		{`LParen`, `\(`},
		{`RParen`, `\)`},
		{"comment", `[#;][^\n]*`},
		{"Newline", `\n`},
	})
	parser = participle.MustBuild[Input](
		participle.Lexer(eflintLexer),
		participle.Union[Phrase](Fact{}, Query{}, Statement{}, Placeholder{}, Predicate{}, Event{}, Act{}, Duty{}, Extend{}),
		// TODO: Figure out how to deal with parentheses in expressions (e.g. "a && (b || c)").
		participle.Union[Expression](String{}, Int{}, ConstructorApplication{}, Reference{}, Arithmetic{}),
		participle.Union[Range](String{}, Int{}),
	)
	version = "0.1.0"
	kind    = "phrases"
	updates = true
)

type Input struct {
	Version string   `json:"version" parser:""`
	Kind    string   `json:"kind"    parser:""`
	Phrases []Phrase `json:"phrases" parser:"@@*"`
	Updates bool     `json:"updates" parser:""`
}

type Phrase interface {
	phrase()
}

type Range interface {
	isRange()
}

type Fact struct {
	Kind          string        `json:"kind"                     parser:""`
	Stateless     bool          `json:"stateless,omitempty"      parser:""`
	Updates       bool          `json:"updates,omitempty"        parser:""`
	Name          string        `json:"name,omitempty"           parser:"Fact @FactID"`
	Type          string        `json:"type,omitempty"           parser:"( (IdentifiedBy @(StringType | IntType))"`
	IdentifiedBy  []string      `json:"identified-by,omitempty"  parser:"| (IdentifiedBy @FactID ( Star @FactID )*)"`
	Range         []Range       `json:"range,omitempty"          parser:"| (IdentifiedBy (?= Int) @@ Range (?= Int) @@) | (IdentifiedBy @@ (Comma @@)*))?"`
	DerivedFrom   []Expression  `json:"derived-from,omitempty"   parser:"( (DerivedFrom @@ (Comma @@)*)"`
	HoldsWhen     []Expression  `json:"holds-when,omitempty"     parser:"| (HoldsWhen @@ (Comma @@)*)"`
	ConditionedBy []Expression  `json:"conditioned-by,omitempty" parser:"| (ConditionedBy @@ (Comma @@)*) )*"`
	Tokens        []lexer.Token `json:"-" parser:""`
}

func (f Fact) phrase() {}

type Query struct {
	Kind      string     `json:"kind"                     parser:"@(Iquery | Bquery)"`
	Stateless bool       `json:"stateless,omitempty"      parser:""`
	Updates   bool       `json:"updates,omitempty"        parser:""`
	Operand   Expression `json:"expression"               parser:"@@"`
}

func (q Query) phrase() {}

type Statement struct {
	Kind    string     `json:"kind"    parser:"@(Create | Obfuscate | Terminate)"`
	Operand Expression `json:"operand" parser:"@@"`
}

func (s Statement) phrase() {}

type Placeholder struct {
	Kind string   `json:"kind" parser:"Placeholder"`
	Name []string `json:"name" parser:"@FactID"`
	For  string   `json:"for"  parser:"For @FactID"`
}

func (p Placeholder) phrase() {}

type IsInvariant bool

func (b *IsInvariant) Capture(values []string) error {
	*b = values[0] == "Invariant"
	return nil
}

type Predicate struct {
	Kind        string      `json:"kind"                   parser:""`
	IsInvariant IsInvariant `json:"is-invariant,omitempty" parser:"@(Invariant | Predicate)"`
	Name        string      `json:"name"                   parser:"@FactID"`
	Expression  Expression  `json:"expression"             parser:"When @@"`
}

func (p Predicate) phrase() {}

type Event struct {
	Kind          string       `json:"kind"                     parser:"Event" default:"Event"`
	Name          string       `json:"name"                     parser:"@FactID"`
	RelatedTo     []string     `json:"related-to,omitempty"     parser:"(RelatedTo @FactID ( Comma @FactID )*)?"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"   parser:"( (DerivedFrom   @@ (Comma @@)*)"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"     parser:"| (HoldsWhen     @@ (Comma @@)*)"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty" parser:"| (ConditionedBy @@ (Comma @@)*)"`
	SyncsWith     []Expression `json:"syncs-with,omitempty"     parser:"| (SyncsWith     @@ (Comma @@)*)"`
	Creates       []Expression `json:"creates,omitempty"        parser:"| (Creates       @@ (Comma @@)*)"`
	Terminates    []Expression `json:"terminates,omitempty"     parser:"| (Terminates    @@ (Comma @@)*)"`
	Obfuscates    []Expression `json:"obfuscates,omitempty"     parser:"| (Obfuscates    @@ (Comma @@)*) )*"`
}

func (e Event) phrase() {}

type Act struct {
	Kind          string       `json:"kind"                     parser:"Act" default:"Act"`
	Name          string       `json:"name"                     parser:"@FactID"`
	Actor         string       `json:"actor,omitempty"          parser:"Actor @FactID"`
	RelatedTo     []string     `json:"related-to,omitempty"     parser:"(RelatedTo @FactID ( Comma @FactID )*)?"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"   parser:"( (DerivedFrom   @@ (Comma @@)*)"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"     parser:"| (HoldsWhen     @@ (Comma @@)*)"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty" parser:"| (ConditionedBy @@ (Comma @@)*)"`
	SyncsWith     []Expression `json:"syncs-with,omitempty"     parser:"| (SyncsWith     @@ (Comma @@)*)"`
	Creates       []Expression `json:"creates,omitempty"        parser:"| (Creates       @@ (Comma @@)*)"`
	Terminates    []Expression `json:"terminates,omitempty"     parser:"| (Terminates    @@ (Comma @@)*)"`
	Obfuscates    []Expression `json:"obfuscates,omitempty"     parser:"| (Obfuscates    @@ (Comma @@)*) )*"`
}

func (a Act) phrase() {}

type Duty struct {
	Kind          string       `json:"kind"                     parser:"Duty" default:"Duty"`
	Name          string       `json:"name"                     parser:"@FactID"`
	Holder        string       `json:"holder"                   parser:"Holder @FactID"`
	Claimant      string       `json:"claimant"                 parser:"Claimant @FactID"`
	RelatedTo     []string     `json:"related-to,omitempty"     parser:"(RelatedTo @FactID ( Comma @FactID )*)?"`
	DerivedFrom   []Expression `json:"derived-from,omitempty"   parser:"( (DerivedFrom   @@ (Comma @@)*)"`
	HoldsWhen     []Expression `json:"holds-when,omitempty"     parser:"| (HoldsWhen     @@ (Comma @@)*)"`
	ConditionedBy []Expression `json:"conditioned-by,omitempty" parser:"| (ConditionedBy @@ (Comma @@)*) )*"`
	ViolatedWhen  Expression   `json:"violated-when"            parser:"@@"`
}

func (d Duty) phrase() {}

type Extend struct {
	ParentKind string `json:"parent-kind" parser:"Extend @(Fact | Act | Event | Duty)"`
	Name       string `json:"name"        parser:"@FactID"`
	// TODO: Add support for extending
}

func (e Extend) phrase() {}

type Expression interface {
	expression()
}

type SubExpression struct {
	Expression Expression `json:"expression" parser:"LParen @@ RParen"`
}

func (s SubExpression) expression() {}
func (s SubExpression) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Expression)
}

type Arithmetic struct {
	Left     Expression `json:"left"     parser:"@@"`
	Operator string     `json:"operator" parser:"@(Div | Star | Create | Terminate)"`
	Right    Expression `json:"right"    parser:"@@"`
}

func (a Arithmetic) expression() {}
func (a Arithmetic) MarshalJSON() ([]byte, error) {
	return json.Marshal(Operator{
		Operator: a.Operator,
		Operands: []Expression{a.Left, a.Right},
	})
}

type Iterator struct { // TODO: Only a foreach can be inside an iterator
	Operator string       `json:"operator" parser:"@(Count | Sum | Max | Min)"`
	Operands []Expression `json:"operands" parser:"LParen @@ RParen"`
}

type Not struct {
	Expression Expression `json:"expression" parser:"Not @@"`
}

func (n Not) expression() {}
func (n Not) MarshalJSON() ([]byte, error) {
	return json.Marshal(Operator{
		Operator: "NOT",
		Operands: []Expression{n.Expression},
	})
}

type String struct {
	Value string `parser:"@String"`
}

func (s String) expression() {}
func (s String) isRange()    {}

func (s String) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

type Int struct {
	Value int `parser:"@Int"`
}

func (i Int) expression() {}
func (i Int) isRange()    {}

func (i Int) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Value)
}

type Reference struct {
	Value string `parser:"@FactID"`
}

func (r Reference) expression() {}

func (r Reference) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{r.Value})
}

type ConstructorApplication struct {
	Identifier string       `json:"identifier" parser:"@FactID"`
	Operands   []Expression `json:"operands"   parser:"( LParen (@@ (Comma @@)*)? RParen )+"`
}

func (c ConstructorApplication) expression() {}

type Operator struct {
	Operator string       `json:"operator" parser:""`
	Operands []Expression `json:"operands" parser:""`
}

func (o Operator) expression() {}

func parseRangeValues(r []Range, tokens []lexer.Token) ([]Range, error) {
	for t := range tokens {
		if tokens[t].Value == ".." {
			if len(r) != 2 {
				return nil, fmt.Errorf("invalid range")
			}
			if _, ok := r[0].(Int); !ok {
				return nil, fmt.Errorf("invalid range")
			}
			if _, ok := r[1].(Int); !ok {
				return nil, fmt.Errorf("invalid range")
			}
			lower, upper := r[0].(Int).Value, r[1].(Int).Value

			if lower > upper {
				return nil, fmt.Errorf("invalid range")
			}

			r = []Range{Int{lower}}
			for i := lower + 1; i <= upper; i++ {
				r = append(r, Int{i})
			}
		}
	}

	return r, nil
}

func parseRangeType(r []Range) (string, bool) {
	rangeType := ""

	for _, e := range r {
		switch e.(type) {
		case String:
			if rangeType == "" {
				rangeType = "String"
			} else if rangeType != "String" {
				return rangeType, false
			}
		case Int:
			if rangeType == "" {
				rangeType = "Int"
			} else if rangeType != "Int" {
				return rangeType, false
			}
		}
	}

	return rangeType, true
}

func main() {
	//fmt.Println(parser.String())
	filename := ""
	file := os.Stdin
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}
		defer f.Close()
		filename = os.Args[1]
		file = f
	}

	ini, err := parser.Parse(filename, file)
	if err != nil {
		panic(err)
	}
	//pp.Println(ini)

	// Add metadata
	ini.Version = version
	ini.Kind = kind
	ini.Updates = updates

	// Fill in missing fields
	for i, phrase := range (*ini).Phrases {
		switch phrase.(type) {
		case Fact:
			f := phrase.(Fact)
			if len(f.IdentifiedBy) > 0 {
				// Composite fact
				f.Kind = "cfact"
			} else {
				// Atomic fact
				f.Kind = "afact"

				if f.Type == "" {
					if len(f.Range) > 0 {
						rangeType, ok := parseRangeType(f.Range)
						if !ok {
							panic("range type mismatch")
						}
						f.Type = rangeType
						rangeValues, err := parseRangeValues(f.Range, f.Tokens)
						if err != nil {
							panic(err)
						}
						f.Range = rangeValues
					} else {
						f.Type = "String"
					}
				}
			}

			ini.Phrases[i] = f
		case Query:
			q := phrase.(Query)
			if q.Kind == "?" {
				q.Kind = "bquery"
			} else if q.Kind == "?-" {
				q.Kind = "iquery"
			} else {
				panic("unknown query type")
			}
			ini.Phrases[i] = q
		case Statement:
			s := phrase.(Statement)
			if s.Kind == "+" {
				s.Kind = "create"
			} else if s.Kind == "-" {
				s.Kind = "terminate"
			} else if s.Kind == "~" {
				s.Kind = "obfuscate"
			} else {
				panic("unknown statement type")
			}
			ini.Phrases[i] = s
		case Placeholder:
			p := phrase.(Placeholder)
			p.Kind = "placeholder"
			ini.Phrases[i] = p
		case Predicate:
			p := phrase.(Predicate)
			if p.IsInvariant {
				p.Kind = "invariant"
			} else {
				p.Kind = "predicate"
			}
			ini.Phrases[i] = p
		}
	}

	// Encode it and output it as JSON
	json, err := json.MarshalIndent(ini, "", "  ")
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(json)
}
