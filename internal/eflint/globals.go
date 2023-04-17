package eflint

import "reflect"

var SupportedVersions = []string{"0.1.0"}

const Reasoner = "eflint"
const ReasonerVersion = "3"
const SharesUpdates = true
const SharesTriggers = true
const SharesViolations = false

var stringType = reflect.TypeOf("")
var boolType = reflect.TypeOf(true)
var arrayType = reflect.TypeOf([]interface{}{})
var objectType = reflect.TypeOf(map[string]interface{}{})

// TODO: Phrases can be stateless, so 1 global state is not enough.
//       Can split into a global state and a local state.
// TODO: Look into possibility of storing all the stateful phrases,
//       and running those at the start of every request.
var globalState = make(map[string]map[string]interface{})

var localState = make(map[string]interface{})
var globalResults = make([]interface{}, 0)
var globalErrors = make([]Error, 0)
