package eval

import (
	"fmt"
	"time"
)

func callTimeParse(args []any) (any, error) {
	if err := checkArgs("time.Parse", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(string)
	arg1, ok1 := args[1].(string)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.Parse(), invalid args %v", args)
	}
	return time.Parse(arg0, arg1)
}

func callTimeFormat(args []any) (any, error) {
	if err := checkArgs("time.Format", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	arg1, ok1 := args[1].(string)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.Format(), invalid args %v", args)
	}
	return arg0.Format(arg1), nil
}

func callTimeFixedZone(args []any) (any, error) {
	if err := checkArgs("time.FixedZone", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(string) // Name
	arg1, ok1 := args[1].(int64)  // Offset in min
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.FixedZone(), invalid args %v", args)
	}
	return time.FixedZone(arg0, int(arg1)), nil
}

func callTimeDate(args []any) (any, error) {
	if err := checkArgs("time.Date", 8, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(int64)      // Year
	arg1, ok1 := args[1].(time.Month) // Month
	arg2, ok2 := args[2].(int64)      // Day
	arg3, ok3 := args[3].(int64)      // Hour
	arg4, ok4 := args[4].(int64)      // Min
	arg5, ok5 := args[5].(int64)      // Sec
	arg6, ok6 := args[6].(int64)      // Nsec
	arg7, ok7 := args[7].(*time.Location)
	if !ok0 || !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 {
		return nil, fmt.Errorf("cannot evaluate time.Date(), invalid args %v", args)
	}
	return time.Date(int(arg0), arg1, int(arg2), int(arg3), int(arg4), int(arg5), int(arg6), arg7), nil
}

func callTimeDiffMilli(args []any) (any, error) {
	if err := checkArgs("time.DiffMilli", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	arg1, ok1 := args[1].(time.Time)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.DiffMilli(), invalid args %v", args)
	}

	return arg0.Sub(arg1).Milliseconds(), nil
}

func callTimeNow(args []any) (any, error) {
	if err := checkArgs("time.Now", 0, len(args)); err != nil {
		return nil, err
	}
	return time.Now(), nil
}

func callTimeUnix(args []any) (any, error) {
	if err := checkArgs("time.Unix", 1, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	if !ok0 {
		return nil, fmt.Errorf("cannot evaluate time.Unix(), invalid args %v", args)
	}

	return arg0.Unix(), nil
}

func callTimeUnixMilli(args []any) (any, error) {
	if err := checkArgs("time.UnixMilli", 1, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	if !ok0 {
		return nil, fmt.Errorf("cannot evaluate time.UnixMilli(), invalid args %v", args)
	}

	return arg0.UnixMilli(), nil
}

func callTimeBefore(args []any) (any, error) {
	if err := checkArgs("time.Before", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	arg1, ok1 := args[1].(time.Time)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.Before(), invalid args %v", args)
	}

	return arg0.Before(arg1), nil
}

func callTimeAfter(args []any) (any, error) {
	if err := checkArgs("time.After", 2, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(time.Time)
	arg1, ok1 := args[1].(time.Time)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot evaluate time.After(), invalid args %v", args)
	}

	return arg0.After(arg1), nil
}
