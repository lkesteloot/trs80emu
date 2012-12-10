package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Copy and pasted from z80.txt (http://guide.ticalc.org/download/z80.txt)
var instructionList string = `
ADC A,(HL)    7     1   +0V+++  8E
ADC A,(IX+N)  19    3   +0V+++  DD 8E XX
ADC A,(IY+N)  19    3   +0V+++  FD 8E XX
ADC A,r       4     1   +0V+++  88+r
ADC A,HX            2   +0V+++  DD 8C
ADC A,HY            2   +0V+++  FD 8C
ADC A,LX            2   +0V+++  DD 8D
ADC A,LY            2   +0V+++  FD 8D
ADC A,N       7     2   +0V+++  CE XX
ADC HL,BC     15    2   +0V ++  ED 4A
ADC HL,DE     15    2   +0V ++  ED 5A
ADC HL,HL     15    2   +0V ++  ED 6A
ADC HL,SP     15    2   +0V ++  ED 7A
ADD A,(HL)    7     1   +0V+++  86
ADD A,(IX+N)  19    3   +0V+++  DD 86 XX
ADD A,(IY+N)  19    3   +0V+++  FD 86 XX
ADD A,r       4     1   +0V+++  8r
ADD A,HX            2   +0V+++  DD 84
ADD A,HY            2   +0V+++  FD 84
ADD A,LX            2   +0V+++  DD 85
ADD A,LY            2   +0V+++  FD 85
ADD A,N       7     2   +0V+++  C6 XX
ADD HL,BC     11    1   +0- --  09
ADD HL,DE     11    1   +0- --  19
ADD HL,HL     11    1   +0- --  29
ADD HL,SP     11    1   +0- --  39
ADD IX,BC     15    2   +0- --  DD 09
ADD IX,DE     15    2   +0- --  DD 19
ADD IX,IX     15    2   +0- --  DD 29
ADD IX,SP     15    2   +0- --  DD 39
ADD IY,BC     15    2   +0- --  FD 09
ADD IY,DE     15    2   +0- --  FD 19
ADD IY,IY     15    2   +0- --  FD 29
ADD IY,SP     15    2   +0- --  FD 39
AND (HL)      7     1   00P1++  A6
AND (IX+N)    19    3   00P1++  DD A6 XX
AND (IY+N)    19    3   00P1++  FD A6 XX
AND r         4     1   00P1++  Ar
AND HX              2   00P1++  DD A4
AND HY              2   00P1++  FD A4
AND LX              2   00P1++  DD A5
AND LY              2   00P1++  FD A5
AND N         7     2   00P1++  E6 XX
BIT b,(HL)    12    2   -0 1+   CB 46+8*b
BIT b,(IX+N)  20    4   -0 1+   DD CB XX 46+8*b
BIT b,(IY+N)  20    4   -0 1+   FD CB XX 46+8*b
BIT b,r       8     2   -0 1+   CB 4r+8*b
CALL C,NN     17/10 3   ------  DC XX XX
CALL M,NN     17/10 3   ------  FC XX XX
CALL NC,NN    17/10 3   ------  D4 XX XX
CALL NN       17    3   ------  CD XX XX
CALL NZ,NN    17/10 3   ------  C4 XX XX
CALL P,NN     17/10 3   ------  F4 XX XX
CALL PE,NN    17/10 3   ------  EC XX XX
CALL PO,NN    17/10 3   ------  E4 XX XX
CALL Z,NN     17/10 3   ------  CC XX XX
CCF           4     1   +0- --  3F
CP (HL)       7     1   +1V+++  BE
CP (IX+N)     19    3   +1V+++  DD BE XX
CP (IY+N)     19    3   +1V+++  FD BE XX
CP r          4     1   +1V+++  B8+r
CP HX               2   +1V+++  DD BC
CP HY               2   +1V+++  FD BC
CP LX               2   +1V+++  DD BD
CP LY               2   +1V+++  FD BD
CP N          7     2   +1V+++  FE XX
CPD           16    2   -1++++  ED A9
CPDR          21/16 2   -1++++  ED B9
CPI           16    2   -1++++  ED A1
CPIR          21/16 2   -1++++  ED B1
CPL           4     1   -1-1--  2F
DAA           4     1   +-P+++  27
DEC (HL)      11    1   -1V+++  35
DEC (IX+N)    23    3   -1V+++  DD 35 XX
DEC (IY+N)    23    3   -1V+++  FD 35 XX
DEC A         4     1   -1V+++  3D
DEC B         4     1   -1V+++  05
DEC BC        6     1   ------  0B
DEC C         4     1   -1V+++  0D
DEC D         4     1   -1V+++  15
DEC DE        6     1   ------  1B
DEC E         4     1   -1V+++  1D
DEC H         4     1   -1V+++  25
DEC HL        6     1   ------  2B
DEC IX        10    2   ------  DD 2B
DEC IY        10    2   ------  FD 2B
DEC L         4     2   -1V+++  2D
DEC SP        6     1   ------  3B
DI            4     1   ------  F3
DJNZ $N+2     13/8  2   ------  10 XX
EI            4     1   ------  FB
EX (SP),HL    19    1   ------  E3
EX (SP),IX    23    2   ------  DD E3
EX (SP),IY    23    2   ------  FD E3
EX AF,AF'     4     1   ------  08
EX DE,HL      4     1   ------  EB
EXX           4     1   ------  D9
HALT          4+    1   ------  76
IM 0          8     2   ------  ED 46
IM 1          8     2   ------  ED 56
IM 2          8     2   ------  ED 5E
IN A,(C)      12    2   -0P+++  ED 78
IN A,(N)      11    2   ------  DB XX
IN B,(C)      12    2   -0P+++  ED 40
IN C,(C)      12    2   -0P+++  ED 48
IN D,(C)      12    2   -0P+++  ED 50
IN E,(C)      12    2   -0P+++  ED 58
IN H,(C)      12    2   -0P+++  ED 60
IN L,(C)      12    2   -0P+++  ED 68
IN (C)        12    2   -0P+++  ED 70
INC (HL)      11    1   - V +   34
INC (IX+N)    23    3   - V +   DD 34 XX
INC (IY+N)    23    3   - V +   FD 34 XX
INC A         4     1   -0V+++  3C
INC B         4     1   -0V+++  04
INC BC        6     1   ------  03
INC C         4     1   -0V+++  0C
INC D         4     1   -0V+++  14
INC DE        6     1   ------  13
INC E         4     1   -0V+++  1C
INC H         4     1   -0V+++  24
INC HL        6     1   ------  23
INC HX              2   -0V+++  DD 24
INC HY              2   -0V+++  FD 24
INC IX        10    2   ------  DD 23
INC IY        10    2   ------  FD 23
INC L         4     1   -0V+++  2C
INC LX              2   -0V+++  DD 2C
INC LY              2   -0V+++  FD 2C
INC SP        6     1   ------  33
IND           16    2   -1  +   ED AA
INDR          21/16 2   -1  1   ED BA
INI           16    2   -1  +   ED A2
INIR          21/16 2   -1  1   ED B2
JP $NN        10    3   ------  C3 XX XX
JP (HL)       4     1   ------  E9
JP (IX)       8     2   ------  DD E9
JP (IY)       8     2   ------  FD E9
JP C,$NN      10    3   ------  DA XX XX
JP M,$NN      10    3   ------  FA XX XX
JP NC,$NN     10    3   ------  D2 XX XX
JP NZ,$NN     10    3   ------  C2 XX XX
JP P,$NN      10    3   ------  F2 XX XX
JP PE,$NN     10    3   ------  EA XX XX
JP PO,$NN     10    3   ------  E2 XX XX
JP Z,$NN      10    3   ------  CA XX XX
JR $N+2       12    2   ------  18 XX
JR C,$N+2     12/7  2   ------  38 XX
JR NC,$N+2    12/7  2   ------  30 XX
JR NZ,$N+2    12/7  2   ------  20 XX
JR Z,$N+2     12/7  2   ------  28 XX
LD (BC),A     7     1   ------  02
LD (DE),A     7     1   ------  12
LD (HL),r     7     1   ------  7r
LD (HL),N     10    2   ------  36 XX
LD (IX+N),r   19    3   ------  DD 7r XX
LD (IX+N),N   19    4   ------  DD 36 XX XX
LD (IY+N),r   19    3   ------  FD 7r XX
LD (IY+N),N   19    4   ------  FD 36 XX XX
LD (NN),A     13    3   ------  32 XX XX
LD (NN),BC    20    4   ------  ED 43 XX XX
LD (NN),DE    20    4   ------  ED 53 XX XX
LD (NN),HL    16    3   ------  22 XX XX
LD (NN),IX    20    4   ------  DD 22 XX XX
LD (NN),IY    20    4   ------  FD 22 XX XX
LD (NN),SP    20    4   ------  ED 73 XX XX
LD A,(BC)     7     1   ------  0A
LD A,(DE)     7     1   ------  1A
LD A,(HL)     7     1   ------  7E
LD A,(IX+N)   19    3   ------  DD 7E XX
LD A,(IY+N)   19    3   ------  FD 7E XX
LD A,(NN)     13    3   ------  3A XX XX
LD A,r        4     1   ------  78+r
LD A,HX             2   ------  DD 7C
LD A,HY             2   ------  FD 7C
LD A,LX             2   ------  DD 7D
LD A,LY             2   ------  FD 7D
LD A,I        9     2   -0+0++  ED 57
LD A,N        7     2   ------  3E XX
LD A,R        9     2   -0+0++  ED 5F
LD B,(HL)     7     1   ------  46
LD B,(IX+N)   19    3   ------  DD 46 XX
LD B,(IY+N)   19    3   ------  FD 46 XX
LD B,HX             2   ------  DD 44
LD B,HY             2   ------  FD 44
LD B,LX             2   ------  DD 45
LD B,LY             2   ------  FD 45
LD B,r        4     1   ------  4r
LD B,N        7     2   ------  06 XX
LD BC,(NN)    20    4   ------  ED 4B XX XX
LD BC,NN      10    3   ------  01 XX XX
LD C,(HL)     7     1   ------  4E
LD C,(IX+N)   19    3   ------  DD 4E XX
LD C,(IY+N)   19    3   ------  FD 4E XX
LD C,HX             2   ------  DD 4C
LD C,HY             2   ------  FD 4C
LD C,LX             2   ------  DD 4D
LD C,LY             2   ------  FD 4D
LD C,r        4     1   ------  48+r
LD C,N        7     2   ------  0E XX
LD D,(HL)     7     1   ------  56
LD D,(IX+N)   19    3   ------  DD 56 XX
LD D,(IY+N)   19    3   ------  FD 56 XX
LD D,HX             2   ------  DD 54
LD D,HY             2   ------  FD 54
LD D,LX             2   ------  DD 55
LD D,LY             2   ------  FD 55
LD D,r        4     1   ------  5r
LD D,N        7     2   ------  16 XX
LD DE,(NN)    20    4   ------  ED 5B XX XX
LD DE,NN      10    3   ------  11 XX XX
LD E,(HL)     7     1   ------  5E
LD E,(IX+N)   19    3   ------  DD 5E XX
LD E,(IY+N)   19    3   ------  FD 5E XX
LD E,HX             2   ------  DD 5C
LD E,HY             2   ------  FD 5C
LD E,LX             2   ------  DD 5D
LD E,LY             2   ------  FD 5D
LD E,r        4     1   ------  58+r
LD E,N        7     2   ------  1E XX
LD H,(HL)     7     1   ------  66
LD H,(IX+N)   19    3   ------  DD 66 XX
LD H,(IY+N)   19    3   ------  FD 66 XX
LD H,r        4     1   ------  6r
LD H,N        7     2   ------  26 XX
LD HL,(NN)    20    3   ------  2A XX XX
LD HL,NN      10    3   ------  21 XX XX
LD HX,r*            2   ------  DD 6r*
LD HX,N             3   ------  DD 26 XX
LD HY,r*            2   ------  FD 6r*
LD HY,N             3   ------  FD 26 XX
LD I,A        9     2   ------  ED 47
LD IX,(NN)    20    4   ------  DD 2A XX XX
LD IX,NN      14    4   ------  DD 21 XX XX
LD IY,(NN)    20    4   ------  FD 2A XX XX
LD IY,NN      14    4   ------  FD 21 XX XX
LD L,(HL)     7     1   ------  6E
LD L,(IX+N)   19    3   ------  DD 6E XX
LD L,(IY+N)   19    3   ------  FD 6E XX
LD L,r        4     1   ------  68+r
LD L,N        7     2   ------  2E XX
LD LX,r*            2   ------  DD 68+r*
LD LX,N             3   ------  DD 2E XX
LD LY,r*            2   ------  FD 68+r*
LD LY,N             3   ------  FD 2E XX
LD R,A        9     2   ------  ED 4F
LD SP,(NN)    20    4   ------  ED 7B XX XX
LD SP,HL      6     1   ------  F9
LD SP,IX      10    2   ------  DD F9
LD SP,IY      10    2   ------  FD F9
LD SP,NN      10    3   ------  31 XX XX
LDD           16    2   -0+0--  ED A8
LDDR          21/16 2   -000--  ED B8
LDI           16    2   -0+0--  ED A0
LDIR          21/16 2   -000--  ED B0
NEG           8     2   +1V+++  ED 44
NOP           4     1   ------  00
OR (HL)       7     1   00P0++  B6
OR (IX+N)     19    3   00P0++  DD B6 XX
OR (IY+N)     19    3   00P0++  FD B6 XX
OR r          4     1   00P0++  Br
OR HX               2   00P0++  DD B4
OR HY               2   00P0++  FD B4
OR LX               2   00P0++  DD B5
OR LY               2   00P0++  FD B5
OR N          7     2   00P0++  F6 XX
OTDR          21/16 2   -1  1   ED BB
OTIR          21/16 2   -1  1   ED B3
OUT (C),A     12    2   ------  ED 79
OUT (C),B     12    2   ------  ED 41
OUT (C),C     12    2   ------  ED 49
OUT (C),D     12    2   ------  ED 51
OUT (C),E     12    2   ------  ED 59
OUT (C),H     12    2   ------  ED 61
OUT (C),L     12    2   ------  ED 69
OUT (C),0     12    2   ------  ED 71
OUT (N),A     11    2   ------  D3 XX
OUTD          16    2   -1  +   ED AB
OUTI          16    2   -1  +   ED A3
POP AF        10    1   ------  F1
POP BC        10    1   ------  C1
POP DE        10    1   ------  D1
POP HL        10    1   ------  E1
POP IX        14    2   ------  DD E1
POP IY        14    2   ------  FD E1
PUSH AF       11    1   ------  F5
PUSH BC       11    1   ------  C5
PUSH DE       11    1   ------  D5
PUSH HL       11    1   ------  E5
PUSH IX       15    2   ------  DD E5
PUSH IY       15    2   ------  FD E5
RES b,(HL)    15    2   ------  CB 86+8*b
RES b,(IX+N)  23    4   ------  DD CB XX 86+8*b
RES b,(IY+N)  23    4   ------  FD CB XX 86+8*b
RES b,r       8     2   ------  CB 8r+8*b
RET           10    1   ------  C9
RET C         11/5  1   ------  D8
RET M         11/5  1   ------  F8
RET NC        11/5  1   ------  D0
RET NZ        11/5  1   ------  C0
RET P         11/5  1   ------  F0
RET PE        11/5  1   ------  E8
RET PO        11/5  1   ------  E0
RET Z         11/5  1   ------  C8
RETI          14    2   ------  ED 4D
RETN          14    2   ------  ED 45
RL (HL)       15    2   +0P0++  CB 16
RL r          8     2   +0P0++  CB 1r
RL (IX+N)     23    4   +0P0++  DD CB XX 16
RL (IY+N)     23    4   +0P0++  FD CB XX 16
RLA           4     1   +0-0--  17
RLC (HL)      15    2   +0P0++  CB 06
RLC (IX+N)    23    4   +0P0++  DD CB XX 06
RLC (IY+N)    23    4   +0P0++  FD CB XX 06
RLC r         8     2   +0P0++  CB 0r
RLCA          4     1   +0-0--  07
RLD           18    2   -0P0++  ED 6F
RR (HL)       15    2   +0P0++  CB 1E
RR r          8     2   +0P0++  CB 18+r
RR (IX+N)     23    4   +0P0++  DD CB XX 1E
RR (IY+N)     23    4   +0P0++  FD CB XX 1E
RRA           4     1   +0-0--  1F
RRC (HL)      15    2   +0P0++  CB 0E
RRC (IX+N)    23    4   +0P0++  DD CB XX 0E
RRC (IY+N)    23    4   +0P0++  FD CB XX 0E
RRC r         8     2   +0P0++  CB 08+r
RRCA          4     1   +0-0--  0F
RRD           18    2   -0P0++  ED 67
RST 0         11    1   ------  C7
RST 8H        11    1   ------  CF
RST 10H       11    1   ------  D7
RST 18H       11    1   ------  DF
RST 20H       11    1   ------  E7
RST 28H       11    1   ------  EF
RST 30H       11    1   ------  F7
RST 38H       11    1   ------  FF
SBC A,(HL)    7     1   +1V+++  9E
SBC A,(IX+N)  19    3   +1V+++  DD 9E XX
SBC A,(IY+N)  19    3   +1V+++  FD 9E XX
SBC A,r       4     1   +1V+++  98+r
SBC HX              2   +1V+++  DD 9C
SBC HY              2   +1V+++  FD 9C
SBC LX              2   +1V+++  DD 9D
SBC LY              2   +1V+++  FD 9D
SBC A,N       7     2   +1V+++  DE XX
SBC HL,BC     15    2   +1V ++  ED 42
SBC HL,DE     15    2   +1V ++  ED 52
SBC HL,HL     15    2   +1V ++  ED 62
SBC HL,SP     15    2   +1V ++  ED 72
SCF           4     1   10-0--  37
SET b,(HL)    15    2   ------  CB C6+8*b
SET b,(IX+N)  23    4   ------  DD CB XX C6+8*b
SET b,(IY+N)  23    4   ------  FD CB XX C6+8*b
SET b,r       8     2   ------  CB Cr+8*b
SLA (HL)      15    2   +0P0++  CB 26
SLA (IX+N)    23    4   +0P0++  DD CB XX 26
SLA (IY+N)    23    4   +0P0++  FD CB XX 26
SLA r         8     2   +0P0++  CB 2r
SLL (HL)      15    2   +0P0++  CB 36
SLL (IX+N)    23    4   +0P0++  DD CB XX 36
SLL (IY+N)    23    4   +0P0++  FD CB XX 36
SLL r         8     2   +0P0++  CB 3r
SRA (HL)      15    2   +0P0++  CB 2E
SRA (IX+N)    23    4   +0P0++  DD CB XX 2E
SRA (IY+N)    23    4   +0P0++  FD CB XX 2E
SRA r         8     2   +0P0++  CB 28+r
SRL (HL)      15    2   +0P0++  CB 3E
SRL (IX+N)    23    4   +0P0++  DD CB XX 3E
SRL (IY+N)    23    4   +0P0++  FD CB XX 3E
SRL r         8     2   +0P0++  CB 38+r
SUB (HL)      7     1   ++V+++  96
SUB (IX+N)    19    3   ++V+++  DD 96 XX
SUB (IY+N)    19    3   ++V+++  FD 96 XX
SUB r         4     1   ++V+++  9r
SUB HX              2   ++V+++  DD 94
SUB HY              2   ++V+++  FD 94
SUB LX              2   ++V+++  DD 95
SUB LY              2   ++V+++  FD 95
SUB N         7     2   ++V+++  D6 XX
XOR (HL)      7     1   00P0++  AE
XOR (IX+N)    19    3   00P0++  DD AE XX
XOR (IY+N)    19    3   00P0++  FD AE XX
XOR r         4     1   00P0++  A8+r
XOR HX              2   00P0++  DD AC
XOR HY              2   00P0++  FD AC
XOR LX              2   00P0++  DD AD
XOR LY              2   00P0++  FD AD
XOR N         7     2   00P0++  EE XX
`

// See explanation of +r in z80.txt
var registerNybble []string = []string{"B", "C", "D", "E", "H", "L", /*"(HL)"*/"", "A"}
var registerStarNybble []string = []string{"B", "C", "D", "E", "HX", "LX", "", "A"}

type instructionMap map[byte]*instruction

type instruction struct {
	// Leaf of tree.
	asm, cycles, flags, opcodes string

	// For XX data byte.
	xx *instruction

	// For extended instructions.
	imap instructionMap
}

func parseByte(s string) byte {
	v, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		panic(err)
	}

	return byte(v)
}

