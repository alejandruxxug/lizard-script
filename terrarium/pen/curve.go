package pen

import (
	"fmt"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"math"
)

// Curve is the response curve of one input device. A Force Touch trackpad
// needs a firm press before it registers a click, so its pressure starts
// high; the curve cuts that off and bends the rest.
//
// The pressure a program sees is clamp((raw - Low) / (High - Low), 0, 1) ** Gamma.
// Low and High pick the range of raw pressure that counts; a Gamma above 1
// keeps light pressures small, below 1 raises them.
type Curve struct {
	Low, High, Gamma float64
}

// Identity changes nothing.
var Identity = Curve{Low: 0, High: 1, Gamma: 1}

// Limits of a valid curve: 0 <= Low < High <= 1 and minGamma <= Gamma <= maxGamma.
const (
	minGamma = 0.1
	maxGamma = 10
)

// Apply maps a raw pressure from 0 to 1 to the pressure a program sees.
func (c Curve) Apply(raw float64) float64 {
	p := clamp01((raw - c.Low) / (c.High - c.Low))
	if c.Gamma == 1 {
		return p
	}
	return math.Pow(p, c.Gamma)
}

// Validate says why the curve can't be used, or returns nil.
func (c Curve) Validate() error {
	switch {
	case !(c.Low >= 0 && c.Low < 1):
		return fmt.Errorf("low must be from 0 up to (not including) 1, got %v", c.Low)
	case !(c.High > c.Low && c.High <= 1):
		return fmt.Errorf("high must be above low (%v) and at most 1, got %v", c.Low, c.High)
	case !(c.Gamma >= minGamma && c.Gamma <= maxGamma):
		return fmt.Errorf("gamma must be from %v to %v, got %v", minGamma, float64(maxGamma), c.Gamma)
	}
	return nil
}

// With is the curve with the keys of h ("low", "high", "gamma") replaced;
// the keys h leaves out keep their value. The result is validated.
func (c Curve) With(h *object.Hash) (Curve, error) {
	fields := map[string]*float64{"low": &c.Low, "high": &c.High, "gamma": &c.Gamma}
	var err error
	h.Each(func(pair object.HashPair) {
		key := pair.Key.Inspect()
		field, known := fields[key]
		n, isNumber := arg.Number(pair.Value)
		switch {
		case err != nil:
		case !known:
			err = fmt.Errorf("unknown key %q: a curve has low, high and gamma", key)
		case !isNumber:
			err = fmt.Errorf("%s must be a number, got %s", key, pair.Value.Type())
		default:
			*field = n
		}
	})
	if err != nil {
		return c, err
	}
	return c, c.Validate()
}

// Hash is the curve as a program sees it: {"low", "high", "gamma"}.
func (c Curve) Hash() *object.Hash {
	h := object.NewHash()
	arg.Set(h, "low", arg.Float(c.Low))
	arg.Set(h, "high", arg.Float(c.High))
	arg.Set(h, "gamma", arg.Float(c.Gamma))
	return h
}

// Sources are the kinds of input device a curve belongs to.
var Sources = []string{"trackpad", "tablet", "mouse"}

// defaultCurves are the curves before any setting. A Mac trackpad reaches
// about 90% of its range by the time a click registers, so it is dampened;
// the numbers are a starting point to tune (see the docs). A tablet and a
// mouse are left alone.
func defaultCurves() map[string]Curve {
	return map[string]Curve{
		"trackpad": {Low: 0.8, High: 1, Gamma: 1.5},
		"tablet":   Identity,
		"mouse":    Identity,
	}
}

// curves are the curves in use, by source. It is guarded by mu.
var curves = defaultCurves()

// CurveOf is the curve of a source, the identity for a source it doesn't know.
func CurveOf(source string) Curve {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := curves[source]; ok {
		return c
	}
	return Identity
}

// SetCurve replaces the curve of a known source.
func SetCurve(source string, c Curve) error {
	if err := checkSource(source); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	mu.Lock()
	curves[source] = c
	mu.Unlock()
	return nil
}

// Shape is the pressure a program sees for a raw pressure from a source.
func Shape(source string, raw float64) float64 { return CurveOf(source).Apply(raw) }

// checkSource fails for a source that has no curve.
func checkSource(source string) error {
	for _, s := range Sources {
		if s == source {
			return nil
		}
	}
	return fmt.Errorf("unknown source %q: use \"trackpad\", \"tablet\" or \"mouse\"", source)
}

// curveFunc is pen.curve(source) and pen.curve(source, hash).
func curveFunc(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	source, err := arg.String(args, 0, "source")
	if err != nil {
		return err
	}
	if e := checkSource(source); e != nil {
		return arg.Errorf("%v", e)
	}
	if len(args) == 2 {
		h, err := arg.Hash(args, 1, "curve")
		if err != nil {
			return err
		}
		c, e := CurveOf(source).With(h)
		if e == nil {
			e = SetCurve(source, c)
		}
		if e != nil {
			return arg.Errorf("%v", e)
		}
	}
	return CurveOf(source).Hash()
}
