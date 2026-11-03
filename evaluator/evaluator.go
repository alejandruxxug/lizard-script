// Package evaluator runs a lizard-script program by walking its syntax tree.
//
// Eval returns a value for every node. Three internal values travel up the
// tree to stop what is running: a ReturnValue up to the function call that it
// leaves, a Break up to the innermost loop, and an Error all the way up.
package evaluator

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/object"
	"lizardscript/token"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The values that exist only once, so they can be compared by pointer.
var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	BREAK = &object.Break{}
)

// maxCallDepth limits how many function calls can be open at once, so a
// recursion that never ends becomes an error instead of crashing Go.
const maxCallDepth = 10000

// maxStringLength limits the strings made with *, so "a" * 1000000000000
// is an error instead of using all the memory.
const maxStringLength = 1 << 28

// maxEvalDepth limits how many calls of Eval can be open at once. The
// parser already limits how deeply a program nests, so they can only pile up
// through function calls: 10000 calls whose bodies nest deeply would still
// overflow Go's stack. The deepest path, a chain of operators, takes about
// 700 bytes of stack per level, so the limit keeps the stack near 200 MB.
// Go stops a stack before it grows past 1 GB, and it grows by doubling, so
// that leaves room.
const maxEvalDepth = 300000

// callDepth counts the function calls open right now, and evalDepth the
// calls of Eval.
var callDepth, evalDepth int

// Eval runs node in env and returns its value. A program that fails returns
// an *object.Error, whose message has the line and column of the mistake.
func Eval(node ast.Node, env *object.Environment) object.Object {
	evalDepth++
	defer func() { evalDepth-- }()

	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, object.NewEnclosedEnvironment(env))

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.LetStatement:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		env.Define(node.Name.Value, value)
		return NULL

	case *ast.ReturnStatement:
		if node.ReturnValue == nil {
			return &object.ReturnValue{Value: NULL}
		}
		value := Eval(node.ReturnValue, env)
		if isError(value) {
			return value
		}
		return &object.ReturnValue{Value: value}

	case *ast.BreakStatement:
		return BREAK

	// Leaf expressions
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	// Operators
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node, right)

	case *ast.InfixExpression:
		return evalInfixExpression(node, env)

	case *ast.AssignExpression:
		return evalAssignExpression(node, env)

	// Control flow
	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.WhileExpression:
		return evalWhileExpression(node, env)

	case *ast.ForExpression:
		return evalForExpression(node, env)

	// Functions
	case *ast.FunctionLiteral:
		return &object.Function{Parameters: node.Parameters, Body: node.Body, Env: env}

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(node.Token, function, args)

	// Data structures
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(node.Token, left, index)

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	}

	return nil
}

// Errors

// newError builds a runtime error at the position of tok.
func newError(tok token.Token, format string, args ...any) *object.Error {
	return &object.Error{Message: fmt.Sprintf("[line %d, column %d] runtime error: %s",
		tok.Line, tok.Column, fmt.Sprintf(format, args...))}
}

func isError(obj object.Object) bool {
	return obj != nil && obj.Type() == object.ERROR_OBJ
}

// Statements

// evalProgram runs the statements in order. A return outside of a function
// ends the program, and its value is the value of the program.
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object = NULL

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

// evalBlockStatement runs the statements of a block in env, which the caller
// creates. A return, break or error stops the block and goes up as it is, so
// the function or loop around it can handle it.
func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object = NULL

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			switch result.Type() {
			case object.RETURN_VALUE_OBJ, object.BREAK_OBJ, object.ERROR_OBJ:
				return result
			}
		}
	}

	return result
}

// Expressions

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if value, ok := env.Get(node.Value); ok {
		return value
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	return newError(node.Token, "identifier not found: %s", node.Value)
}

