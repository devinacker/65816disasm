// 1 june 2013
package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"strconv"
)

// TODO better name than ui?

var commands = []struct {
	Name	string
	Desc		string
	Func		func(fields []string)
}{
	{ "help", "show this help", c_help },
	{ "doauto", "auto-analyze vectors", c_doauto },		// TODO keep "vectors"?
	{ "specialsub", "mark a subroutine as doing something special", c_specialsub },
	{ "dowordptr", "mark a word as a pointer to code (with a pre-existing environment)", c_dowordptr },
	{ "disasm", "disassemble code at a given location", c_disasm },
}

// the map key is a logical address
// TODO this will need to be made part of MemoryMap later if it branches out from SNES ROMs
var vectorLocs = map[uint32]string{
	0x00FFFE:	"EmuIRQBRK",
	0x00FFFC:	"EmuRESET",
	0x00FFFA:	"EmuNMI",
	0x00FFF8:	"EmuABORT",
	0x00FFF6:	"EmuReserved1",
	0x00FFF4:	"EmuCOP",
	0x00FFF2:	"EmuReserved2",
	0x00FFF0:	"EmuReserved3",
	0x00FFEE:	"NativeIRQ",
	0x00FFEC:	"NativeReserved1",
	0x00FFEA:	"NativeNMI",
	0x00FFE8:	"NativeABORT",
	0x00FFE6:	"NativeBRK",
	0x00FFE4:	"NativeCOP",
	0x00FFE2:	"NativeReserved2",
	0x00FFE0:	"NativeReserved3",
}

func c_doauto(fields []string) {
	for addr, label := range vectorLocs {
		pos, inROM := memmap.Physical(addr)				// addr is a logical address
		if !inROM {
			errorf("sanity check failure: vector %s ($%06X) not in ROM (memmap.Physical() returned $%06X)", label, addr, pos)
		}
		posw, _ := getword(pos)
		pos, inROM = memmap.Physical(uint32(posw))		// always bank 0
		if !inROM {
			fmt.Fprintf(os.Stderr, "physical address for %s vector ($%06X) not in ROM\n", label, uint32(posw))
			continue
		}
		if labels[pos] != "" {		// if already defined as a different vector, concatenate the labels to make sure everything is represented
			// TODO because this uses a map, it will not be in vector order
			labels[pos] = labels[pos] + ":\n" + label
		} else {
			labels[pos] = label
		}
		labelpriorities[pos] = lpSub
		env.pbr = 0			// we execute from bank 0
		disassemble(pos)
	}
	fmt.Fprintf(os.Stderr, "finished auto-analyzing vectors\n")
}

func c_dowordptr(fields []string) {
	if len(fields) != 2 {
		fmt.Fprintf(os.Stderr, "dowordptr usage: dowordptr word-address env-address\n")
		return
	}

	// TODO addr and envaddr must be bare hex numbers with this
	addr64, err := strconv.ParseUint(fields[0], 16, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dowordptr error: invalid address hex number %q: %v", fields[0], err)
		return
	}
	addr := uint32(addr64)

	envaddr64, err := strconv.ParseUint(fields[1], 16, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "specialsubs error: invalid environment address hex number %q (for $%06X): %v", fields[1], addr, err)
		return
	}
	envaddr := uint32(envaddr64)

	if addr + 1 > uint32(len(bytes)) {
		fmt.Fprintf(os.Stderr, "specialsubs error: address $%06X not in ROM\n", addr)
		return
	}
	npbase := (uint16(bytes[addr + 1]) << 8) | uint16(bytes[addr])

	if env, ok := savedenvs[envaddr]; ok {
		nplogical := (uint32(env.pbr) << 16) | uint32(npbase)
		npaddr, inROM := memmap.Physical(nplogical)
		if !inROM {
			fmt.Fprintf(os.Stderr, "dowordptr error: new address $%06X (from $%06X) not in ROM\n", nplogical, addr)
			return
		}
		mklabel(npaddr, "loc", lpLoc)
		restoreenv(env)
		disassemble(npaddr)
		labelplaces[addr] = npaddr
		instructions[addr] = "dc.w\t(%s & 0xFFFF)"
		instructions[addr + 1] = operandString
	} else {
		fmt.Fprintf(os.Stderr, "dowordptr error: no environment available for environment $%06X (from $%06X)\n", envaddr, addr)
		return
	}
}

