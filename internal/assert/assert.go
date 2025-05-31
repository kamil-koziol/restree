package assert

import (
	"fmt"
	"testing"
)

func Assert(t *testing.T, c bool, msg string) {
	if !c {
		t.Error(msg)
		t.FailNow()
	}
}

func Eq(t *testing.T, expected any, is any) {
	Assert(t, is == expected, fmt.Sprintf("'%s' != '%s'", expected, is))
}

func Neq(t *testing.T, expected any, is any) {
	Assert(t, is != expected, fmt.Sprintf("'%s' == '%s'", expected, is))
}
