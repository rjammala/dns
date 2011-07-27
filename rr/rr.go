// Copyright (c) 2011 CZ.NIC z.s.p.o. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// blame: jnml, labs.nic.cz

// Package rr supports DNS resource records (RFC 1035 chapter 3.2).
package rr

import (
	"bytes"
	"github.com/cznic/dns"
	"github.com/cznic/strutil"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const asserts = false

func init() {
	if asserts {
		println("WARNING: cznic/dns/rr - assertions enabled")
	}
}

const timeLayout = "20060201150405"

// A holds the zone A RData
type A struct {
	Address net.IP // A 32 bit Internet address.
}

// Implementation of dns.Wirer
func (d *A) Encode(b *dns.Wirebuf) {
	ip4(d.Address).Encode(b)
}

// Implementation of dns.Wirer
func (d *A) Decode(b []byte, pos *int) (err os.Error) {
	return (*ip4)(&d.Address).Decode(b, pos)
}

func (d *A) String() string {
	return d.Address.String()
}

// AAAA holds the zone AAAA RData
type AAAA struct {
	Address net.IP // A 128 bit Internet address.
}

// Implementation of dns.Wirer
func (d *AAAA) Encode(b *dns.Wirebuf) {
	ip6(d.Address).Encode(b)
}

// Implementation of dns.Wirer
func (d *AAAA) Decode(b []byte, pos *int) (err os.Error) {
	return (*ip6)(&d.Address).Decode(b, pos)
}

func (d *AAAA) String() string {
	return d.Address.String()
}

// CNAME holds the zone CNAME RData
type CNAME struct {
	Name string
}

// Implementation of dns.Wirer
func (c CNAME) Encode(b *dns.Wirebuf) {
	(dns.DomainName)(c.Name).Encode(b)
}

// Implementation of dns.Wirer
func (c *CNAME) Decode(b []byte, pos *int) (err os.Error) {
	err = (*dns.DomainName)(&c.Name).Decode(b, pos)
	return
}

func (c CNAME) String() string {
	return c.Name
}

// DNSSEC Algorithm Types
// 
// The DNSKEY, RRSIG, and DS RRs use an 8-bit number to identify the
// security algorithm being used.  These values are stored in the
// "Algorithm number" field in the resource record RDATA.
// 
// Some algorithms are usable only for zone signing (DNSSEC), some only
// for transaction security mechanisms (SIG(0) and TSIG), and some for
// both.  Those usable for zone signing may appear in DNSKEY, RRSIG, and
// DS RRs.  Those usable for transaction security would be present in
// SIG(0) and KEY RRs, as described in [RFC2931].
// 
//	                             Zone
//	Value Algorithm [Mnemonic]  Signing  References   Status
//	----- -------------------- --------- ----------  ---------
//	  0   reserved
//	  1   RSA/MD5 [RSAMD5]         n      [RFC2537]  NOT RECOMMENDED
//	  2   Diffie-Hellman [DH]      n      [RFC2539]   -
//	  3   DSA/SHA-1 [DSA]          y      [RFC2536]  OPTIONAL
//	  4   Elliptic Curve [ECC]              TBA       -
//	  5   RSA/SHA-1 [RSASHA1]      y      [RFC3110]  MANDATORY
//	252   Indirect [INDIRECT]      n                  -
//	253   Private [PRIVATEDNS]     y      see below  OPTIONAL
//	254   Private [PRIVATEOID]     y      see below  OPTIONAL
//	255   reserved
//	
//	6 - 251  Available for assignment by IETF Standards Action.
type AlgorithmType byte

// AlgorithmType values
const (
	AlgorithmReserved0 AlgorithmType = iota
	AlgorithmRSA_MD5
	AlgorithmDiffie_Hellman
	AlgorithmDSA_SHA1
	AlgorithmElliptic
	AlgorithmRSA_SHA1
	AlgorithmIndirect     AlgorithmType = 252
	AlgorithmPrivateDNS   AlgorithmType = 253
	AlgorithmPrivateOID   AlgorithmType = 254
	AlgorithmReserved1255 AlgorithmType = 255
)

// Class is a RR CLASS
type Class uint16

// Class values
const (
	CLASS_NONE Class = iota
	CLASS_IN         // the Internet
	CLASS_CS         // the CSNET class (Obsolete - used only for examples in some obsolete RFCs)
	CLASS_CH         // the CHAOS class
	CLASS_HS         // Hesiod
)

var classStr = map[Class]string{
	CLASS_NONE: "",
	CLASS_IN:   "IN",
	CLASS_CS:   "CS",
	CLASS_CH:   "CH",
	CLASS_HS:   "HS",
}

func (n Class) String() (s string) {
	var ok bool
	if s, ok = classStr[n]; !ok {
		panic(fmt.Errorf("unexpected Class %d", uint16(n)))
	}
	return
}

// Implementation of dns.Wirer
func (n Class) Encode(b *dns.Wirebuf) {
	dns.Octets2(n).Encode(b)
}

// Implementation of dns.Wirer
func (n *Class) Decode(b []byte, pos *int) (err os.Error) {
	return (*dns.Octets2)(n).Decode(b, pos)
}

// DNSKEY holds the DNS key RData // RFC 4034
type DNSKEY struct {
	// Bit 7 of the Flags field is the Zone Key flag.  If bit 7 has value 1,
	// then the DNSKEY record holds a DNS zone key, and the DNSKEY RR's
	// owner name MUST be the name of a zone.  If bit 7 has value 0, then
	// the DNSKEY record holds some other type of DNS public key and MUST
	// NOT be used to verify RRSIGs that cover RRsets.
	// 
	// Bit 15 of the Flags field is the Secure Entry Point flag, described
	// in [RFC3757].  If bit 15 has value 1, then the DNSKEY record holds a
	// key intended for use as a secure entry point.  This flag is only
	// intended to be a hint to zone signing or debugging software as to the
	// intended use of this DNSKEY record; validators MUST NOT alter their
	// behavior during the signature validation process in any way based on
	// the setting of this bit.  This also means that a DNSKEY RR with the
	// SEP bit set would also need the Zone Key flag set in order to be able
	// to generate signatures legally.  A DNSKEY RR with the SEP set and the
	// Zone Key flag not set MUST NOT be used to verify RRSIGs that cover
	// RRsets.
	// 
	// Bits 0-6 and 8-14 are reserved: these bits MUST have value 0 upon
	// creation of the DNSKEY RR and MUST be ignored upon receipt.
	Flags uint16
	// The Protocol Field MUST have value 3, and the DNSKEY RR MUST be
	// treated as invalid during signature verification if it is found to be
	// some value other than 3.
	Protocol byte
	// The Algorithm field identifies the public key's cryptographic
	// algorithm and determines the format of the Public Key field.  A list
	// of DNSSEC algorithm types can be found in Appendix A.1
	Algorithm AlgorithmType
	// The Public Key Field holds the public key material.  The format
	// depends on the algorithm of the key being stored and is described in
	// separate documents.
	Key []byte
}

func NewDNSKEY(Flags uint16, Algorithm AlgorithmType, Key []byte) *DNSKEY {
	return &DNSKEY{Flags, 3, Algorithm, Key}
}

// Implementation of dns.Wirer
func (d *DNSKEY) Encode(b *dns.Wirebuf) {
	dns.Octets2(d.Flags).Encode(b)
	dns.Octet(d.Protocol).Encode(b)
	dns.Octet(d.Algorithm).Encode(b)
	b.Buf = append(b.Buf, d.Key...)
}

// Implementation of dns.Wirer
func (d *DNSKEY) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.Octets2)(&d.Flags).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octet)(&d.Protocol).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octet)(&d.Algorithm).Decode(b, pos); err != nil {
		return
	}
	n := len(b) - *pos
	if n <= 0 {
		return fmt.Errorf("(*DNSKEY).Decode: no key data")
	}
	d.Key = make([]byte, n)
	copy(d.Key, b[*pos:])
	*pos += n
	return
}