func (cpu *cpu) loadInstructions(instructionList string) {
	cpu.root = &instruction{}

	lines := strings.Split(instructionList, "\n")

	for _, line := range lines {
		cpu.parseInstructionLine(line)
	}
}

func (cpu *cpu) parseInstructionLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	asm := strings.TrimSpace(line[:14])
	cycles := strings.TrimSpace(line[14:20])
	flags := strings.TrimSpace(line[24:32])
	opcodes := strings.Split(strings.TrimSpace(line[32:]), " ")

	// Remove "$" from asm, not sure it means anything.
	asm = strings.Replace(asm, "$", "", -1)

	cpu.root.addInstruction(asm, cycles, flags, opcodes)
}

func (inst *instruction) addInstruction(asm, cycles, flags string, opcodes []string) {
	// See if it's a terminal node.
	if len(opcodes) == 0 {
		if inst.asm != "" {
			panic(fmt.Sprintf("Already have %s in leaf, trying to write %s", inst.asm, asm))
		}

		// Leaf of tree.
		inst.asm = asm
		inst.cycles = cycles
		inst.flags = flags
	} else {
		opcodeStr := opcodes[0]

		// See if it's user data.
		if opcodeStr == "XX" {
			inst.xx = &instruction{}
			inst.xx.addInstruction(asm, cycles, flags, opcodes[1:])
		} else {
			// Expand "8r" abbreviation to "80+r"
			if strings.Contains(opcodeStr, "r") && !strings.Contains(opcodeStr, "+r") {
				opcodeStr = strings.Replace(opcodeStr, "r", "0+r", 1)
			}

			opcode := parseByte(opcodeStr[:2])

			if strings.HasSuffix(opcodeStr, "+8*b") {
				// Expand "+8*b" to each bit value.
				opcodeRest := opcodeStr[2 : len(opcodeStr)-4] // May be empty.

				// Replace "b" with bit value.
				opcodes[0] = fmt.Sprintf("%02X%s", opcode+0, opcodeRest)
				inst.addInstruction(strings.Replace(asm, "b", "0", -1), cycles, flags, opcodes)
				opcodes[0] = fmt.Sprintf("%02X%s", opcode+8, opcodeRest)
				inst.addInstruction(strings.Replace(asm, "b", "1", -1), cycles, flags, opcodes)
			} else if strings.HasSuffix(opcodeStr, "+r") || strings.HasSuffix(opcodeStr, "+r*") {
				// Expand registers.
				for n := byte(0); n < 8; n++ {
					// Replace "r" with register in asm.
					var r string
					if strings.HasSuffix(opcodeStr, "*") {
						r = registerStarNybble[n]
					} else {
						r = registerNybble[n]
					}

					if r != "" {
						opcodes[0] = fmt.Sprintf("%02X", opcode+n)
						inst.addInstruction(strings.Replace(asm, "r", r, -1),
							cycles, flags, opcodes)
					}
				}
			} else {
				if inst.imap == nil {
					inst.imap = make(instructionMap)
				}

				// Get or create node in tree.
				subInst, ok := inst.imap[opcode]
				if !ok {
					subInst = &instruction{}
					inst.imap[opcode] = subInst
				}

				subInst.addInstruction(asm, cycles, flags, opcodes[1:])
			}
		}
	}

	var hasiMap, hasXx, isLeaf int
	if inst.imap != nil {
		hasiMap = 1
	}
	if inst.xx != nil {
		hasXx = 1
	}
	if inst.asm != "" {
		isLeaf = 1
	}

	if hasiMap + hasXx + isLeaf != 1 {
		panic(fmt.Sprintf("Instruction %s has wrong number of children (%d, %d, %d)",
			asm, hasiMap, hasXx, isLeaf))
	}
}