// evalExpressions evaluates a list of expressions from left to right. If one
// fails, it returns only that error.
func evalExpressions(expressions []ast.Expression, env *object.Environment) []object.Object {
	result := []object.Object{}

	for _, e := range expressions {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

// isTruthy decides whether a value counts as true in a condition. false,
// null, 0, "" and empty arrays and hashes are false. Everything else is true.
func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	case *object.Integer:
		return obj.Value != 0
	case *object.String:
		return obj.Value != ""
	case *object.Array:
		return len(obj.Elements) != 0
	case *object.Hash:
		return len(obj.Pairs) != 0
	default:
		return true
	}
}

func nativeBoolToBooleanObject(value bool) *object.Boolean {
	if value {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(node *ast.PrefixExpression, right object.Object) object.Object {
	switch node.Operator {
	case "!":
		return nativeBoolToBooleanObject(!isTruthy(right))
	case "-":
		integer, ok := right.(*object.Integer)
		if !ok {
			return newError(node.Token, "unknown operator: -%s", right.Type())
		}
		return &object.Integer{Value: -integer.Value}
	default:
		return newError(node.Token, "unknown operator: %s%s", node.Operator, right.Type())
	}
}

func evalInfixExpression(node *ast.InfixExpression, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	// && and || only evaluate their right side when the left one doesn't
	// already decide the result.
	switch node.Operator {
	case "&&":
		if !isTruthy(left) {
			return FALSE
		}
		return evalLogicalRight(node, env)
	case "||":
		if isTruthy(left) {
			return TRUE
		}
		return evalLogicalRight(node, env)
	}

	right := Eval(node.Right, env)
	if isError(right) {
		return right
	}

	switch node.Operator {
	case "==":
		return nativeBoolToBooleanObject(looseEquals(left, right))
	case "!=":
		return nativeBoolToBooleanObject(!looseEquals(left, right))
	case "===":
		return nativeBoolToBooleanObject(strictEquals(left, right))
	case "!==":
		return nativeBoolToBooleanObject(!strictEquals(left, right))
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(node.Token, node.Operator, left.(*object.Integer).Value, right.(*object.Integer).Value)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(node.Token, node.Operator, left.(*object.String).Value, right.(*object.String).Value)
	case node.Operator == "*" && left.Type() == object.STRING_OBJ && right.Type() == object.INTEGER_OBJ:
		return repeatString(node.Token, left.(*object.String).Value, right.(*object.Integer).Value)
	case node.Operator == "*" && left.Type() == object.INTEGER_OBJ && right.Type() == object.STRING_OBJ:
		return repeatString(node.Token, right.(*object.String).Value, left.(*object.Integer).Value)
	case node.Operator == "+" && left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ:
		leftElements, rightElements := left.(*object.Array).Elements, right.(*object.Array).Elements
		elements := make([]object.Object, 0, len(leftElements)+len(rightElements))
		elements = append(elements, leftElements...)
		return &object.Array{Elements: append(elements, rightElements...)}
	case left.Type() != right.Type():
		return newError(node.Token, "type mismatch: %s %s %s", left.Type(), node.Operator, right.Type())
	default:
		return newError(node.Token, "unknown operator: %s %s %s", left.Type(), node.Operator, right.Type())
	}
}

// evalLogicalRight evaluates the right side of && or ||, which decides the
// result once the left side didn't.
func evalLogicalRight(node *ast.InfixExpression, env *object.Environment) object.Object {
	right := Eval(node.Right, env)
	if isError(right) {
		return right
	}
	return nativeBoolToBooleanObject(isTruthy(right))
}

func evalIntegerInfixExpression(tok token.Token, operator string, left, right int64) object.Object {
	switch operator {
	case "+":
		return &object.Integer{Value: left + right}
	case "-":
		return &object.Integer{Value: left - right}
	case "*":
		return &object.Integer{Value: left * right}
	case "/", "//", "%":
		if right == 0 {
			return newError(tok, "division by zero")
		}
		switch operator {
		case "/":
			return &object.Integer{Value: left / right}
		case "//":
			return &object.Integer{Value: floorDiv(left, right)}
		default:
			return &object.Integer{Value: left - floorDiv(left, right)*right}
		}
	case "**":
		if right < 0 {
			return newError(tok, "negative exponent: %d ** %d", left, right)
		}
		return &object.Integer{Value: power(left, right)}
	case "<":
		return nativeBoolToBooleanObject(left < right)
	case ">":
		return nativeBoolToBooleanObject(left > right)
	case "<=":
		return nativeBoolToBooleanObject(left <= right)
	case ">=":
		return nativeBoolToBooleanObject(left >= right)
	default:
		return newError(tok, "unknown operator: INTEGER %s INTEGER", operator)
	}
}

// floorDiv divides rounding down, so -7 // 2 is -4. Go's / rounds towards
// zero and gives -3.
func floorDiv(left, right int64) int64 {
	quotient := left / right
	if left%right != 0 && (left < 0) != (right < 0) {
		quotient--
	}
	return quotient
}

// power computes base ** exponent by squaring. Like the other operators, it
// wraps around when the result doesn't fit in 64 bits.
func power(base, exponent int64) int64 {
	result := int64(1)
	for exponent > 0 {
		if exponent&1 == 1 {
			result *= base
		}
		base *= base
		exponent >>= 1
	}
	return result
}

func evalStringInfixExpression(tok token.Token, operator string, left, right string) object.Object {
	switch operator {
	case "+":
		if len(left)+len(right) > maxStringLength {
			return newError(tok, "string too long")
		}
		return &object.String{Value: left + right}
	case "<":
		return nativeBoolToBooleanObject(left < right)
	case ">":
		return nativeBoolToBooleanObject(left > right)
	case "<=":
		return nativeBoolToBooleanObject(left <= right)
	case ">=":
		return nativeBoolToBooleanObject(left >= right)
	default:
		return newError(tok, "unknown operator: STRING %s STRING", operator)
	}
}

// repeatString handles "ab" * 3, which is "ababab".
func repeatString(tok token.Token, s string, count int64) object.Object {
	if count < 0 {
		return newError(tok, "can't repeat a string a negative number of times: %d", count)
	}
	if len(s) > 0 && count > int64(maxStringLength/len(s)) {
		return newError(tok, "string too long")
	}
	return &object.String{Value: strings.Repeat(s, int(count))}
}

// Equality

// strictEquals is ===: both values have the same type and the same value.
// Arrays and hashes are equal when their contents are. Functions are only
// equal to themselves.
func strictEquals(left, right object.Object) bool {
	return equals(left, right, true)
}

// looseEquals is ==. It works like === but first converts between some
// types: true and false are 1 and 0, and a string that is a whole number
// equals that number. So 1 == true and "5" == 5, but 1 !== true.
func looseEquals(left, right object.Object) bool {
	return equals(left, right, false)
}

// toNumber is the conversion of looseEquals. It fails for anything that
// isn't an integer, a boolean or a string with a whole number.
func toNumber(obj object.Object) (int64, bool) {
	switch obj := obj.(type) {
	case *object.Integer:
		return obj.Value, true
	case *object.Boolean:
		if obj.Value {
			return 1, true
		}
		return 0, true
	case *object.String:
		number, err := strconv.ParseInt(obj.Value, 10, 64)
		return number, err == nil
	default:
		return 0, false
	}
}

// pair is two values being compared.
type pair struct {
	left, right object.Object
}

// equals compares two values with === when strict is set, or with ==.
// Arrays and hashes are compared element by element, with the same kind of
// equality.
//
// The pairs left to compare are kept in a stack instead of the function
// calling itself, so arrays nested a million times don't overflow Go's
// stack. seen has the pairs of arrays and hashes already taken apart.
// Meeting one again, because a value contains itself or the same value
// appears twice, needs no second look: its elements are already on the way.
// That is also why comparing an array that contains itself ends.
func equals(left, right object.Object, strict bool) bool {
	stack := []pair{{left, right}}
	var seen map[pair]bool

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		left, right := current.left, current.right

		if left.Type() != right.Type() {
			if strict {
				return false
			}
			leftNumber, leftOk := toNumber(left)
			rightNumber, rightOk := toNumber(right)
			if !leftOk || !rightOk || leftNumber != rightNumber {
				return false
			}
			continue
		}

		switch left := left.(type) {
		case *object.Integer:
			if left.Value != right.(*object.Integer).Value {
				return false
			}
		case *object.Boolean:
			if left.Value != right.(*object.Boolean).Value {
				return false
			}
		case *object.String:
			if left.Value != right.(*object.String).Value {
				return false
			}
		case *object.Null:
		case *object.Array:
			right := right.(*object.Array)
			if len(left.Elements) != len(right.Elements) {
				return false
			}
			if seen[current] {
				continue
			}
			if seen == nil {
				seen = map[pair]bool{}
			}
			seen[current] = true
			for i := range left.Elements {
				stack = append(stack, pair{left.Elements[i], right.Elements[i]})
			}
		case *object.Hash:
			right := right.(*object.Hash)
			if len(left.Pairs) != len(right.Pairs) {
				return false
			}
			if seen[current] {
				continue
			}
			if seen == nil {
				seen = map[pair]bool{}
			}
			seen[current] = true
			for key, entry := range left.Pairs {
				other, ok := right.Pairs[key]
				if !ok {
					return false
				}
				stack = append(stack, pair{entry.Value, other.Value})
			}
		default:
			if left != right {
				return false
			}
		}
	}

	return true
}

// Assignment

func evalAssignExpression(node *ast.AssignExpression, env *object.Environment) object.Object {
	switch left := node.Left.(type) {
	case *ast.Identifier:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		if !env.Assign(left.Value, value) {
			return newError(left.Token, "can't assign to %s: it was never declared, use let %s = ...",
				left.Value, left.Value)
		}
		return value

	case *ast.IndexExpression:
		container := Eval(left.Left, env)
		if isError(container) {
			return container
		}
		index := Eval(left.Index, env)
		if isError(index) {
			return index
		}
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		return assignIndex(left.Token, container, index, value)

	default:
		return newError(node.Token, "can't assign to %s", node.Left.String())
	}
}

// assignIndex handles list[i] = value and hash[key] = value. An array index
// must already exist: push adds new elements.
func assignIndex(tok token.Token, container, index, value object.Object) object.Object {
	switch container := container.(type) {
	case *object.Array:
		integer, ok := index.(*object.Integer)
		if !ok {
			return newError(tok, "array index must be INTEGER, got %s", index.Type())
		}
		i := integer.Value
		if i < 0 || i >= int64(len(container.Elements)) {
			return newError(tok, "index out of range: %d, the array has %d elements", i, len(container.Elements))
		}
		container.Elements[i] = value
		return value

	case *object.Hash:
		key, ok := index.(object.Hashable)
		if !ok {
			return newError(tok, "unusable as hash key: %s", index.Type())
		}
		container.Set(key, value)
		return value

	case *object.String:
		return newError(tok, "strings can't be changed")

	default:
		return newError(tok, "index assignment not supported: %s", container.Type())
	}
}

// Control flow

func evalIfExpression(node *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(node.Consequence, env)
	} else if node.Alternative != nil {
		return Eval(node.Alternative, env)
	}
	return NULL
}