func (d *DNSKEY) String() string {
	return fmt.Sprintf("%d %d %d %s", d.Flags, d.Protocol, d.Algorithm, strutil.Base64Encode(d.Key))
}

// The delegation signer (DS) resource record (RR) is inserted at a zone
// cut (i.e., a delegation point) to indicate that the delegated zone is
// digitally signed and that the delegated zone recognizes the indicated
// key as a valid zone key for the delegated zone. (RFC 3658)
type DS struct {
	// The key tag is calculated as specified in RFC 2535
	KeyTag uint16
	// Algorithm MUST be allowed to sign DNS data
	AlgorithmType
	// The digest type is an identifier for the digest algorithm used
	DigestType HashAlgorithm
	// The digest is calculated over the
	// canonical name of the delegated domain name followed by the whole
	// RDATA of the KEY record (all four fields)
	Digest []byte
}

// Implementation of dns.Wirer
func (d *DS) Encode(b *dns.Wirebuf) {
	dns.Octets2(d.KeyTag).Encode(b)
	dns.Octet(d.AlgorithmType).Encode(b)
	dns.Octet(d.DigestType).Encode(b)
	b.Buf = append(b.Buf, d.Digest...)
}

// Implementation of dns.Wirer
func (d *DS) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.Octets2)(&d.KeyTag).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octet)(&d.AlgorithmType).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octet)(&d.DigestType).Decode(b, pos); err != nil {
		return
	}
	var n int
	switch d.DigestType {
	case HashAlgorithmSHA1:
		n = 20
	default:
		return fmt.Errorf("unsupported digest type %d", d.DigestType)
	}

	end := *pos + n
	if end > len(b) {
		return fmt.Errorf("DS.Decode - buffer underflow")
	}
	d.Digest = append([]byte{}, b[*pos:end]...)
	*pos = end
	return
}

