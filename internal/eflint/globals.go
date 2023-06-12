package eflint

import (
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"reflect"
)

var SupportedVersions = []string{"0.1.0"}

const Reasoner = "eflint"
const ReasonerVersion = "3"
const SharesUpdates = true
const SharesTriggers = true
const SharesViolations = false

var intType = reflect.TypeOf(int64(0))
var stringType = reflect.TypeOf("")
var boolType = reflect.TypeOf(true)
var arrayType = reflect.TypeOf([]interface{}{})
var objectType = reflect.TypeOf(map[string]interface{}{})

var FactType = 0
var EventType = 1
var ActType = 2
var DutyType = 3

var defaultFacts = map[string]string{
	"actor":  "String",
	"int":    "Int",
	"ref":    "String",
	"string": "String",
}

// TODO: Phrases can be stateless, so 1 global state is not enough.
//       Can split into a global state and a local state.
// TODO: Look into possibility of storing all the stateful phrases,
//       and running those at the start of every request.
var globalState = make(map[string]map[string]interface{})
var globalInstances = make(map[string]*orderedmap.OrderedMap[uint64, Expression])
var globalNonInstances = make(map[string]*orderedmap.OrderedMap[uint64, Expression])
var globalViolations = make(map[string][]Expression)

var globalResults = make([]PhraseResult, 0)
var globalErrors = make([]Error, 0)