// evalWhileExpression runs the body while the condition holds. A loop has
// no value of its own, so it produces null.
func evalWhileExpression(node *ast.WhileExpression, env *object.Environment) object.Object {
	for {
		condition := Eval(node.Condition, env)
		if isError(condition) {
			return condition
		}
		if !isTruthy(condition) {
			return NULL
		}

		if result, stop := evalLoopBody(node.Body, env); stop {
			return result
		}
	}
}

// evalForExpression runs a C-style loop. The variables of the init part
// live in a scope of their own, so they are gone after the loop.
func evalForExpression(node *ast.ForExpression, env *object.Environment) object.Object {
	loopEnv := object.NewEnclosedEnvironment(env)

	if node.Init != nil {
		if init := Eval(node.Init, loopEnv); isError(init) {
			return init
		}
	}

	for {
		if node.Condition != nil {
			condition := Eval(node.Condition, loopEnv)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				return NULL
			}
		}

		if result, stop := evalLoopBody(node.Body, loopEnv); stop {
			return result
		}

		if node.Update != nil {
			if update := Eval(node.Update, loopEnv); isError(update) {
				return update
			}
		}
	}
}

// evalLoopBody runs one round of a loop in a fresh scope. It reports whether
// the loop must stop, and with which value: null after a break, or the
// return or error that goes on up.
func evalLoopBody(body *ast.BlockStatement, env *object.Environment) (object.Object, bool) {
	result := Eval(body, env)

	switch result.Type() {
	case object.BREAK_OBJ:
		return NULL, true
	case object.RETURN_VALUE_OBJ, object.ERROR_OBJ:
		return result, true
	}
	return nil, false
}

