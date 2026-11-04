package terrarium

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
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
		"utc":   timeUTC,
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
// day, hour (0 to 23), minute, second, weekday (1 for Monday to 7 for
// Sunday), epoch (the seconds since 1970-01-01 UTC, a float with the
// fractions) and utc_offset (the seconds the local time is ahead of UTC,
// negative when behind).
func timeNow(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return dateHash(time.Now())
}

// time.utc() is time.now in UTC: the same hash, with a utc_offset of 0.
func timeUTC(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return dateHash(time.Now().UTC())
}

// dateHash is the hash of time.now for t, in t's own time zone.
func dateHash(t time.Time) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "year", arg.Int64(int64(t.Year())))
	arg.Set(h, "month", arg.Int64(int64(t.Month())))
	arg.Set(h, "day", arg.Int64(int64(t.Day())))
	arg.Set(h, "hour", arg.Int64(int64(t.Hour())))
	arg.Set(h, "minute", arg.Int64(int64(t.Minute())))
	arg.Set(h, "second", arg.Int64(int64(t.Second())))
	weekday := int64(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday ends the week
	}
	arg.Set(h, "weekday", arg.Int64(weekday))
	arg.Set(h, "epoch", arg.Float(float64(t.UnixMicro())/1e6))
	_, offset := t.Zone()
	arg.Set(h, "utc_offset", arg.Int64(int64(offset)))
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
