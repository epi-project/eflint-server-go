package eflint

import "reflect"

var SupportedVersions = []string{"1", "2", "3"}

const Reasoner = "eflint"
const ReasonerVersion = "3"
const SharesUpdates = true
const SharesTriggers = true
const SharesViolations = false

var stringType = reflect.TypeOf("")
var boolType = reflect.TypeOf(true)
var arrayType = reflect.TypeOf([]interface{}{})
var objectType = reflect.TypeOf(map[string]interface{}{})

var globalState = make(map[string]interface{})
var globalResults = make([]Result, 0)
var globalErrors = make([]Error, 0)