// Functions

// applyFunction calls function with args. tok is the '(' of the call, where
// errors from builtins are reported.
func applyFunction(tok token.Token, function object.Object, args []object.Object) object.Object {
	switch function := function.(type) {
	case *object.Function:
		if len(args) != len(function.Parameters) {
			return newError(tok, "wrong number of arguments: want %d, got %d",
				len(function.Parameters), len(args))
		}
		if callDepth >= maxCallDepth {
			return newError(tok, "stack overflow: more than %d nested calls", maxCallDepth)
		}
		if evalDepth >= maxEvalDepth {
			return newError(tok, "stack overflow: the calls and the code inside them nest too deeply")
		}

		env := object.NewEnclosedEnvironment(function.Env)
		for i, parameter := range function.Parameters {
			env.Define(parameter.Value, args[i])
		}

		callDepth++
		result := evalBlockStatement(function.Body, env)
		callDepth--

		// A function without return gives the value of its last statement.
		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
		return result

	case *object.Builtin:
		result := function.Fn(args...)
		if err, ok := result.(*object.Error); ok {
			return newError(tok, "%s: %s", function.Name, err.Message)
		}
		return result

	default:
		return newError(tok, "not a function: %s", function.Type())
	}
}

// Data structures

// evalIndexExpression reads list[i], hash[key] or text[i]. An index that
// doesn't exist gives null.
func evalIndexExpression(tok token.Token, left, index object.Object) object.Object {
	switch left := left.(type) {
	case *object.Array:
		integer, ok := index.(*object.Integer)
		if !ok {
			return newError(tok, "array index must be INTEGER, got %s", index.Type())
		}
		i := integer.Value
		if i < 0 || i >= int64(len(left.Elements)) {
			return NULL
		}
		return left.Elements[i]

	case *object.Hash:
		key, ok := index.(object.Hashable)
		if !ok {
			return newError(tok, "unusable as hash key: %s", index.Type())
		}
		pair, ok := left.Pairs[key.HashKey()]
		if !ok {
			return NULL
		}
		return pair.Value

	case *object.String:
		integer, ok := index.(*object.Integer)
		if !ok {
			return newError(tok, "string index must be INTEGER, got %s", index.Type())
		}
		i := integer.Value
		if i < 0 || i >= int64(utf8.RuneCountInString(left.Value)) {
			return NULL
		}
		return &object.String{Value: string([]rune(left.Value)[i])}

	default:
		return newError(tok, "index operator not supported: %s", left.Type())
	}
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	hash := object.NewHash()

	for _, pair := range node.Pairs {
		key := Eval(pair.Key, env)
		if isError(key) {
			return key
		}
		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError(node.Token, "unusable as hash key: %s", key.Type())
		}

		value := Eval(pair.Value, env)
		if isError(value) {
			return value
		}
		hash.Set(hashKey, value)
	}

	return hash
}