func (d *DS) String() string {
	if asserts && len(d.Digest) == 0 {
		panic("internal error")
	}

	return fmt.Sprintf("%d %d %d %s", d.KeyTag, d.AlgorithmType, d.DigestType, hex.EncodeToString(d.Digest))
}

type ip4 net.IP

// Implementation of dns.Wirer
func (d ip4) Encode(b *dns.Wirebuf) {
	b4 := net.IP(d).To4()
	if asserts {
		if b4 == nil {
			panic(fmt.Errorf("%s is not an IPv4 address", net.IP(d)))
		}
	}
	b.Buf = append(b.Buf, b4...)
}

// Implementation of dns.Wirer
func (d *ip4) Decode(b []byte, pos *int) (err os.Error) {
	p := *pos
	if p+4 > len(b) {
		return fmt.Errorf("ip4.Decode() - buffer underflow")
	}

	*d = ip4(net.IPv4(b[p], b[p+1], b[p+2], b[p+3]))
	*pos = p + 4
	return
}

type ip6 net.IP

// Implementation of dns.Wirer
func (d ip6) Encode(b *dns.Wirebuf) {
	b16 := net.IP(d).To16()
	if asserts {
		if b16 == nil {
			panic(fmt.Errorf("%s is not an IPv6 address", d))
		}
	}
	b.Buf = append(b.Buf, b16...)
}

// Implementation of dns.Wirer
func (d *ip6) Decode(b []byte, pos *int) (err os.Error) {
	p := *pos
	if p+16 > len(b) {
		return fmt.Errorf("ip6.Decode() - buffer underflow")
	}

	*d = make([]byte, 16)
	copy(*d, b[p:])
	*pos = p + 16
	return
}

// MX holds the zone MX RData
type MX struct {
	// A 16 bit integer which specifies the preference given to
	// this RR among others at the same owner.  Lower values
	// are preferred.
	Preference uint16
	// A <domain-name> which specifies a host willing to act as
	// a mail exchange for the owner name.
	Exchange string
}

// Implementation of dns.Wirer
func (d *MX) Encode(b *dns.Wirebuf) {
	dns.Octets2(d.Preference).Encode(b)
	dns.DomainName(d.Exchange).Encode(b)
}

// Implementation of dns.Wirer
func (d *MX) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.Octets2)(&d.Preference).Decode(b, pos); err != nil {
		return
	}
	return (*dns.DomainName)(&d.Exchange).Decode(b, pos)
}

func (d *MX) String() string {
	return fmt.Sprintf("%d %s", d.Preference, d.Exchange)
}

// NODATA is used for negative caching of authoritative answers
// for queried non existent Type/Class combinations.
type NODATA struct {
	Type // The Type for which we are caching the NODATA
}

// Implementation of dns.Wirer
func (d *NODATA) Encode(b *dns.Wirebuf) {
	d.Type.Encode(b)
}

// Implementation of dns.Wirer
func (d *NODATA) Decode(b []byte, pos *int) (err os.Error) {
	return d.Type.Decode(b, pos)
}

func (d *NODATA) String() string {
	return fmt.Sprintf("%s", d.Type)
}

// NXDOMAIN is used for negative caching of authoritave answers 
// for queried non existing domain names.
type NXDOMAIN struct{}

// Implementation of dns.Wirer
func (d *NXDOMAIN) Encode(b *dns.Wirebuf) {
}

// Implementation of dns.Wirer
func (d *NXDOMAIN) Decode(b []byte, pos *int) (err os.Error) {
	return
}

func (d *NXDOMAIN) String() (s string) {
	return
}

// NS holds the zone NS RData
type NS struct {
	// A <domain-name> which specifies a host which should be
	// authoritative for the specified class and domain.
	NSDName string
}

// Implementation of dns.Wirer
func (d *NS) Encode(b *dns.Wirebuf) {
	dns.DomainName(d.NSDName).Encode(b)
}

// Implementation of dns.Wirer
func (d *NS) Decode(b []byte, pos *int) (err os.Error) {
	return (*dns.DomainName)(&d.NSDName).Decode(b, pos)
}

func (d *NS) String() string {
	return d.NSDName
}

// HashAlgorithm is the type of the hash algorithm in the NSEC3 RR
type HashAlgorithm byte

// IANA registry for "DNSSEC NSEC3 Hash Algorithms".
// Values of HashAlgorithm.
const (
	HashAlgorithmReserved HashAlgorithm = iota
	HashAlgorithmSHA1
)

// The NSEC3 Resource Record (RR) provides authenticated denial of
// existence for DNS Resource Record Sets. (RFC 5155)
type NSEC3 struct {
	NSEC3PARAM
	// The Next Hashed Owner Name field contains the next hashed owner name
	// in hash order.  This value is in binary format.
	NextHashedOwnerName []byte
	// The Type Bit Maps field identifies the RRSet types that exist at the
	// original owner name of the NSEC3 RR
	TypeBitMaps []byte
}

