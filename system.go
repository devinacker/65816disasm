// 25 may 2013
// modified from huc6280disasm
package main

type knownbyte struct {
	value	byte
	known	bool
}

type knownword struct {
	value	uint16
	known	bool
}

type envt struct {
	a		knownword
	carryflag	knownword
	direct	knownword
	dbr		knownbyte
	stack	[]knownbyte
	m		knownbyte	// 16-bit accumulator flag
	x		knownbyte	// 16-bit index register flag
}

var env *envt

func newenv() *evnt {
	e := new(envt)
	// TODO verify this
	e.a.known = false
	e.carryflag.value = 0
	e.carryflag.known = false
	// this part comes from the datasheet
	e.direct.value = 0
	e.direct.known = true
	e.dbr.value = 0
	e.dbr.known = true
	return e
}

func init() {
	env = newenv()
}

const (
	carryflagbit = (1 << 0)
	mflagbit = (1 << 5)
	xflagbit = (1 << 4)
)

func getp() (p byte, err error) {
	if !env.carryflag.known {
		return 0, fmt.Errorf("cannot get p: carry flag not known")
	}
	if env.carryflag.value != 0 {
		p |= carryflagbit
	}
	if !env.m.known {
		return 0, fmt.Errorf("cannot get p: m not known")
	}
	if env.m.value != 0 {
		p |= mflagbit
	}
	if !env.x.known {
		return 0, fmt.Errorf("cannot get p: x not known")
	}
	if env.x.value != 0 {
		p |= xflagbit
	}
	return p
}

func setp(p byte, known bool) {
	env.carryflag.value = p & carryflagbit
	env.carryflag.known = known
	env.m.value = p & mflagbit
	env.m.known = known
	env.x.value = p & xflagbit
	env.x.known = known
}

func makeAUnknown() {
	env.a.known = false
	env.carryflag.known = false
}

func pushbyte(value byte, known bool) {
	env.stack = append(env.stack, knownbyte{
		value:	value,
		known:	known,
	})
}

func pushword(value uint16, known bool {
	pushbyte(byte((value >> 8) & 0xFF), known)
	pushbyte(byte(value & 0xFF), known)
}

func popbyte() (value byte, known bool) {
	if len(env.stack) == 0 {
		return 0, false	// TODO correct?
	}
	t := env.stack[len(env.stack) - 1]
	stack = env.stack[:len(env.stack) - 1]
	return t.value, t.known
}

func popword() (value word, known bool) {
	a, ak := popbyte()		// low byte
	b, bk := popbyte()		// high byte
	return uint16(a) | (uint16(b) << 8),
		(ak && bk)			// both must be known
}

func saveenv() *envt {
	e := new(envt)
	e.a = env.a
	e.carryflag = env.carryflag
	e.direct = env.direct
	e.dbr = env.dbr
	e.stack = make([]knownbyte, len(env.stack))
	copy(e.stack, env.stack)
	e.m = env.m
	e.x = env.x
	return e
}

func restoreenv(e *envt) {
	env = e
}
