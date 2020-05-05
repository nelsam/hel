// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/poy/onpar/diff"
	"github.com/poy/onpar/matchers"
)

// Matcher is any type that can match values.  Some code in this package supports
// matching against child matchers, for example:
//    HaveBeenExecuted("Foo", WithArgs(matchers.HaveLen(12)))
type Matcher interface {
	Match(actual interface{}) (interface{}, error)
}

type any int

// Any is a special value to tell pers to allow any value at the position used.
// For example, you can assert only on the second argument with:
//     HaveMethodExecuted("Foo", WithArgs(Any, 22))
const Any any = -1

// HaveMethodExecutedOption is an option function for the HaveMethodExecutedMatcher.
type HaveMethodExecutedOption func(HaveMethodExecutedMatcher) HaveMethodExecutedMatcher

// Within returns a HaveMethodExecutedOption which sets the HaveMethodExecutedMatcher
// to be executed within a given timeframe.
func Within(d time.Duration) HaveMethodExecutedOption {
	return func(m HaveMethodExecutedMatcher) HaveMethodExecutedMatcher {
		m.within = d
		return m
	}
}

// WithArgs returns a HaveMethodExecutedOption which sets the HaveMethodExecutedMatcher
// to only pass if the latest execution of the method called it with the passed in
// arguments.
func WithArgs(args ...interface{}) HaveMethodExecutedOption {
	return func(m HaveMethodExecutedMatcher) HaveMethodExecutedMatcher {
		m.args = args
		return m
	}
}

// StoreArgs returns a HaveMethodExecutedOption which stores the arguments passed to
// the method in the addresses provided.
//
// StoreArgs will panic if the values provided are not pointers or cannot store data
// of the same type as the method arguments.
func StoreArgs(targets ...interface{}) HaveMethodExecutedOption {
	return func(m HaveMethodExecutedMatcher) HaveMethodExecutedMatcher {
		m.saveTo = targets
		return m
	}
}

// HaveMethodExecutedMatcher is a matcher to ensure that a method on a mock was
// executed.
type HaveMethodExecutedMatcher struct {
	MethodName string
	within     time.Duration
	args       []interface{}
	saveTo     []interface{}

	differ matchers.Differ
}

// HaveMethodExecuted returns a matcher that asserts that the method referenced
// by name was executed.  Options can modify the behavior of the matcher.
func HaveMethodExecuted(name string, opts ...HaveMethodExecutedOption) *HaveMethodExecutedMatcher {
	m := HaveMethodExecutedMatcher{MethodName: name, differ: diff.New()}
	for _, opt := range opts {
		m = opt(m)
	}
	return &m
}

// UseDiffer sets m to use d when showing a diff between actual and expected values.
func (m *HaveMethodExecutedMatcher) UseDiffer(d matchers.Differ) {
	m.differ = d
}

// Match checks the mock value v to see if it has a method matching m.MethodName
// which has been called.
func (m HaveMethodExecutedMatcher) Match(v interface{}) (interface{}, error) {
	mv := reflect.ValueOf(v)
	method, exists := mv.Type().MethodByName(m.MethodName)
	if !exists {
		return v, fmt.Errorf("pers: could not find method '%s' on type %T", m.MethodName, v)
	}
	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}
	calledField := mv.FieldByName(m.MethodName + "Called")
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: calledField},
	}
	switch m.within {
	case 0:
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
	default:
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(m.within))})
	}

	chosen, _, _ := reflect.Select(cases)
	if chosen == 1 {
		return v, fmt.Errorf("pers: expected method %s to have been called, but it was not", m.MethodName)
	}
	inputField := mv.FieldByName(m.MethodName + "Input")
	if !inputField.IsValid() {
		return v, nil
	}

	var calledWith []interface{}
	for i := 0; i < inputField.NumField(); i++ {
		fv, ok := inputField.Field(i).Recv()
		if !ok {
			return v, fmt.Errorf("pers: field %s is closed; cannot perform matches against this mock", inputField.Type().Field(i).Name)
		}
		calledWith = append(calledWith, fv.Interface())

		if m.saveTo != nil {
			reflect.ValueOf(m.saveTo[i]).Elem().Set(fv)
		}
	}
	if len(m.args) == 0 {
		return v, nil
	}

	args := append([]interface{}(nil), m.args...)
	if method.Type.IsVariadic() {
		lastTypeArg := method.Type.NumIn() - 1
		lastArg := lastTypeArg - 1 // lastTypeArg is including the receiver as an argument
		variadic := reflect.MakeSlice(method.Type.In(lastTypeArg), 0, 0)
		for i := lastArg; i < len(m.args); i++ {
			variadic = reflect.Append(variadic, reflect.ValueOf(m.args[i]))
		}
		args = append(args[:lastArg], variadic.Interface())
	}
	if len(args) != len(calledWith) {
		return v, fmt.Errorf("pers: expected %d arguments, but got %d", len(calledWith), len(args))
	}

	var argsDiff []string
	matched := true
	for i, a := range args {
		called := calledWith[i]
		format := formatFor(called)
		calledStr := fmt.Sprintf(format, called)
		switch src := a.(type) {
		case any:
			argsDiff = append(argsDiff, calledStr)
		case Matcher:
			_, err := src.Match(called)
			if err != nil {
				matched = false
				argsDiff = append(argsDiff, m.differ.Diff("", err.Error()))
				break
			}
			argsDiff = append(argsDiff, calledStr)
		default:
			if !reflect.DeepEqual(called, a) {
				matched = false
				argsDiff = append(argsDiff, fmt.Sprintf(format, m.differ.Diff(called, a)))
				break
			}
			argsDiff = append(argsDiff, calledStr)
		}
	}
	if matched {
		return v, nil
	}
	const msg = "pers: %s was called with incorrect arguments: [ %s ]"
	return v, fmt.Errorf(msg, m.MethodName, strings.Join(argsDiff, ", "))
}

// formatFor returns the format string that should be used for
// the passed in actual type.
func formatFor(actual interface{}) string {
	switch actual.(type) {
	case string:
		return `"%v"`
	default:
		return `%v`

	}
}