// Implementation of dns.Wirer
func (d *NSEC3) Encode(b *dns.Wirebuf) {
	d.NSEC3PARAM.Encode(b)
	n := dns.Octets2(len(d.NextHashedOwnerName))
	n.Encode(b)
	b.Buf = append(b.Buf, d.NextHashedOwnerName...)
	b.Buf = append(b.Buf, d.TypeBitMaps...)
}

// Implementation of dns.Wirer
func (d *NSEC3) Decode(b []byte, pos *int) (err os.Error) {
	if err = d.NSEC3PARAM.Decode(b, pos); err != nil {
		return
	}

	var n dns.Octets2
	if err = n.Decode(b, pos); err != nil {
		return
	}

	in := int(n)
	if *pos+in > len(b) {
		return fmt.Errorf("rr.*NSEC3.Decode - buffer underflow")
	}

	d.NextHashedOwnerName = append([]byte{}, b[*pos:*pos+in]...)
	*pos = *pos + in

	// here we (have to) rely on b being sliced exactly at the end of the wire format packet
	end := len(b)
	d.TypeBitMaps = append([]byte{}, b[*pos:end]...)
	*pos = end
	return
}

func (d *NSEC3) String() string {
	types, err := TypesDecode(d.TypeBitMaps)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s %s %s", d.NSEC3PARAM.String(), strutil.Base32ExtEncode(d.NextHashedOwnerName), TypesString(types))
}

// The NSEC3PARAM RR contains the NSEC3 parameters (hash algorithm,
// flags, iterations, and salt) needed by authoritative servers to
// calculate hashed owner names. (RFC 5155)
type NSEC3PARAM struct {
	// The Hash Algorithm field identifies the cryptographic hash algorithm
	// used to construct the hash-value.
	HashAlgorithm
	// The Flags field contains 8 one-bit flags that can be used to indicate
	// different processing.  All undefined flags must be zero.  The only
	// flag defined by this specification is the Opt-Out flag.
	Flags byte
	// The Iterations field defines the number of additional times the hash
	// function has been performed.
	Iterations uint16
	// The Salt field is appended to the original owner name before hashing
	// in order to defend against pre-calculated dictionary attacks
	Salt []byte
	// The Hash Length field defines the length of the Next Hashed Owner
	// Name field, ranging in value from 1 to 255 octets
}

// Implementation of dns.Wirer
func (d *NSEC3PARAM) Encode(b *dns.Wirebuf) {
	dns.Octet(d.HashAlgorithm).Encode(b)
	dns.Octet(d.Flags).Encode(b)
	dns.Octets2(d.Iterations).Encode(b)
	if asserts && len(d.Salt) > 255 {
		panic("internal error")
	}
	dns.Octet(len(d.Salt)).Encode(b)
	b.Buf = append(b.Buf, d.Salt...)
}

// Implementation of dns.Wirer
func (d *NSEC3PARAM) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.Octet)(&d.HashAlgorithm).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octet)(&d.Flags).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.Octets2)(&d.Iterations).Decode(b, pos); err != nil {
		return
	}
	var n byte
	if err = (*dns.Octet)(&n).Decode(b, pos); err != nil {
		return
	}
	p := *pos
	next := p + int(n)
	if next > len(b) {
		return fmt.Errorf("NSEC3PARAM.Decode() - buffer underflow")
	}
	d.Salt = append([]byte{}, b[p:next]...)
	*pos = next
	return
}

func (d *NSEC3PARAM) String() string {
	s := hex.EncodeToString(d.Salt)
	if s == "" {
		s = "-"
	}
	return fmt.Sprintf("%d %d %d %s", d.HashAlgorithm, d.Flags, d.Iterations, s)
}

// PTR holds the zone PTR RData
type PTR struct {
	// A <domain-name> which points to some location in the
	// domain name space.
	PTRDName string
}

// Implementation of dns.Wirer
func (d *PTR) Encode(b *dns.Wirebuf) {
	dns.DomainName(d.PTRDName).Encode(b)
}

// Implementation of dns.Wirer
func (d *PTR) Decode(b []byte, pos *int) (err os.Error) {
	return (*dns.DomainName)(&d.PTRDName).Decode(b, pos)
}

func (d *PTR) String() string {
	return d.PTRDName
}

// RDATA hodls DNS RR rdata for a unknown/unsupported RR type
type RDATA []byte

// Implementation of dns.Wirer
func (d *RDATA) Encode(b *dns.Wirebuf) {
	b.Buf = append(b.Buf, *d...)
}

// Implementation of dns.Wirer
func (d *RDATA) Decode(b []byte, pos *int) (err os.Error) {
	n := len(b) - *pos
	*d = b[*pos:]
	*pos += n
	return
}

