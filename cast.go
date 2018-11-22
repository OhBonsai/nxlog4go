package nxlog4go

import (
	"time"
	"strconv"
)

// ToString casts an interface to a string type.
func ToString(i interface{}) (s string, err error) {
	s = ""
	err = nil

	switch  i.(type) {
	case string:
		s = i.(string)
	case []byte:
		s = string(i.([]byte))
	default:
		err = ErrBadValue
	}
	return
}

// ToBool casts an interface to a bool type.
func ToBool(i interface{}) (b bool, err error) {
	b = false
	err = nil
	
	switch i.(type) {
	case bool:
		b = i.(bool)
	case int:
		if i.(int) > 0 {
			b = true
		}
	case string:
		return strconv.ParseBool(i.(string))
	default:
		err = ErrBadValue
	}
	return
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func strToNumSuffix(str string, mult int) (int, error) {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, err := strconv.Atoi(str)
	return parsed * num, err
}

// ToInt casts an interface to an int type.
func ToInt(i interface{}) (n int, err error) {
	n = 0
	err = nil
	
	switch i.(type) {
	case int:
		n = i.(int)
	case string:
		n, err = strToNumSuffix(i.(string), 1024)
	default:
		err = ErrBadValue
	}
	return
}

// ToSeconds casts an interface to an seconds.
func ToSeconds(i interface{}) (s int, err error) {
	s = 0
	err = nil

	switch i.(type) {
	case int:
		s = i.(int)
	case string:
		dur_int64, err0 := time.ParseDuration(i.(string))
		s = int(dur_int64/time.Second)
		err = err0
	default:
		err = ErrBadValue
	}
	return
}

