package terrarium

import (
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"time"
)

// time: clocks and waiting.

// start is when the program started, for time.ms.
var start = time.Now()

// maxSleep is the longest time.sleep, an hour, so a mistake in the units
// doesn't hang a program for days.
const maxSleep = 60 * 60 * 1000

func timeModule() *object.Hash {
	return arg.Module("time", map[string]object.BuiltinFunction{
		"ms":    timeMs,
		"now":   timeNow,
		"sleep": timeSleep,
	})
}

// time.ms() is the milliseconds since the program started, as a float. It
// only goes forward, even when the system clock is changed, so it is the one
// to measure with.
func timeMs(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return arg.Float(float64(time.Since(start).Microseconds()) / 1000)
}

// time.now() is the local date and time as a hash: year, month (1 to 12),
// day, hour (0 to 23), minute and second.
func timeNow(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return dateHash(time.Now())
}

// dateHash is the hash of time.now for t.
func dateHash(t time.Time) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "year", arg.Int64(int64(t.Year())))
	arg.Set(h, "month", arg.Int64(int64(t.Month())))
	arg.Set(h, "day", arg.Int64(int64(t.Day())))
	arg.Set(h, "hour", arg.Int64(int64(t.Hour())))
	arg.Set(h, "minute", arg.Int64(int64(t.Minute())))
	arg.Set(h, "second", arg.Int64(int64(t.Second())))
	return h
}

// time.sleep(ms) waits that many milliseconds (an integer or a float, at
// most an hour).
func timeSleep(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	ms, err := arg.NumberArg(args, 0, "of milliseconds")
	if err != nil {
		return err
	}
	if ms < 0 || ms > maxSleep {
		return arg.Errorf("the milliseconds must be from 0 to %d, got %s", maxSleep, args[0].Inspect())
	}
	time.Sleep(time.Duration(ms * float64(time.Millisecond)))
	return arg.Null
}