func (d *RDATA) String() string {
	return fmt.Sprintf("\\# %d % 02x", len(*d), *d)
}

// RR holds a zone resource record data.
type RR struct {
	// An owner name, i.e., the name of the node to which this resource record pertains.
	Name string
	// Two octets containing one of the RR TYPE codes.
	Type
	// Two octets containing one of the RR CLASS codes.
	Class
	// A 32 bit signed integer that specifies the time interval
	// that the resource record may be cached before the source
	// of the information should again be consulted.  Zero
	// values are interpreted to mean that the RR can only be
	// used for the transaction in progress, and should not be
	// cached.  For example, SOA records are always distributed
	// with a zero TTL to prohibit caching.  Zero values can
	// also be used for extremely volatile data. 
	TTL int32
	//The format of this information varies according to the TYPE and CLASS of the resource record.
	RData dns.Wirer
}

func (rr *RR) String() string {
	return fmt.Sprintf("%s\t%s\t%d\t%s %s", rr.Name, rr.Class, rr.TTL, rr.Type, rr.RData)
}

// Implementation of dns.Wirer
func (rr *RR) Encode(b *dns.Wirebuf) {
	dns.DomainName(rr.Name).Encode(b)
	rr.Type.Encode(b)
	rr.Class.Encode(b)
	dns.Octets4(rr.TTL).Encode(b)
	p0 := len(b.Buf)
	b.Buf = append(b.Buf, 0, 0)
	rr.RData.Encode(b)
	n := len(b.Buf) - (p0 + 2)
	b.Buf[p0] = byte(n >> 8)
	b.Buf[p0+1] = byte(n)
}

// Implementation of dns.Wirer
func (rr *RR) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.DomainName)(&rr.Name).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octets2)(&rr.Type).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octets2)(&rr.Class).Decode(b, pos); err != nil {
		return
	}

	var ttl dns.Octets4
	if err = ttl.Decode(b, pos); err != nil {
		return
	}

	rr.TTL = int32(ttl)

	var rdlength dns.Octets2
	if err = rdlength.Decode(b, pos); err != nil {
		return
	}

	switch rr.Type {
	case TYPE_A:
		rr.RData = &A{}
	case TYPE_AAAA:
		rr.RData = &AAAA{}
	case TYPE_CNAME:
		rr.RData = &CNAME{}
	case TYPE_DNSKEY:
		rr.RData = &DNSKEY{}
	case TYPE_DS:
		rr.RData = &DS{}
	case TYPE_MX:
		rr.RData = &MX{}
	case TYPE_NODATA:
		rr.RData = &NODATA{}
	case TYPE_NS:
		rr.RData = &NS{}
	case TYPE_NXDOMAIN:
		rr.RData = &NXDOMAIN{}
	case TYPE_NSEC3:
		rr.RData = &NSEC3{}
	case TYPE_NSEC3PARAM:
		rr.RData = &NSEC3PARAM{}
	case TYPE_PTR:
		rr.RData = &PTR{}
	case TYPE_RRSIG:
		rr.RData = &RRSIG{}
	case TYPE_SOA:
		rr.RData = &SOA{}
	case TYPE_TXT:
		rr.RData = &TXT{}
	default:
		rr.RData = &RDATA{}
	}

	err = rr.RData.Decode(b[:*pos+int(rdlength)], pos)
	return
}

