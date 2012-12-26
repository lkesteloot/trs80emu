// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Constants for each instruction type, so we can dispatch faster.
const (
	// 56 of 68 implemented.
	instAdc = iota
	instAdd
	instAnd
	instBit
	instCall
	instCcf
	instCp
	instCpd
	instCpdr
	instCpi
	instCpir
	instCpl
	instDaa
	instDec
	instDi
	instDjnz
	instEi
	instEx
	instExx
	instHalt // Not yet implemented.
	instIm
	instIn
	instInc
	instInd  // Not yet implemented.
	instIndr // Not yet implemented.
	instIni
	instInir // Not yet implemented.
	instJp
	instJr
	instLd
	instLdd
	instLddr
	instLdi // Not yet implemented.
	instLdir
	instNeg
	instNop
	instOr
	instOtdr // Not yet implemented.
	instOtir // Not yet implemented.
	instOut
	instOutd // Not yet implemented.
	instOuti // Not yet implemented.
	instPop
	instPush
	instRes
	instRet
	instReti // Not yet implemented.
	instRetn // Not yet implemented.
	instRl
	instRla
	instRlc
	instRlca
	instRld
	instRr
	instRra
	instRrc
	instRrca
	instRrd
	instRst
	instSbc
	instScf
	instSet
	instSla
	instSll // Not yet implemented.
	instSra
	instSrl
	instSub
	instXor
)

// Look-up table from instruction string to type.
var instToInstInt = map[string]int{
	"ADC":  instAdc,
	"ADD":  instAdd,
	"AND":  instAnd,
	"BIT":  instBit,
	"CALL": instCall,
	"CCF":  instCcf,
	"CP":   instCp,
	"CPD":  instCpd,
	"CPDR": instCpdr,
	"CPI":  instCpi,
	"CPIR": instCpir,
	"CPL":  instCpl,
	"DAA":  instDaa,
	"DEC":  instDec,
	"DI":   instDi,
	"DJNZ": instDjnz,
	"EI":   instEi,
	"EX":   instEx,
	"EXX":  instExx,
	"HALT": instHalt,
	"IM":   instIm,
	"IN":   instIn,
	"INC":  instInc,
	"IND":  instInd,
	"INDR": instIndr,
	"INI":  instIni,
	"INIR": instInir,
	"JP":   instJp,
	"JR":   instJr,
	"LD":   instLd,
	"LDD":  instLdd,
	"LDDR": instLddr,
	"LDI":  instLdi,
	"LDIR": instLdir,
	"NEG":  instNeg,
	"NOP":  instNop,
	"OR":   instOr,
	"OTDR": instOtdr,
	"OTIR": instOtir,
	"OUT":  instOut,
	"OUTD": instOutd,
	"OUTI": instOuti,
	"POP":  instPop,
	"PUSH": instPush,
	"RES":  instRes,
	"RET":  instRet,
	"RETI": instReti,
	"RETN": instRetn,
	"RL":   instRl,
	"RLA":  instRla,
	"RLC":  instRlc,
	"RLCA": instRlca,
	"RLD":  instRld,
	"RR":   instRr,
	"RRA":  instRra,
	"RRC":  instRrc,
	"RRCA": instRrca,
	"RRD":  instRrd,
	"RST":  instRst,
	"SBC":  instSbc,
	"SCF":  instScf,
	"SET":  instSet,
	"SLA":  instSla,
	"SLL":  instSll,
	"SRA":  instSra,
	"SRL":  instSrl,
	"SUB":  instSub,
	"XOR":  instXor,
}

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
# The real instructions here are "JP (HL)" etc, but this implies that it uses
# the value pointed to by HL, so I removed the parentheses.
JP HL         4     1   ------  E9
JP IX         8     2   ------  DD E9
JP IY         8     2   ------  FD E9
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
# The real instructions are "RST 8H" and "RST 28H" and so on, but removing the "H"
# and making the addresses two characters makes parsing easier.
RST 00        11    1   ------  C7
RST 08        11    1   ------  CF
RST 10        11    1   ------  D7
RST 18        11    1   ------  DF
RST 20        11    1   ------  E7
RST 28        11    1   ------  EF
RST 30        11    1   ------  F7
RST 38        11    1   ------  FF
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
var registerNybble []string = []string{"B", "C", "D", "E", "H", "L", "", "A"}
var registerStarNybble []string = []string{"B", "C", "D", "E", "HX", "LX", "", "A"}

// Map from instruction byte to an instruction record.
type instructionMap [256]*instruction

// Node in the instruction tree. Can be a leaf (instruction itself), a data
// byte (XX), or a literal byte for extended instructions.
type instruction struct {
	// Leaf of tree.
	asm, opcodes        string
	cycles, jumpPenalty uint64
	fields              []string
	subfields           []string
	instInt             int

	// For XX data byte.
	xx *instruction

	// For extended instructions.
	imap *instructionMap
}

// Convert a two-digit hex number to a byte.
func parseByte(s string) byte {
	v, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		panic(err)
	}

	return byte(v)
}

