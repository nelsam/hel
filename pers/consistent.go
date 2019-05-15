// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers

import (
	"errors"
	"fmt"
	"reflect"
)

// ConsistentlyReturn will continue adding a given value to the channel
// until the returned done function is called.  You may pass in either
// a channel (in which case you should pass in a single argument) or a
// struct full of channels (in which case you should pass in arguments
// in the order the fields appear in the struct).
//
// After the returned function is called, you will still need to drain
// any remaining calls from the channel(s) before it will start blocking
// again.
func ConsistentlyReturn(mock interface{}, args ...interface{}) (func(), error) {
	cases, err := selectCases(mock, args...)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	exited := make(chan struct{})
	go consistentlyReturn(cases, done, exited, args...)
	return func() {
		close(done)
		<-exited
	}, nil
}

func selectCases(mock interface{}, args ...interface{}) ([]reflect.SelectCase, error) {
	v := reflect.ValueOf(mock)
	switch v.Kind() {
	case reflect.Chan:
		if len(args) != 1 {
			return nil, fmt.Errorf("expected 1 argument for %#v; got %d", mock, len(args))
		}
		return []reflect.SelectCase{{Dir: reflect.SelectSend, Chan: v, Send: reflect.ValueOf(args[0])}}, nil
	case reflect.Struct:
		if v.NumField() == 0 {
			return nil, errors.New("cannot consistently return on unsupported type struct{}")
		}
		if len(args) != v.NumField() {
			argString := "argument"
			if v.NumField() != 1 {
				argString = "arguments"
			}
			return nil, fmt.Errorf("expected %d %s for %#v; got %d", v.NumField(), argString, mock, len(args))
		}
		var cases []reflect.SelectCase
		for i := 0; i < v.NumField(); i++ {
			c, err := selectCases(v.Field(i).Interface(), args[i])
			if err != nil {
				return nil, err
			}
			cases = append(cases, c...)
		}
		return cases, nil
	default:
		return nil, fmt.Errorf("cannot consistently return on unsupported type %T", mock)
	}
}

func consistentlyReturn(cases []reflect.SelectCase, done, exited chan struct{}, args ...interface{}) {
	defer close(exited)
	doneIdx := len(cases)
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(done)})
	for {
		chosen, _, _ := reflect.Select(cases)
		if chosen == doneIdx {
			return
		}
	}
}