// Equal compares a and b as per rfc2136/1.1
func (a *RR) Equal(b *RR) (equal bool) {
	if a.Type != b.Type || a.Class != b.Class || strings.ToLower(a.Name) != strings.ToLower(b.Name) {
		return
	}

	// Name, Type, Class match
	switch x := a.RData.(type) {
	default:
		log.Fatalf("rr.RR.Equal() - internal error %T", x)
	case *RDATA:
		return bytes.Equal(*x, *b.RData.(*RDATA))
	case *A:
		return x.Address.String() == b.RData.(*A).Address.String()
	case *AAAA:
		return x.Address.String() == b.RData.(*AAAA).Address.String()
	case *CNAME:
		return strings.ToLower(x.Name) == strings.ToLower(b.RData.(*CNAME).Name)
	case *DNSKEY:
		y := b.RData.(*DNSKEY)
		return x.Flags == y.Flags &&
			x.Protocol == y.Protocol &&
			x.Algorithm == y.Algorithm &&
			bytes.Equal(x.Key, y.Key)
	case *DS:
		y := b.RData.(*DS)
		return x.KeyTag == y.KeyTag &&
			x.AlgorithmType == y.AlgorithmType &&
			x.DigestType == y.DigestType &&
			bytes.Equal(x.Digest, y.Digest)
	case *MX:
		y := b.RData.(*MX)
		return x.Preference == y.Preference &&
			strings.ToLower(x.Exchange) == strings.ToLower(y.Exchange)
	case *NODATA:
		y := b.RData.(*NODATA)
		return x.Type == y.Type
	case *NXDOMAIN:
		return true
	case *NS:
		y := b.RData.(*NS)
		return strings.ToLower(x.NSDName) == strings.ToLower(y.NSDName)
	case *NSEC3:
		y := b.RData.(*NSEC3)
		return x.HashAlgorithm == y.HashAlgorithm &&
			x.Flags == y.Flags &&
			x.Iterations == y.Iterations &&
			bytes.Equal(x.Salt, y.Salt) &&
			bytes.Equal(x.NextHashedOwnerName, y.NextHashedOwnerName) &&
			bytes.Equal(x.TypeBitMaps, y.TypeBitMaps)
	case *NSEC3PARAM:
		y := b.RData.(*NSEC3PARAM)
		return x.HashAlgorithm == y.HashAlgorithm &&
			x.Flags == y.Flags &&
			x.Iterations == y.Iterations &&
			bytes.Equal(x.Salt, y.Salt)
	case *PTR:
		y := b.RData.(*PTR)
		return strings.ToLower(x.PTRDName) == strings.ToLower(y.PTRDName)
	case *RRSIG:
		y := b.RData.(*RRSIG)
		return x.Type == y.Type &&
			x.AlgorithmType == y.AlgorithmType &&
			x.Labels == y.Labels &&
			x.TTL == y.TTL &&
			x.Expiration == y.Expiration &&
			x.KeyTag == y.KeyTag &&
			strings.ToLower(x.Name) == strings.ToLower(y.Name) &&
			bytes.Equal(x.Signature, y.Signature)
	case *SOA:
		y := b.RData.(*SOA)
		return strings.ToLower(x.MName) == strings.ToLower(y.MName) &&
			strings.ToLower(x.RName) == strings.ToLower(y.RName) &&
			x.Serial == y.Serial &&
			x.Refresh == y.Refresh &&
			x.Retry == y.Retry &&
			x.Expire == y.Expire &&
			x.Minimum == y.Minimum
	case *TXT:
		return x.S == b.RData.(*TXT).S
	}
	return
}

// RRs is a slice of resource records with attached convenience methods
type RRs []*RR

// Filter divides r to the wanted and other partitions.
func (r RRs) Filter(want func(r *RR) bool) (wanted, other RRs) {
	for _, r := range r {
		if want(r) {
			wanted = append(wanted, r)
		} else {
			other = append(other, r)
		}
	}
	return
}

func (r RRs) String() string {
	a := make([]string, len(r))
	for i, rec := range r {
		a[i] = rec.String()
	}
	return strings.Join(a, "\n")
}

// SetAdd computes a set union of r and rrs. Set membership predicate is RR.Equal,
// i.e. only resource records from rrs not comparing equal to any resource records
// in r are added/merged into the result set.
func (r *RRs) SetAdd(rrs RRs) {
	for _, newrec := range rrs {
		isnew := true
		for _, oldrec := range *r {
			if newrec.Equal(oldrec) {
				isnew = false
				break
			}
		}

		if isnew {
			*r = append(*r, newrec)
		}
	}
}

// Unique filters out any records from r which are Equal to any other record in r.
func (r *RRs) Unique() {
	y := RRs{}
	y.SetAdd(*r)
	*r = y
}

// Partition groups resource record of the same type.
// If unique == true then the result parts are processed by Unique.
func (r RRs) Partition(unique bool) (parts Parts) {
	parts = make(map[Type]RRs, len(r))
	for _, v := range r {
		parts[v.Type] = append(parts[v.Type], v)
	}
	if unique {
		for typ, part := range parts {
			part.Unique()
			parts[typ] = part
		}
	}
	return
}

// Pack packs r to Bytes
func (r RRs) Pack() (y Bytes) {
	y.Pack(r)
	return
}

// Parts is the type returned by Partition()
type Parts map[Type]RRs

// Join returns all parts of p.
func (p Parts) Join() (rrs RRs) {
	for _, part := range p {
		rrs = append(rrs, part...)
	}
	return
}

// SetAdd returns a set union of a and b. Set membership predicate is RR.Equal,
// i.e. only resource records from b not comparing equal to any resource records
// in a are added/merged into a.
func (a Parts) SetAdd(b Parts) {
	for newtyp, newrecs := range b {
		oldrecs, ok := a[newtyp]
		if !ok {
			a[newtyp] = newrecs
			continue
		}

		oldrecs.SetAdd(newrecs)
		a[newtyp] = oldrecs

	}
	return
}