func (cpu *cpu) step2() {
	// Look up the instruction in the tree.
	instPc := cpu.pc
	inst, byteData, wordData := cpu.lookUpInst()

	fmt.Printf("%04X ", instPc)
	for pc := instPc; pc < instPc + 4; pc++ {
		if pc < cpu.pc {
			fmt.Printf("%02X ", cpu.memory[pc])
		} else {
			fmt.Print("   ")
		}
	}
	fmt.Printf("%-15s ", inst.asm)

	fields := strings.Split(inst.asm, " ")
	var subfields []string
	switch len(fields) {
	case 1:
		subfields = nil
	case 2:
		subfields = strings.Split(fields[1], ",")
	default:
		panic(fmt.Sprintf("Unknown number of fields %d", len(fields)))
	}

	switch fields[0] {
	case "AND", "XOR", "OR":
		value := cpu.getByteValue(subfields[0], byteData, wordData)
		before := cpu.a
		var symbol string
		switch fields[0] {
		case "AND":
			cpu.a &= value
			symbol = "&"
		case "XOR":
			cpu.a ^= value
			symbol = "^"
		case "OR":
			cpu.a |= value
			symbol = "|"
		}
		cpu.f.updateFromByte(cpu.a, inst.flags)
		fmt.Printf("%02X %s %02X = %02X", before, symbol, value, cpu.a)
	case "CP":
		value := cpu.getByteValue(subfields[0], byteData, wordData)
		diff := int(cpu.a) - int(value)
		cpu.f.setS(diff < 0)
		cpu.f.setZ(diff == 0)
		cpu.f.setC(diff < 0) // Borrow.
		cpu.f.setN(true)
		fmt.Printf("%02X - %02X", cpu.a, value)
	case "DEC":
		if isWordOperand(subfields[0]) {
			value := cpu.getWordValue(subfields[0], byteData, wordData) - 1
			fmt.Printf("%04X - 1 = %04X", value+1, value)
			cpu.setWord(subfields[0], value, byteData, wordData)
			cpu.f.updateFromWord(value, inst.flags)
		} else {
			value := cpu.getByteValue(subfields[0], byteData, wordData) - 1
			fmt.Printf("%02X + 1 = %02X", value+1, value)
			cpu.setByte(subfields[0], value, byteData, wordData)
			cpu.f.updateFromByte(value, inst.flags)
		}
	case "DI":
		cpu.iff = false
	case "DJNZ":
		rel := signExtend(byteData)
		cpu.bc.setH(cpu.bc.h() - 1)
		if cpu.bc.h() != 0 {
			cpu.pc += rel
			fmt.Printf("%04X (%d), b = %02X", cpu.pc, int16(rel), cpu.bc.h())
		} else {
			fmt.Print("jump skipped")
		}
	case "INC":
		if isWordOperand(subfields[0]) {
			value := cpu.getWordValue(subfields[0], byteData, wordData) + 1
			fmt.Printf("%04X + 1 = %04X", value-1, value)
			cpu.setWord(subfields[0], value, byteData, wordData)
			cpu.f.updateFromWord(value, inst.flags)
		} else {
			value := cpu.getByteValue(subfields[0], byteData, wordData) + 1
			fmt.Printf("%02X + 1 = %02X", value-1, value)
			cpu.setByte(subfields[0], value, byteData, wordData)
			cpu.f.updateFromByte(value, inst.flags)
		}
	case "JP", "CALL":
		addr := cpu.getWordValue(subfields[len(subfields)-1], byteData, wordData)
		if len(subfields) == 1 || cpu.conditionSatisfied(subfields[0]) {
			if fields[0] == "CALL" {
				cpu.pushWord(cpu.pc)
			}
			cpu.pc = addr
			fmt.Printf("%04X", addr)
		} else {
			fmt.Print("jump skipped")
		}
	case "JR":
		if subfields[len(subfields)-1] != "N+2" {
			panic("Can only handle relative jumps to N, not " + subfields[len(subfields)-1])
		}
		// Relative jump is signed.
		rel := signExtend(byteData)
		if len(subfields) == 1 || cpu.conditionSatisfied(subfields[0]) {
			cpu.pc += rel
			fmt.Printf("%04X (%d)", cpu.pc, int16(rel))
		} else {
			fmt.Print("jump skipped")
		}
	case "LD":
		if isWordOperand(subfields[0]) || isWordOperand(subfields[1]) {
			value := cpu.getWordValue(subfields[1], byteData, wordData)
			cpu.setWord(subfields[0], value, byteData, wordData)
			fmt.Printf("%04X", value)
		} else {
			value := cpu.getByteValue(subfields[1], byteData, wordData)
			cpu.setByte(subfields[0], value, byteData, wordData)
			fmt.Printf("%02X", value)
		}
	case "LDIR":
		// Not sure if this should be while or do while.
		fmt.Printf("copying %04X bytes from %04X to %04X", cpu.bc, cpu.hl, cpu.de)
		for cpu.bc != 0xFFFF {
			cpu.writeMem(cpu.de, cpu.readMem(cpu.hl))
			cpu.hl++
			cpu.de++
			cpu.bc--
		}
	case "NOP":
		// Nothing to do!
		panic("Probably a bug")
	case "OUT":
		var port byte
		value := cpu.getByteValue(subfields[1], byteData, wordData)
		switch subfields[0] {
		case "(C)":
			port = cpu.bc.l()
		case "(N)":
			port = byteData
		default:
			panic("Unknown OUT destination " + subfields[0])
		}
		portDescription, ok := ports[port]
		if !ok {
			panic(fmt.Sprintf("Unknown port %02X", port))
		}
		fmt.Printf("%02X (%s) <- %02X", port, portDescription, value)
	case "POP":
		value := cpu.popWord()
		cpu.setWord(subfields[0], value, byteData, wordData)
		fmt.Printf("%04X", value)
	case "PUSH":
		value := cpu.getWordValue(subfields[0], byteData, wordData)
		cpu.pushWord(value)
		fmt.Printf("%04X", value)
	case "RET":
		if subfields == nil || cpu.conditionSatisfied(subfields[0]) {
			cpu.pc = cpu.popWord()
			fmt.Printf("%04X", cpu.pc)
		} else {
			fmt.Print("return skipped")
		}
	default:
		panic(fmt.Sprintf("Don't know how to handle %s (at %04X)",
			inst.asm, instPc))
	}

	fmt.Println()
}

