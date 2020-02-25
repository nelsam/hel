// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers

import (
	"fmt"
	"reflect"
	"time"

	"github.com/poy/onpar/diff"
	"github.com/poy/onpar/matchers"
)

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

// HaveMethodExecutedMatcher is a matcher to ensure that a method on a mock was
// executed.
type HaveMethodExecutedMatcher struct {
	MethodName string
	within     time.Duration
	args       []interface{}
	differ     matchers.Differ
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
	var calledWith []interface{}
	for i := 0; i < inputField.NumField(); i++ {
		fv, ok := inputField.Field(i).Recv()
		if !ok {
			return v, fmt.Errorf("pers: field %s is closed; cannot perform matches against this mock", inputField.Type().Field(i).Name)
		}
		calledWith = append(calledWith, fv.Interface())
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

	matched := true
	for i, a := range args {
		called := calledWith[i]
		matched = matched && reflect.DeepEqual(called, a)
	}
	if matched {
		return v, nil
	}
	msg := "pers: %s was called with incorrect arguments: %s"
	return v, fmt.Errorf(msg, m.MethodName, m.differ.Diff(calledWith, args))
}