// RRSIG holds the zone RRSIG RData (RFC4034)
type RRSIG struct {
	// The Type Covered field identifies the type of the RRset that is covered 
	// by this RRSIG record.
	Type
	//  The Algorithm Number field identifies the cryptographic algorithm used
	// to create the signature. 
	AlgorithmType
	// The Labels field specifies the number of labels in the original RRSIG RR owner name.
	Labels byte
	// The Original TTL field specifies the TTL of the covered RRset as it appears
	// in the authoritative zone.
	TTL int32
	// The Signature Expiration field specifies a validity period for the signature.
	// The RRSIG record MUST NOT be used for authentication after the expiration date.
	Expiration uint32
	// The Signature Inception field specifies a validity period for the signature.
	// The RRSIG record MUST NOT be used for authentication prior to the inception date.
	Inception uint32
	// The Key Tag field contains the key tag value of the DNSKEY RR that validates
	// this signature, in network byte order.
	KeyTag uint16
	// The Signer's Name field value identifies the owner name of the DNSKEY
	// RR that a validator is supposed to use to validate this signature.
	Name string
	// The Signature field contains the cryptographic signature that covers
	// the RRSIG RDATA (excluding the Signature field) and the RRset
	// specified by the RRSIG owner name, RRSIG class, and RRSIG Type
	// Covered field.
	Signature []byte
}

// Implementation of dns.Wirer
func (r *RRSIG) Encode(b *dns.Wirebuf) {
	dns.Octets2(r.Type).Encode(b)
	dns.Octet(r.AlgorithmType).Encode(b)
	dns.Octet(r.Labels).Encode(b)
	dns.Octets4(r.TTL).Encode(b)
	dns.Octets4(r.Expiration).Encode(b)
	dns.Octets4(r.Inception).Encode(b)
	dns.Octets2(r.KeyTag).Encode(b)
	b.DisableCompression()
	(*dns.DomainName)(&r.Name).Encode(b)
	b.EnableCompression()
	b.Buf = append(b.Buf, r.Signature...)
}

// Implementation of dns.Wirer
func (r *RRSIG) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.Octets2)(&r.Type).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octet)(&r.AlgorithmType).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octet)(&r.Labels).Decode(b, pos); err != nil {
		return
	}

	var ttl dns.Octets4
	if err = ttl.Decode(b, pos); err != nil {
		return
	}

	r.TTL = int32(ttl)

	if err = (*dns.Octets4)(&r.Expiration).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octets4)(&r.Inception).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.Octets2)(&r.KeyTag).Decode(b, pos); err != nil {
		return
	}

	if err = (*dns.DomainName)(&r.Name).Decode(b, pos); err != nil {
		return
	}

	n := len(b) - *pos
	if n <= 0 {
		return fmt.Errorf("(*RRSIG).Decode: no signature data")
	}

	r.Signature = make([]byte, n)
	copy(r.Signature, b[*pos:])
	*pos += n
	return
}

func (r *RRSIG) String() string {
	return fmt.Sprintf("%s %d %d %d %s %s %d %s %s",
		r.Type,
		r.AlgorithmType,
		r.Labels,
		r.TTL,
		time.SecondsToUTC(int64(r.Expiration)).Format(timeLayout),
		time.SecondsToUTC(int64(r.Inception)).Format(timeLayout),
		r.KeyTag,
		r.Name,
		strutil.Base64Encode(r.Signature),
	)
}

// SOA holds the zone SOA RData
type SOA struct {
	// The <domain-name> of the name server that was the
	// original or primary source of data for this zone.
	MName string
	// A <domain-name> which specifies the mailbox of the
	// person responsible for this zone.
	RName string
	// The unsigned 32 bit version number of the original copy
	// of the zone.  Zone transfers preserve this value.  This
	// value wraps and should be compared using sequence space
	// arithmetic.
	Serial uint32
	// A 32 bit time interval before the zone should be
	// refreshed.
	Refresh uint32
	// A 32 bit time interval that should elapse before a
	// failed refresh should be retried.
	Retry uint32
	// A 32 bit time value that specifies the upper limit on
	// the time interval that can elapse before the zone is no
	// longer authoritative.
	Expire uint32
	// The unsigned 32 bit minimum TTL field that should be
	// exported with any RR from this zone.
	Minimum uint32
}

// Implementation of dns.Wirer
func (d *SOA) Encode(b *dns.Wirebuf) {
	dns.DomainName(d.MName).Encode(b)
	dns.DomainName(d.RName).Encode(b)
	dns.Octets4(d.Serial).Encode(b)
	dns.Octets4(d.Refresh).Encode(b)
	dns.Octets4(d.Retry).Encode(b)
	dns.Octets4(d.Expire).Encode(b)
	dns.Octets4(d.Minimum).Encode(b)
}

// Implementation of dns.Wirer
func (d *SOA) Decode(b []byte, pos *int) (err os.Error) {
	if err = (*dns.DomainName)(&d.MName).Decode(b, pos); err != nil {
		return
	}
	if err = (*dns.DomainName)(&d.RName).Decode(b, pos); err != nil {
		return
	}
	if (*dns.Octets4)(&d.Serial).Decode(b, pos); err != nil {
		return
	}
	if (*dns.Octets4)(&d.Refresh).Decode(b, pos); err != nil {
		return
	}
	if (*dns.Octets4)(&d.Retry).Decode(b, pos); err != nil {
		return
	}
	if (*dns.Octets4)(&d.Expire).Decode(b, pos); err != nil {
		return
	}
	return (*dns.Octets4)(&d.Minimum).Decode(b, pos)
}

