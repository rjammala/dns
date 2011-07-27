%{
// Copyright (c) 2011 CZ.NIC z.s.p.o. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// blame: jnml, labs.nic.cz


// WARNING: If this file is scanner.go then DO NOT EDIT.
// scanner.go is generated by golex from scanner.l (see the Makefile).


package zone


import(
	"bufio"
	"bytes"
	"github.com/cznic/dns/rr"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"unicode"
)


type lexStackItem struct{
	rrHandler    func(*rr.RR) bool
	errHandler   func(e string) bool
	startCond    int
	buf          []byte
	last         byte
	peek         byte
	line         int
	column       int
	name         string
	src		     *bufio.Reader
	prev         *lexStackItem
	inParen      bool
}


type lex struct {
	lexStackItem
}


func (l *lex) include(name string, source *bufio.Reader) {
	push := &lexStackItem{}
	*push = l.lexStackItem
	l.buf = nil
	l.line = 1
	l.column = 0
	l.name = name
	l.src = source
	l.prev = push
	l.getc(0)	
}


func (l *lex) getc(c byte) byte {
	if c != 0 {
		l.buf = append(l.buf, c)
	}
	l.last = l.peek
	if b, err := l.src.ReadByte(); err == nil {
		if b == '\n' {
			l.line++
			l.column = 0
		} else {
			l.column++
		}
		l.peek = b
		return b
	} else {
		if err == os.EOF {
			l.peek = 0
			return 0
		}
		panic(err)
	}

	panic("unreachable")
}


func newLex(name string, source *bufio.Reader, errHandler func(e string) bool, rrHandler func(*rr.RR) bool) (l *lex) {
	l = &lex{}
	l.errHandler = errHandler
	l.rrHandler = rrHandler
	l.line = 1
	l.name = name
	l.src = source
	l.getc(0)
	return
}


func (l *lex) Error(e string) {
	e = fmt.Sprintf("%s:%d:%d - %s", l.name, l.line, l.column, e)
	if l.errHandler != nil {
		if l.errHandler(e) {
			return
		}
	}
	panic(os.NewError(e))
}


func (l *lex) begin(sc int) {
	l.startCond = sc
}


const (
	sc_INITIAL = iota
	sc_RRHEAD
	sc_DOMAIN
	sc_NUM
	sc_IPV4
	sc_IPV6
	sc_BASE32EXT
	sc_BASE64
	sc_TYPE
	sc_HEX
	sc_QSTR
)


func (l *lex) Lex(lval *yySymType) (ret int) {
	var err os.Error

	/* debug helper
	defer func(sc0 int) {
		var s string
		if ret > 255 {
			if x := ret - 0xE002; x < len(yyToknames) {
				s = yyToknames[x]
			}
		}
		fmt.Printf("*lex.Lex() %s:%d:%d - %d %q %s buf %q, sc %d->%d\n", l.name, l.line, l.column, ret, string(ret), s, l.buf, sc0, l.startCond)
	}(l.startCond) //*/

	c := l.peek
	ret = -1
%}

%yyt l.startCond
%yyb l.last == '\n' || l.last == 0
%yyc c
%yyn c = l.getc(c)

ip_address     {ipv6}|{ipv4}
byte           {digit}{digit}?{digit}?
digit          [0-9]
hex            [0-9a-fA-F]
hex4           {hex}{hex}?{hex}?{hex}?
hexpart        {hexseq}|{hexseq}"::"{hexseq}?|"::"{hexseq}?
hexseq         {hex4}(":"{hex4})*
ipv4           {byte}"."{byte}"."{byte}"."{byte}
ipv6           {hexpart}

domain-name    {label}("."{label})*\.?|\.
let-or-digit   [a-zA-Z0-9]
label          {let-or-digit}(({let-or-digit}|"-")*{let-or-digit})?

%x rrHead domain num ipv4 ipv6 base32ext base64 type hex

%%
	if ret >= 0 {
		lval.str = string(l.buf)
		return
	}

	l.buf = l.buf[:0]

^[ \t]+$       |
<*>[ \t]+$     |
<*>[ \t]*;.*   // ignored

^$INCLUDE

^$ORIGIN

^$TTL
	ret = tDLR_TTL

<*>\n|\r
	if !l.inParen {
		ret = '\n'
	}

<*>\(
	if l.inParen {
		l.Error("nested parenthesis")
	}
	l.inParen = true

<*>\)
	if !l.inParen {
		l.Error(`enexpected ")"`)
	}
	l.inParen = false
	ret = '\n'

^{domain-name}
	ret = tDOMAIN_NAME

^[ \t]+
	ret = tBLANK_START

<*>\"(\\.|[^\\"\n\r])*\"
	lval.str = string(bytes.Replace(l.buf[1:len(l.buf)-1], []byte(`\"`), []byte(`"`), -1))
	return tQSTR

