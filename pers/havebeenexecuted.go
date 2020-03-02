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

	matched := true
	var actual, expected []string
	for i, a := range args {
		called := calledWith[i]
		match, a, e := diff(called, a)
		actual = append(actual, a)
		expected = append(expected, e)
		matched = matched && match
	}
	if matched {
		return v, nil
	}
	msg := "pers: %s was called with (%s); expected (%s)"
	return v, fmt.Errorf(msg, m.MethodName, strings.Join(actual, ", "), strings.Join(expected, ", "))
}

func diff(actual, expected interface{}) (matched bool, actualOutput, expectedOutput string) {
	return diffV(reflect.ValueOf(actual), reflect.ValueOf(expected))
}

type span struct {
	start, end int
}

func diffV(av, ev reflect.Value) (matched bool, actualOutput, expectedOutput string) {
	if av.Kind() != ev.Kind() {
		format := ">type mismatch: %#v<"
		return false, fmt.Sprintf(format, av.Interface()), fmt.Sprintf(format, ev.Interface())
	}
	if av.Type().Comparable() {
		matched := true
		format := "%#v"
		if av.Interface() != ev.Interface() {
			matched = false
			format = ">%#v<"
		}
		return matched, fmt.Sprintf(format, av.Interface()), fmt.Sprintf(format, ev.Interface())
	}

	switch av.Interface().(type) {
	case []rune, []byte:
		// make almost-string types a little prettier, when possible.
		if av.Len() != ev.Len() {
			break // let the default logic handle this
		}

		strTyp := reflect.TypeOf("")
		matchSection := true
		matched := true
		var outa, oute string
		for i := 0; i < av.Len(); i++ {
			match, _, _ := diffV(av.Index(i), ev.Index(i))
			if !match && matchSection {
				outa += ">"
				oute += ">"
			}
			if match && !matchSection {
				outa += "<"
				oute += "<"
			}
			matchSection = match
			matched = matched && match
			outa += av.Index(i).Convert(strTyp).Interface().(string)
			oute += ev.Index(i).Convert(strTyp).Interface().(string)
		}
		return matched, outa, oute
	}

	switch av.Kind() {
	case reflect.Ptr, reflect.Interface:
		return diffV(av.Elem(), ev.Elem())
	case reflect.Slice, reflect.Array:
		if av.Len() != ev.Len() {
			// TODO: do a more thorough diff of values
			format := ">%T(length %d)<"
			return false, fmt.Sprintf(format, av.Interface(), av.Len()), fmt.Sprintf(format, ev.Interface(), ev.Len())
		}
		format := func(parts []string) string {
			return "[" + strings.Join(parts, ",") + "]"
		}
		var aParts, eParts []string
		matched := true
		for i := 0; i < av.Len(); i++ {
			match, a, e := diffV(av.Index(i), ev.Index(i))
			matched = matched && match
			aParts = append(aParts, a)
			eParts = append(eParts, e)
		}
		return matched, format(aParts), format(eParts)
	case reflect.Map:
		format := func(parts []string) string {
			return "{" + strings.Join(parts, ",") + "}"
		}
		var aParts, eParts []string
		matched := true
		for _, kv := range ev.MapKeys() {
			k := kv.Interface()
			emv := ev.MapIndex(kv)
			amv := av.MapIndex(kv)
			if !amv.IsValid() {
				aParts = append(aParts, fmt.Sprintf(">missing key %v<", k))
				eParts = append(eParts, fmt.Sprintf(">%v: %v<", k, emv.Interface()))
				continue
			}
			match, a, e := diffV(amv, emv)
			matched = matched && match
			aParts = append(aParts, fmt.Sprintf("%v: %s", k, a))
			eParts = append(eParts, fmt.Sprintf("%v: %s", k, e))
		}
		for _, kv := range av.MapKeys() {
			// We've already compared all keys that exist in both maps; now we're
			// just looking for keys that only exist in the actual.
			k := kv.Interface()
			if !ev.MapIndex(kv).IsValid() {
				matched = false
				aParts = append(aParts, fmt.Sprintf(">extra key %v: %v<", k, av.MapIndex(kv).Interface()))
				eParts = append(eParts, fmt.Sprintf(">%v: nil<", k))
				continue
			}
		}
		return matched, format(aParts), format(eParts)
	case reflect.Struct:
		if av.Type().Name() != ev.Type().Name() {
			return false, ">" + av.Type().Name() + "(mismatched types)<", ">" + ev.Type().Name() + "(mismatched types)<"
		}
		format := func(parts []string) string {
			return fmt.Sprintf("%T{\n", av.Interface()) + strings.Join(parts, ",\n") + "}"
		}
		var aParts, eParts []string
		matched := true
		for i := 0; i < ev.NumField(); i++ {
			name := ev.Type().Field(i).Name
			efv := ev.Field(i)
			afv := av.Field(i)
			match, a, e := diffV(afv, efv)
			matched = matched && match
			aParts = append(aParts, fmt.Sprintf("%s: %s", name, a))
			eParts = append(eParts, fmt.Sprintf("%s: %s", name, e))
		}
		return matched, format(aParts), format(eParts)
	default:
		msg := fmt.Sprintf("> UNSUPPORTED: could not compare values of type %T <", av.Interface())
		return false, msg, msg
	}
}