func (d *SOA) String() string {
	return fmt.Sprintf("%s %s %d %d %d %d %d", d.MName, d.RName, d.Serial, d.Refresh, d.Retry, d.Expire, d.Minimum)
}

// TXT holds the TXT RData
type TXT struct {
	S string
}

// Implementation of dns.Wirer
func (t *TXT) Encode(b *dns.Wirebuf) {
	dns.CharString(t.S).Encode(b)
}

// Implementation of dns.Wirer
func (t *TXT) Decode(b []byte, pos *int) (err os.Error) {
	s := ""
	for *pos < len(b) {
		var part dns.CharString
		if err = part.Decode(b, pos); err != nil {
			return
		}

		s += string(part)
	}
	t.S = s
	return
}

func (t *TXT) String() string {
	return fmt.Sprintf(`"%s"`, strings.Replace(t.S, `"`, `\"`, -1))
}

// TYPE fields are used in resource records.  Note that these types are a
// subset of msg.QTYPEs.
type Type uint16

// Type codes
const (
	TYPE_A          Type = iota + 1 //   1: a host address (IPv4)
	TYPE_NS                         //   2: an authoritative name server
	TYPE_MD                         //   3: a mail destination (Obsolete - use MX)
	TYPE_MF                         //   4: a mail forwarder (Obsolete - use MX)
	TYPE_CNAME                      //   5: the canonical name for an alias
	TYPE_SOA                        //   6: marks the start of a zone of authority
	TYPE_MB                         //   7: a mailbox domain name (EXPERIMENTAL)
	TYPE_MG                         //   8: a mail group member (EXPERIMENTAL)
	TYPE_MR                         //   9: a mail rename domain name (EXPERIMENTAL)
	TYPE_NULL                       //  10: a null RR (EXPERIMENTAL)
	TYPE_WKS                        //  11: a well known service description
	TYPE_PTR                        //  12: a domain name pointer
	TYPE_HINFO                      //  13: host information
	TYPE_MINFO                      //  14: mailbox or mail list information
	TYPE_MX                         //  15: mail exchange
	TYPE_TXT                        //  16: text strings
	TYPE_AAAA       Type = 28       //  28: a host address (IPv6)
	TYPE_DS         Type = 43       //  43: delegation signer
	TYPE_RRSIG      Type = 46       //  46: RR set signature
	TYPE_NSEC       Type = 47       //  47: authenticated denial of existence
	TYPE_DNSKEY     Type = 48       //  48: DNS key
	TYPE_NSEC3      Type = 50       //  50: authenticated denial of existence
	TYPE_NSEC3PARAM Type = 51       //  51: NSEC3 parameters

	TYPE_NODATA   Type = 0xFF00 //      Pseudo types in the "reserved for private use" area
	TYPE_NXDOMAIN Type = 0xFF01
)

var typeStr = map[Type]string{
	TYPE_A:          "A",
	TYPE_AAAA:       "AAAA",
	TYPE_CNAME:      "CNAME",
	TYPE_DNSKEY:     "DNSKEY",
	TYPE_DS:         "DS",
	TYPE_HINFO:      "HINFO",
	TYPE_MB:         "MB",
	TYPE_MD:         "MD",
	TYPE_MF:         "MF",
	TYPE_MG:         "MG",
	TYPE_MINFO:      "MINFO",
	TYPE_MR:         "MR",
	TYPE_MX:         "MX",
	TYPE_NS:         "NS",
	TYPE_NSEC:       "NSEC",
	TYPE_NSEC3:      "NSEC3",
	TYPE_NSEC3PARAM: "NSEC3PARAM",
	TYPE_NULL:       "NULL",
	TYPE_PTR:        "PTR",
	TYPE_RRSIG:      "RRSIG",
	TYPE_SOA:        "SOA",
	TYPE_TXT:        "TXT",
	TYPE_WKS:        "WKS",
	TYPE_NODATA:     "NODATA",
	TYPE_NXDOMAIN:   "NXDOMAIN",
}

func (n Type) String() (s string) {
	var ok bool
	if s, ok = typeStr[n]; !ok {
		panic(fmt.Errorf("unexpected Type %d", uint16(n)))
	}
	return
}

// Implementation of dns.Wirer
func (n Type) Encode(b *dns.Wirebuf) {
	dns.Octets2(n).Encode(b)
}

// Implementation of dns.Wirer
func (n *Type) Decode(b []byte, pos *int) (err os.Error) {
	return (*dns.Octets2)(n).Decode(b, pos)
}