func c_disasm(fields []string) {
	if len(fields) == 0 {
		fmt.Fprintf(os.Stderr, "disasm usage: dowordptr address [e=0/1/?] [m=0/1/?] [x=0/1/?] [c=0/1/?] [a=0000-ffff/?] [dp=0000-ffff/?] [db=00-ff/?]\n")
		return
	}
	
	address, _ := strconv.ParseUint(fields[0], 16, 32)
	
	saved := saveenv()
	env = newenv()
	
	for _, v := range fields[1:] {
		split := strings.Split(v, "=")
		if len(split) != 2 {
			fmt.Fprintf(os.Stderr, "disasm: invalid parameter %s\n", v)
			return
		}
		
		reg, val := split[0], split[1]
		
		switch {
		case reg == "e" && val == "?":
			env.e.known = false
		case reg == "e":
			ival , _ := strconv.ParseUint(val, 16, 8)
			env.e.value = uint8(ival)
			env.e.known = true
			
		case reg == "m" && val == "?":
			env.m.known = false
		case reg == "m":
			ival , _ := strconv.ParseUint(val, 16, 8)
			env.m.value = uint8(ival)
			env.m.known = true
			
		case reg == "x" && val == "?":
			env.x.known = false
		case reg == "x":
			ival , _ := strconv.ParseUint(val, 16, 8)
			env.x.value = uint8(ival)
			env.x.known = true
			
		case reg == "c" && val == "?":
			env.carryflag.known = false
		case reg == "c":
			ival , _ := strconv.ParseUint(val, 16, 8)
			env.carryflag.value = uint8(ival)
			env.carryflag.known = true
			
		case reg == "a" && val == "?":
			env.a.known = false
		case reg == "a":
			ival , _ := strconv.ParseUint(val, 16, 16)
			env.a.value = uint16(ival)
			env.a.known = true
			
		case reg == "dp" && val == "?":
			env.direct.known = false
		case reg == "dp":
			ival , _ := strconv.ParseUint(val, 16, 16)
			env.direct.value = uint16(ival)
			env.direct.known = true
			
		case reg == "db" && val == "?":
			env.dbr.known = false
		case reg == "db":
			ival , _ := strconv.ParseUint(val, 16, 8)
			env.dbr.value = uint8(ival)
			env.dbr.known = true
		}
	}
	
	logical, _ := memmap.Logical(uint32(address))
	env.pbr = uint8(logical >> 16)
	mklabel(uint32(address), "sub", lpSub)
	disassemble(uint32(address))
	restoreenv(saved)
}

var helptext string

func c_help(fields []string) {
	fmt.Fprintf(os.Stderr, "%s", helptext)
}

func init() {
	for _, v := range commands {
		helptext += fmt.Sprintf("%10s - %s\n", v.Name, v.Desc)
	}
}

func doui() {
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		line := stdin.Text()
		fields := strings.Fields(line)
		// TODO this means comments cannot start in the middle of a token
		lastValid := 0					// strip comments
		for ; lastValid < len(fields); lastValid++ {
			if fields[lastValid][0] == '#' {
				break
			}
		}
		fields = fields[:lastValid]
		if len(fields) == 0 {				// blank line
			continue
		}
		command := fields[0]
		found := false
		for _, v := range commands {
			if command == v.Name {
				v.Func(fields[1:])
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "command not found\n")
		}
	}
	if err := stdin.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading standard input: %v\n", err)
		os.Exit(1)
	}
}
