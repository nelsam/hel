// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers

import (
	"fmt"
	"reflect"
	"strings"
	"time"
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
}

// HaveMethodExecuted returns a matcher that asserts that the method referenced
// by name was executed.  Options can modify the behavior of the matcher.
func HaveMethodExecuted(name string, opts ...HaveMethodExecutedOption) HaveMethodExecutedMatcher {
	m := HaveMethodExecutedMatcher{MethodName: name}
	for _, opt := range opts {
		m = opt(m)
	}
	return m
}

// Match checks the mock value v to see if it has a method matching m.MethodName
// which has been called.
func (m HaveMethodExecutedMatcher) Match(v interface{}) (interface{}, error) {
	mv := reflect.ValueOf(v)
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
	var matches [][]interface{}
	for i := 0; i < inputField.NumField(); i++ {
		fv, ok := inputField.Field(i).Recv()
		if !ok {
			return v, fmt.Errorf("pers: field %s is closed; cannot perform matches against this mock", inputField.Type().Field(i).Name)
		}
		matches = append(matches, []interface{}{fv.Interface()})
	}
	if len(m.args) == 0 {
		return v, nil
	}
	if len(m.args) != inputField.NumField() {
		return v, fmt.Errorf("pers: expected %d arguments, but got %d", inputField.NumField(), len(m.args))
	}
	matched := true
	for i, a := range m.args {
		matches[i] = append(matches[i], a)
		if matches[i][0] != a {
			matched = false
		}
	}
	if matched {
		return v, nil
	}
	msg := "pers: %s was called with (%s); expected (%s)"
	var actual, expected []string
	for _, match := range matches {
		format := "%#v"
		if match[0] != match[1] {
			format = ">%#v<"
		}
		actual = append(actual, fmt.Sprintf(format, match[0]))
		expected = append(expected, fmt.Sprintf(format, match[1]))
	}
	return v, fmt.Errorf(msg, m.MethodName, strings.Join(actual, ", "), strings.Join(expected, ", "))
}
