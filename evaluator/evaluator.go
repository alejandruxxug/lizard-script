// Package evaluator runs a lizard-script program by walking its syntax tree.
//
// Eval returns a value for every node. Four internal values travel up the
// tree to stop what is running: a ReturnValue up to the function call that it
// leaves, a Break or a Continue up to the innermost loop, and an Error up to
// the nearest try statement, or all the way up if there is none.
package evaluator

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/object"
	"lizardscript/token"
	"strconv"
	"strings"
)

// The values that exist only once, so they can be compared by pointer.
var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	BREAK = &object.Break{}

	CONTINUE = &object.Continue{}
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

// MaxSteps limits how many steps a program can take, so a host that runs
// code it doesn't control (an editor that runs it on every keystroke) can
// stop a loop that never ends. A step is one round of a loop or one function
// call, builtins included. 0 means no limit, which is the default.
var MaxSteps int

// steps counts the steps taken since ResetSteps.
var steps int

// ResetSteps starts counting steps from zero. A host that sets MaxSteps calls
// it before each run.
func ResetSteps() {
	steps = 0
}

// Steps returns how many steps have been counted since the last ResetSteps.
// Steps are only counted while MaxSteps is above 0.
func Steps() int {
	return steps
}

// SetSteps puts the step count back to n. A host that runs a program inside
// another one saves Steps(), runs with ResetSteps(), and restores the count
// with SetSteps when it is done.
func SetSteps(n int) {
	steps = n
}

// countStep takes one step. It returns an error at tok once the program has
// gone over MaxSteps, and nil otherwise.
func countStep(tok token.Token) *object.Error {
	if MaxSteps == 0 {
		return nil
	}
	steps++
	if steps > MaxSteps {
		err := newError(tok, "the program ran too long (more than %d steps)", MaxSteps)
		err.Uncatchable = true
		return err
	}
	return nil
}

// OnStatement is a hook for a host. When it is not nil, it is called right
// before every statement runs: in the program, in blocks, in layer bodies and
// in function bodies. The host can read the statement's token for its place
// in the source. The REPL and Run leave it nil.
var OnStatement func(node ast.Statement)

// OnLayer is a hook for a host. When it is not nil, it is called with enter
// true when the body of a layer starts and with enter false when it ends,
// also when it ends by an error, a return or a break. tok is the "layer"
// token. The REPL and Run leave it nil.
var OnLayer func(name string, enter bool, tok token.Token)

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

	case *ast.LayerStatement:
		return evalLayerStatement(node, env)

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

	case *ast.ContinueStatement:
		return CONTINUE

	case *ast.TryStatement:
		return evalTryStatement(node, env)

	// Leaf expressions
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.ColorLiteral:
		return &object.Color{R: node.R, G: node.G, B: node.B, A: node.A}

	case *ast.PointLiteral:
		return evalPointLiteral(node, env)

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
		return &object.Function{Parameters: node.Parameters, Defaults: node.Defaults, Body: node.Body, Env: env}

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
	return &object.Error{
		Message: fmt.Sprintf("[line %d, column %d] runtime error: %s",
			tok.Line, tok.Column, fmt.Sprintf(format, args...)),
		Line:   tok.Line,
		Column: tok.Column,
		Source: tok.Source,
	}
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
		if OnStatement != nil {
			OnStatement(statement)
		}
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
		if OnStatement != nil {
			OnStatement(statement)
		}
		result = Eval(statement, env)

		if result != nil {
			switch result.Type() {
			case object.RETURN_VALUE_OBJ, object.BREAK_OBJ, object.CONTINUE_OBJ, object.ERROR_OBJ:
				return result
			}
		}
	}

	return result
}

// evalLayerStatement runs the body of a layer in a new scope and tells the
// host when it starts and ends. A return, break or error passes through, and
// the layer still ends for the host.
func evalLayerStatement(layer *ast.LayerStatement, env *object.Environment) object.Object {
	if OnLayer != nil {
		OnLayer(layer.Name, true, layer.Token)
		defer func() {
			if OnLayer != nil {
				OnLayer(layer.Name, false, layer.Token)
			}
		}()
	}

	result := evalBlockStatement(layer.Body, object.NewEnclosedEnvironment(env))
	switch result.Type() {
	case object.RETURN_VALUE_OBJ, object.BREAK_OBJ, object.CONTINUE_OBJ, object.ERROR_OBJ:
		return result
	}
	return NULL
}

// evalTryStatement runs the body. If a runtime error stops it, the catch
// block runs instead, in a scope where the name holds a hash with the
// "message" of the error, without its position, and the "line" and "column"
// where it happened. A return, break or continue in either block goes on up
// as it is. The errors that protect the host (the step limit and the stack
// overflows) are not caught. The value is the one of the block that ran.
func evalTryStatement(node *ast.TryStatement, env *object.Environment) object.Object {
	result := Eval(node.Body, env)

	err, ok := result.(*object.Error)
	if !ok || err.Uncatchable {
		return result
	}

	catchEnv := object.NewEnclosedEnvironment(env)
	if node.Name != nil {
		catchEnv.Define(node.Name.Value, errorToHash(err))
	}
	return Eval(node.Catch, catchEnv)
}