<domain>{domain-name}
	ret = tDOMAIN_NAME

<*>[ \t]+

	/* RR TTL */
<rrHead,num>[0-9]+
	lval.str = string(l.buf)
	lval.u64, err = strconv.Atoui64(lval.str)
	if err != nil {
		panic(fmt.Errorf("invalid number %q", l.buf))
	}
	ret = tDECADIC	


	/* RR class */
<rrHead>IN
	lval.class = rr.CLASS_IN
	ret = tCLASS
<rrHead>CH
	lval.class = rr.CLASS_CH
	ret = tCLASS
<rrHead>HS
	lval.class = rr.CLASS_HS
	ret = tCLASS


	/* RR types */
<rrHead,type>A          ret = tA
<rrHead,type>AAAA       ret = tAAAA
<rrHead,type>A6         ret = tA6
<rrHead,type>AFSDB      ret = tAFSDB
<rrHead,type>APL        ret = tAPL
<rrHead,type>CERT       ret = tCERT
<rrHead,type>CNAME      ret = tCNAME
<rrHead,type>DHCID      ret = tDHCID
<rrHead,type>DNAME      ret = tDNAME
<rrHead,type>DNSKEY     ret = tDNSKEY
<rrHead,type>DS         ret = tDS
<rrHead,type>GPOS       ret = tGPOS
<rrHead,type>HINFO      ret = tHINFO
<rrHead,type>IPSECKEY   ret = tIPSECKEY
<rrHead,type>ISDN       ret = tISDN
<rrHead,type>KEY        ret = tKEY
<rrHead,type>KX         ret = tKX
<rrHead,type>LOC        ret = tLOC
<rrHead,type>MX         ret = tMX
<rrHead,type>NAPTR      ret = tNAPTR
<rrHead,type>NSAP       ret = tNSAP
<rrHead,type>NS         ret = tNS
<rrHead,type>NSEC       ret = tNSEC
<rrHead,type>NSEC3      ret = tNSEC3
<rrHead,type>NSEC3PARAM ret = tNSEC3PARAM
<rrHead,type>NXT        ret = tNXT
<rrHead,type>PTR        ret = tPTR
<rrHead,type>PX         ret = tPX
<rrHead,type>RP         ret = tRP
<rrHead,type>RRSIG      ret = tRRSIG
<rrHead,type>RT         ret = tRT
<rrHead,type>SIG        ret = tSIG
<rrHead,type>SOA        ret = tSOA
<rrHead,type>SPF        ret = tSPF
<rrHead,type>SRV        ret = tSRV
<rrHead,type>SSHFP      ret = tSSHFP
<rrHead,type>TXT        ret = tTXT
<rrHead,type>WKS        ret = tWKS
<rrHead,type>X25        ret = tX25

<ipv4>{ipv4}
	ip := net.ParseIP(string(l.buf))
	if ip == nil {
		l.Error(fmt.Sprintf("invalid IP %q", l.buf))
	}
	lval.ip = ip
	return tIPV4

<ipv6>{ipv6}
	ip := net.ParseIP(string(l.buf))
	if ip == nil {
		l.Error(fmt.Sprintf("invalid IP %q", l.buf))
	}
	lval.ip = ip
	return tIPV6

<base32ext>[0-9A-V=]+
	ret = tBASE32EXT

<base64>[A-Za-z0-9+/=]+
	ret = tBASE64

<hex>[0-9a-fA-F]+
	var err os.Error
	lval.data, err = hex.DecodeString(string(l.buf))
	if err != nil {
		l.Error("expected hex string")
	}
	ret = tHEX

<hex>-
	lval.data = nil
	ret = tHEX

%%
	// no pattern recognized
	if len(l.buf) == 0 {
		ret = int(c)
		l.getc(0)
		return
	}

	return unicode.ReplacementChar
}