func (cpu *cpu) getByteValue(ref string, byteData byte, wordData word) byte {
	switch ref {
	case "A":
		return cpu.a
	case "B":
		return cpu.bc.h()
	case "C":
		return cpu.bc.l()
	case "D":
		return cpu.de.h()
	case "E":
		return cpu.de.l()
	case "H":
		return cpu.hl.h()
	case "L":
		return cpu.hl.l()
	case "(BC)":
		return cpu.readMem(cpu.bc)
	case "(DE)":
		return cpu.readMem(cpu.de)
	case "(HL)":
		return cpu.readMem(cpu.hl)
	case "(IX+N)":
		return cpu.readMem(cpu.ix + signExtend(byteData))
	case "(IY+N)":
		return cpu.readMem(cpu.iy + signExtend(byteData))
	case "N":
		return byteData
	case "(NN)":
		return cpu.readMem(wordData)
	}

	panic("We don't yet handle addressing mode " + ref)
}

func (cpu *cpu) getWordValue(ref string, byteData byte, wordData word) word {
	switch ref {
	case "BC":
		return cpu.bc
	case "DE":
		return cpu.de
	case "HL":
		return cpu.hl
	case "IX":
		return cpu.ix
	case "IY":
		return cpu.iy
	case "NN":
		return wordData
	case "(NN)":
		return cpu.readMemWord(wordData)
	case "(HL)":
		return cpu.readMemWord(cpu.hl)
	}

	panic("We don't yet handle addressing mode " + ref)
}

