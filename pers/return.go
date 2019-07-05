// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers

// Return will add a given value to the channel or struct of channels.
// This isn't very useful with a single value, so it's intended more
// to support structs full of channels, such as the ones that hel
// generates for return values in its mocks.
func Return(mock interface{}, args ...interface{}) error {
	cases, err := selectCases(mock, args...)
	if err != nil {
		return err
	}
	for _, c := range cases {
		c.Chan.Send(c.Send)
	}
	return nil
}
