package interpreters

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Val struct {
	vb *bool
	vs *string
	vp *regexp.Regexp
	vn *int64
}

func BoxVal(a any) (Val, error) {
	if a == nil {
		return UndefinedVal(), nil
	}
	switch v := a.(type) {
	case bool:
		return Val{
			vb: &v,
		}, nil
	case string:
		return Val{
			vs: &v,
		}, nil
	case *regexp.Regexp:
		return Val{
			vp: v,
		}, nil
	case int:
		n := int64(v)
		return Val{
			vn: &n,
		}, nil
	case int8:
		n := int64(v)
		return Val{
			vn: &n,
		}, nil
	case int16:
		n := int64(v)
		return Val{
			vn: &n,
		}, nil
	case int32:
		n := int64(v)
		return Val{
			vn: &n,
		}, nil
	case int64:
		return Val{
			vn: &v,
		}, nil
	case float64:
		n := int64(v)
		return Val{
			vn: &n,
		}, nil
	default:
		return UndefinedVal(), fmt.Errorf("unable to box value %v", a)
	}
}

func BoxBool(b bool) Val {
	return Val{
		vb: &b,
	}
}

func UndefinedVal() Val {
	return Val{}
}

func (l Val) Undefined() bool {
	switch {
	case l.vb != nil:
		return false
	case l.vn != nil:
		return false
	case l.vs != nil:
		return false
	}
	return true
}

func (l Val) Unbox() any {
	switch {
	case l.vb != nil:
		return *l.vb
	case l.vs != nil:
		return *l.vs
	case l.vp != nil:
		return l.vp
	case l.vn != nil:
		return *l.vn
	default:
		return nil
	}
}

func (l Val) IsTruthy() bool {
	switch {
	// Undefined considered false
	case l.Undefined():
		return false
	// Booleans truthiness is their value
	case l.vb != nil:
		return *l.vb
	// All strings are truthy
	case l.vs != nil:
		return true
	// All regexps are truthy
	case l.vp != nil:
		return true
	// All numbers are truthy
	case l.vn != nil:
		return true
	// Fallback to false for any as yet unknown expr types
	default:
		return false
	}
}

func (l Val) Bool() (bool, bool) {
	if l.vb != nil {
		return *l.vb, true
	}
	return false, false
}

func (l Val) String() (string, bool) {
	if l.vs != nil {
		return *l.vs, true
	}
	return "", false
}

func (l Val) Regexp() (*regexp.Regexp, bool) {
	if l.vp != nil {
		return l.vp, true
	}
	return nil, false
}

func (l Val) Number() (int64, bool) {
	if l.vn != nil {
		return *l.vn, true
	}
	return 0, false
}

func (l Val) Equal(r Val) bool {
	// Go regexp structs store state. In the `expr` language a regexp can only be
	// used to match a value once, but the state values mean that two regexp
	// structs with the same pattern are not equal.
	// Instead when both operands are regexps, compare the source pattern
	// strings.
	lr, lok := l.Regexp()
	rr, rok := r.Regexp()

	if lok && rok {
		return lr.String() == rr.String()
	}

	return l.Unbox() == r.Unbox()
}

func (l Val) StartsWith(r Val) (bool, error) {
	left, lok := l.String()
	right, rok := r.String()
	if !lok || !rok {
		return false, fmt.Errorf("need string operands, got (%v, %v)", left, right)
	}

	return strings.HasPrefix(left, right), nil
}

func (l Val) Greater(r Val) (bool, error) {
	left, lok := l.Number()
	right, rok := r.Number()
	if !lok || !rok {
		return false, fmt.Errorf("need number operands, got (%v, %v)", left, right)
	}

	return left > right, nil
}

func (l Val) GreaterEqual(r Val) (bool, error) {
	left, lok := l.Number()
	right, rok := r.Number()
	if !lok || !rok {
		return false, fmt.Errorf("need number operands, got (%v, %v)", left, right)
	}

	return left >= right, nil
}

func (l Val) Less(r Val) (bool, error) {
	left, lok := l.Number()
	right, rok := r.Number()
	if !lok || !rok {
		return false, fmt.Errorf("need number operands, got (%v, %v)", left, right)
	}

	return left < right, nil
}

func (l Val) LessEqual(r Val) (bool, error) {
	left, lok := l.Number()
	right, rok := r.Number()
	if !lok || !rok {
		return false, fmt.Errorf("need number operands, got (%v, %v)", left, right)
	}

	return left <= right, nil
}

func (v *Val) UnmarshalJSON(input []byte) error {
	var value any
	err := json.Unmarshal(input, &value)
	if err != nil {
		return err
	}

	if val, ok := value.(string); ok {
		if strings.HasPrefix(val, "corpus/regexp:") {
			pattern := strings.TrimPrefix(val, "corpus/regexp:")

			var err error
			value, err = regexp.Compile(pattern)
			if err != nil {
				return err
			}
		}
	}

	boxed, err := BoxVal(value)
	if err != nil {
		return err
	}

	v.vb = boxed.vb
	v.vs = boxed.vs
	v.vp = boxed.vp
	v.vn = boxed.vn

	return nil
}