func (cpu *cpu) setByte(ref string, value byte, byteData byte, wordData word) {
	switch ref {
	case "A":
		cpu.a = value
	case "B":
		cpu.bc.setH(value)
	case "C":
		cpu.bc.setL(value)
	case "D":
		cpu.de.setH(value)
	case "E":
		cpu.de.setL(value)
	case "H":
		cpu.hl.setH(value)
	case "L":
		cpu.hl.setL(value)
	case "(BC)":
		cpu.writeMem(cpu.bc, value)
	case "(DE)":
		cpu.writeMem(cpu.de, value)
	case "(HL)":
		cpu.writeMem(cpu.hl, value)
	case "(NN)":
		cpu.writeMem(wordData, value)
	default:
		panic("Can't handle destination of " + ref)
	}
}

func (cpu *cpu) setWord(ref string, value word, byteData byte, wordData word) {
	switch ref {
	case "BC":
		cpu.bc = value
	case "DE":
		cpu.de = value
	case "HL":
		cpu.hl = value
	case "SP":
		cpu.sp = value
	case "IX":
		cpu.ix = value
	case "IY":
		cpu.iy = value
	case "(NN)":
		addr := wordData
		cpu.writeMem(addr, value.l())
		cpu.writeMem(addr+1, value.h())
	default:
		panic("Can't handle destination of " + ref)
	}
}