// errorToHash is the value a catch block gets for an error.
func errorToHash(err *object.Error) *object.Hash {
	prefix := fmt.Sprintf("[line %d, column %d] runtime error: ", err.Line, err.Column)

	hash := object.NewHash()
	hash.Set(&object.String{Value: "message"}, &object.String{Value: strings.TrimPrefix(err.Message, prefix)})
	hash.Set(&object.String{Value: "line"}, &object.Integer{Value: int64(err.Line)})
	hash.Set(&object.String{Value: "column"}, &object.Integer{Value: int64(err.Column)})
	return hash
}

// Expressions

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if value, ok := env.Get(node.Value); ok {
		return value
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	if node.Value == "null" {
		return NULL
	}
	if node.Value == "math" {
		return mathLibrary()
	}
	if node.Value == "text" {
		return textLibrary()
	}
	if node.Value == "list" {
		return listLibrary()
	}
	if node.Value == "hash" {
		return hashLibrary()
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
	case *object.Float:
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
		if point, ok := right.(*object.Point); ok {
			return negatePoint(point)
		}
		if float, ok := right.(*object.Float); ok {
			return &object.Float{Value: -float.Value}
		}
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
	return evalInfixValues(node.Token, node.Operator, left, right)
}

// evalInfixValues applies a binary operator, other than && and ||, to two
// values already evaluated. tok is where an error is reported.
func evalInfixValues(tok token.Token, operator string, left, right object.Object) object.Object {
	switch operator {
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
		return evalIntegerInfixExpression(tok, operator, left.(*object.Integer).Value, right.(*object.Integer).Value)
	case isNumber(left) && isNumber(right):
		return evalFloatInfixExpression(tok, operator, toFloat(left), toFloat(right))
	case left.Type() == object.POINT_OBJ || right.Type() == object.POINT_OBJ:
		if result, ok := evalPointInfix(tok, operator, left, right); ok {
			return result
		}
		return pointInfixError(tok, operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(tok, operator, left.(*object.String).Value, right.(*object.String).Value)
	case operator == "*" && left.Type() == object.STRING_OBJ && right.Type() == object.INTEGER_OBJ:
		return repeatString(tok, left.(*object.String).Value, right.(*object.Integer).Value)
	case operator == "*" && left.Type() == object.INTEGER_OBJ && right.Type() == object.STRING_OBJ:
		return repeatString(tok, right.(*object.String).Value, left.(*object.Integer).Value)
	case operator == "+" && left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ:
		leftElements, rightElements := left.(*object.Array).Elements, right.(*object.Array).Elements
		elements := make([]object.Object, 0, len(leftElements)+len(rightElements))
		elements = append(elements, leftElements...)
		return &object.Array{Elements: append(elements, rightElements...)}
	case left.Type() != right.Type():
		return newError(tok, "type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
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

// toNumber is the conversion of looseEquals when no float is involved. It fails for anything that
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

// uncheckedContainers is how many arrays and hashes equals takes apart
// before it starts remembering them.
const uncheckedContainers = 64

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
//
// Most comparisons are small (two points, a hash of a few fields), so the
// stack starts in a fixed buffer and seen is only made once more than
// uncheckedContainers arrays and hashes were taken apart: until then
// meeting one again only repeats comparisons, with the same answer, and a
// value that contains itself soon goes past the limit.
func equals(left, right object.Object, strict bool) bool {
	var buffer [16]pair
	stack := append(buffer[:0], pair{left, right})
	var seen map[pair]bool
	containers := 0

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		left, right := current.left, current.right
		if left == right {
			// The same value: equal to itself under both equalities.
			continue
		}

		if left.Type() != right.Type() {
			if strict {
				return false
			}
			if left.Type() == object.FLOAT_OBJ || right.Type() == object.FLOAT_OBJ {
				leftFloat, leftOk := looseFloat(left)
				rightFloat, rightOk := looseFloat(right)
				if !leftOk || !rightOk || leftFloat != rightFloat {
					return false
				}
				continue
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
		case *object.Float:
			if left.Value != right.(*object.Float).Value {
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
		case *object.Color:
			if *left != *right.(*object.Color) {
				return false
			}
		case *object.Point:
			right := right.(*object.Point)
			if len(left.Coords) != len(right.Coords) {
				return false
			}
			for i := range left.Coords {
				stack = append(stack, pair{left.Coords[i], right.Coords[i]})
			}
		case *object.Null:
		case *object.Array:
			right := right.(*object.Array)
			if len(left.Elements) != len(right.Elements) {
				return false
			}
			if containers++; containers > uncheckedContainers {
				if seen[current] {
					continue
				}
				if seen == nil {
					seen = map[pair]bool{}
				}
				seen[current] = true
			}
			for i := range left.Elements {
				stack = append(stack, pair{left.Elements[i], right.Elements[i]})
			}
		case *object.Hash:
			right := right.(*object.Hash)
			if len(left.Pairs) != len(right.Pairs) {
				return false
			}
			if containers++; containers > uncheckedContainers {
				if seen[current] {
					continue
				}
				if seen == nil {
					seen = map[pair]bool{}
				}
				seen[current] = true
			}
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
		// x += e is x = x + e: x is read first, and must exist.
		var current object.Object
		if node.Operator != "" {
			if current = evalIdentifier(left, env); isError(current) {
				return current
			}
		}
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		if node.Operator != "" {
			if value = evalInfixValues(node.Token, node.Operator, current, value); isError(value) {
				return value
			}
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
		// a[i] += e: a and i are evaluated once, and a[i] is read first.
		var current object.Object
		if node.Operator != "" {
			if current = evalIndexExpression(left.Token, container, index); isError(current) {
				return current
			}
		}
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		if node.Operator != "" {
			if value = evalInfixValues(node.Token, node.Operator, current, value); isError(value) {
				return value
			}
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

	case *object.Color:
		return newError(tok, "colors can't be changed")

	case *object.Point:
		return newError(tok, "points can't be changed")

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
		if err := countStep(node.Token); err != nil {
			return err
		}

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
		if err := countStep(node.Token); err != nil {
			return err
		}

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
// return or error that goes on up. A continue just ends the round.
func evalLoopBody(body *ast.BlockStatement, env *object.Environment) (object.Object, bool) {
	result := Eval(body, env)

	switch result.Type() {
	case object.BREAK_OBJ:
		return NULL, true
	case object.CONTINUE_OBJ:
		return nil, false
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
		if err := countStep(tok); err != nil {
			return err
		}
		if required := requiredParameters(function); len(args) < required || len(args) > len(function.Parameters) {
			if required == len(function.Parameters) {
				return newError(tok, "wrong number of arguments: want %d, got %d", required, len(args))
			}
			return newError(tok, "wrong number of arguments: want %d to %d, got %d",
				required, len(function.Parameters), len(args))
		}
		if callDepth >= maxCallDepth {
			return uncatchable(newError(tok, "stack overflow: more than %d nested calls", maxCallDepth))
		}
		if evalDepth >= maxEvalDepth {
			return uncatchable(newError(tok, "stack overflow: the calls and the code inside them nest too deeply"))
		}

		env := object.NewEnclosedEnvironment(function.Env)
		for i, arg := range args {
			env.Define(function.Parameters[i].Value, arg)
		}

		callDepth++
		defer func() { callDepth-- }()
		if err := bindDefaults(function, env, len(args)); err != nil {
			return err
		}
		result := evalBlockStatement(function.Body, env)

		// A function without return gives the value of its last statement.
		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
		return result

	case *object.Builtin:
		if err := countStep(tok); err != nil {
			return err
		}
		result := function.Fn(args...)
		if err, ok := result.(*object.Error); ok {
			// error(message) is the program's own error: its message
			// is shown as it is, without the builtin name in front.
			if function == builtins["error"] {
				return newError(tok, "%s", err.Message)
			}
			// An error from a function the builtin called (list.sort
			// with a comparison) keeps being one a try lets through.
			wrapped := newError(tok, "%s: %s", function.Name, err.Message)
			wrapped.Uncatchable = err.Uncatchable
			return wrapped
		}
		return result

	case object.Callable:
		// A function of another engine does the same checks itself.
		return function.Call(tok, args)

	default:
		return newError(tok, "not a function: %s", function.Type())
	}
}

// requiredParameters is how many arguments a call must give: the parameters
// before the first one with a default value.
func requiredParameters(function *object.Function) int {
	for i := range function.Parameters {
		if i < len(function.Defaults) && function.Defaults[i] != nil {
			return i
		}
	}
	return len(function.Parameters)
}

// bindDefaults gives the parameters after the first given arguments their
// default values. Each one is evaluated in the call's own scope, so it can
// use the parameters before it. It returns the error of a default that fails.
func bindDefaults(function *object.Function, env *object.Environment, given int) object.Object {
	for i := given; i < len(function.Parameters); i++ {
		value := Eval(function.Defaults[i], env)
		if returned, ok := value.(*object.ReturnValue); ok {
			value = returned.Value
		}
		if isError(value) {
			return value
		}
		env.Define(function.Parameters[i].Value, value)
	}
	return nil
}

// uncatchable marks err as one that a try statement lets through.
func uncatchable(err *object.Error) *object.Error {
	err.Uncatchable = true
	return err
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
		if i < 0 || i >= int64(left.Len()) {
			return NULL
		}
		return &object.String{Value: string(left.RuneAt(int(i)))}

	case *object.Color:
		return evalColorProperty(tok, left, index)

	case *object.Point:
		return evalPointProperty(tok, left, index)

	default:
		return newError(tok, "index operator not supported: %s", left.Type())
	}
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	hash := object.NewHashSized(len(node.Pairs))

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
