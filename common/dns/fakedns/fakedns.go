package fakedns

import (
	"errors"
	"fmt"
	"net"

	"github.com/miekg/dns"

	cdns "github.com/ruilisi/go-tun2socks/common/dns"
	"github.com/ruilisi/go-tun2socks/common/dns/fakeip"
	"github.com/ruilisi/go-tun2socks/common/log"
	"github.com/ruilisi/go-tun2socks/component/pool"
)

const (
	FakeResponseTtl uint32 = 1 // in sec
)

type fakeDNS struct {
	fakePool *fakeip.Pool
}

func canHandleDnsQuery(data []byte) bool {
	req := new(dns.Msg)
	err := req.Unpack(data)
	if err != nil {
		log.Debugf("cannot handle dns query: failed to unpack")
		return false
	}
	if len(req.Question) != 1 {
		log.Debugf("cannot handle dns query: multiple questions")
		return false
	}
	qtype := req.Question[0].Qtype
	if qtype != dns.TypeA && qtype != dns.TypeAAAA {
		log.Debugf("cannot handle dns query: not A/AAAA qtype")
		return false
	}
	qclass := req.Question[0].Qclass
	if qclass != dns.ClassINET {
		log.Debugf("cannot handle dns query: not ClassINET")
		return false
	}
	fqdn := req.Question[0].Name
	domain := fqdn[:len(fqdn)-1]
	if _, ok := dns.IsDomainName(domain); !ok {
		log.Debugf("cannot handle dns query: invalid domain name")
		return false
	}
	return true
}

func NewFakeDNS(ipnet *net.IPNet, size int) cdns.FakeDns {
	fP, _ := fakeip.New(ipnet, size, nil)
	return &fakeDNS{
		fakePool: fP,
	}
}

func (f *fakeDNS) QueryDomain(ip net.IP) string {
	domain, found := f.fakePool.LookBack(ip)
	if found {
		log.Debugf("fake dns returns domain %v for ip %v", domain, ip)
	}
	return domain
}

func (f *fakeDNS) GenerateFakeResponse(request []byte) ([]byte, error) {
	if !canHandleDnsQuery(request) {
		return nil, errors.New("cannot handle DNS request")
	}
	req := new(dns.Msg)
	req.Unpack(request)
	qtype := req.Question[0].Qtype
	fqdn := req.Question[0].Name
	domain := fqdn[:len(fqdn)-1]
	ip := f.fakePool.Lookup(domain) // fakeIP uses IPv4 range
	log.Debugf("fake dns allocated ip %v for domain %v", ip, domain)
	resp := new(dns.Msg)
	resp = resp.SetReply(req)
	if qtype == dns.TypeA {
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:     fqdn,
				Rrtype:   dns.TypeA,
				Class:    dns.ClassINET,
				Ttl:      FakeResponseTtl,
				Rdlength: net.IPv4len,
			},
			A: ip,
		})
	} else if qtype == dns.TypeAAAA {
		// use valid IPv6 form for resp.PackBuffer()
		ipMappedTov6 := net.ParseIP("::ffff:" + ip.String())
		resp.Answer = append(resp.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:     fqdn,
				Rrtype:   dns.TypeAAAA,
				Class:    dns.ClassINET,
				Ttl:      FakeResponseTtl,
				Rdlength: net.IPv6len,
			},
			AAAA: ipMappedTov6,
		})
	} else {
		return nil, fmt.Errorf("unexcepted dns qtype %v", qtype)
	}
	buf := pool.NewBytes(65535)
	defer pool.FreeBytes(buf)
	dnsAnswer, err := resp.PackBuffer(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to pack dns answer: %v, err %v", resp.String(), err)
	}
	return append([]byte(nil), dnsAnswer...), nil
}

func (f *fakeDNS) IsFakeIP(ip net.IP) bool {
	return f.fakePool.Exist(ip)
}