func (cpu *cpu) lookUpInst() (inst *instruction, byteData byte, wordData word) {
	haveByteData := false

	inst = cpu.root

	for {
		// Terminal node.
		if inst.asm != "" {
			return
		}

		opcode := cpu.fetchByte()

		// User data.
		if inst.xx != nil {
			if haveByteData {
				wordData.setH(opcode)
				wordData.setL(byteData)
			} else {
				byteData = opcode
				haveByteData = true
			}

			inst = inst.xx
		} else {
			// Keep fetching as long as it's an extended instruction.
			var ok bool
			inst, ok = inst.imap[opcode]
			if !ok {
				panic(fmt.Sprintf("Don't know how to handle opcode %02X", opcode))
			}
		}
	}

	panic("Can't get here")
}

func (cpu *cpu) conditionSatisfied(cond string) bool {
	switch cond {
	case "C":
		return cpu.f.c()
	case "NC":
		return !cpu.f.c()
	case "Z":
		return cpu.f.z()
	case "NZ":
		return !cpu.f.z()
	case "P":
		return cpu.f.s()
	case "M": // Negative.
		return !cpu.f.s()
	case "PE":
		return cpu.f.pv()
	case "PO":
		return !cpu.f.pv()
	}

	panic("Unknown condition " + cond)
}

func isWordOperand(op string) bool {
	switch op {
	case "BC", "DE", "HL", "NN":
		return true
	}

	return false
}

func signExtend(b byte) word {
	return word(int8(b))
}