// Fills the instruction tree from the string of all instructions.
func (inst *instruction) loadInstructions(instructionList string) {
	lines := strings.Split(instructionList, "\n")

	for _, line := range lines {
		inst.parseInstructionLine(line)
	}

	if dumpInstructionSet {
		inst.dump("")
	}
}

// Recursively dump the whole instruction tree.
func (inst *instruction) dump(path string) {
	if inst.asm != "" {
		log.Printf("%-12s %s", path, inst.asm)
	} else if inst.xx != nil {
		inst.xx.dump(path + "XX ")
	} else {
		for b, child := range inst.imap {
			if child != nil {
				child.dump(path + fmt.Sprintf("%02X ", b))
			}
		}
	}
}

// Parses a line from the instruction string and adds it to the tree rooted at inst.
func (inst *instruction) parseInstructionLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	asm := strings.TrimSpace(line[:14])
	cycles := strings.TrimSpace(line[14:20])
	opcodes := strings.Split(strings.TrimSpace(line[32:]), " ")

	// Remove "$" from asm, not sure it means anything.
	asm = strings.Replace(asm, "$", "", -1)

	inst.addInstruction(asm, cycles, opcodes)
}

// Recursively add opcodes to the instruction tree rooted at inst.
func (inst *instruction) addInstruction(asm, cycles string, opcodes []string) {
	// See if it's a terminal node.
	if len(opcodes) == 0 {
		if inst.asm != "" {
			panic(fmt.Sprintf("Already have %s in leaf, trying to write %s", inst.asm, asm))
		}

		// Leaf of tree.
		inst.asm = asm

		// Split cycles into the "17/10" fields. The first (higher) number, if present,
		// is the number of cycles if the jump is taken.
		cyclesFields := strings.Split(cycles, "/")
		inst.cycles, _ = strconv.ParseUint(cyclesFields[len(cyclesFields)-1], 10, 64)
		if len(cyclesFields) == 2 {
			cyclesWithJump, _ := strconv.ParseUint(cyclesFields[0], 10, 64)
			inst.jumpPenalty = cyclesWithJump - inst.cycles
		}

		// Pre-process asm
		inst.fields = strings.Split(inst.asm, " ")
		switch len(inst.fields) {
		case 1:
			inst.subfields = nil
		case 2:
			inst.subfields = strings.Split(inst.fields[1], ",")
		default:
			panic(fmt.Sprintf("Unknown number of fields %d", len(inst.fields)))
		}

		// Map to an integer constant to make dispatch 40% faster.
		inst.instInt = instToInstInt[inst.fields[0]]
	} else {
		// Create internal node of tree.
		opcodeStr := opcodes[0]

		// See if it's user data.
		if opcodeStr == "XX" {
			if inst.xx == nil {
				inst.xx = &instruction{}
			}
			inst.xx.addInstruction(asm, cycles, opcodes[1:])
		} else {
			// Expand "8r" abbreviation to "80+r"
			if strings.Contains(opcodeStr, "r") && !strings.Contains(opcodeStr, "+r") {
				opcodeStr = strings.Replace(opcodeStr, "r", "0+r", 1)
			}

			opcode := parseByte(opcodeStr[:2])

			if strings.HasSuffix(opcodeStr, "+8*b") {
				// Expand "+8*b" to each bit value (0 to 7).
				opcodeRest := opcodeStr[2 : len(opcodeStr)-4] // May be empty.

				// Replace "b" with bit value.
				for b := byte(0); b < 8; b++ {
					opcodes[0] = fmt.Sprintf("%02X%s", opcode+8*b, opcodeRest)
					inst.addInstruction(strings.Replace(
						asm, "b", fmt.Sprintf("%d", b), -1), cycles, opcodes)
				}
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
						inst.addInstruction(strings.Replace(asm, "r", r, -1), cycles, opcodes)
					}
				}
			} else {
				if inst.imap == nil {
					inst.imap = new(instructionMap)
				}

				// Get or create node in tree.
				subInst := inst.imap[opcode]
				if subInst == nil {
					subInst = &instruction{}
					inst.imap[opcode] = subInst
				}

				// Recurse with rest of instruction.
				subInst.addInstruction(asm, cycles, opcodes[1:])
			}
		}
	}

	// Sanity check.
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

	if hasiMap+hasXx+isLeaf != 1 {
		panic(fmt.Sprintf("Instruction %s has wrong number of children (%d, %d, %d)",
			asm, hasiMap, hasXx, isLeaf))
	}
}

func (vm *vm) lookUpInst(pc *word) (inst *instruction, byteData byte, wordData word) {
	haveByteData := false
	inst = vm.cpu.root

	for {
		// Terminal node.
		if inst.asm != "" {
			return
		}

		opcode := vm.readMem(*pc)
		*pc++

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
			inst = inst.imap[opcode]
			if inst == nil {
				// Unknown instruction.
				return nil, 0, 0
			}
		}
	}

	panic("Can't get here")
}
