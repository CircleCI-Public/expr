package interpreter

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Val struct {
	vb *bool
	vs *string
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

func (l Val) Number() (int64, bool) {
	if l.vn != nil {
		return *l.vn, true
	}
	return 0, false
}

func (l Val) Equal(r Val) bool {
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
	var val any
	err := json.Unmarshal(input, &val)
	if err != nil {
		return err
	}

	boxed, err := BoxVal(val)
	if err != nil {
		return err
	}

	v.vb = boxed.vb
	v.vs = boxed.vs
	v.vn = boxed.vn

	return nil
}
