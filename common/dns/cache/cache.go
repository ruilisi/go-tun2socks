// This file is copied from https://github.com/yinghuocho/gotun2socks/blob/master/udp.go

package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/miekg/dns"

	"github.com/ruilisi/go-tun2socks/common/cache"
	cdns "github.com/ruilisi/go-tun2socks/common/dns"
	"github.com/ruilisi/go-tun2socks/common/log"
)

const minCleanupInterval = 5 * time.Minute

type dnsCacheEntry struct {
	msg []byte
}

type simpleDnsCache struct {
	storage *cache.Cache
}

func NewSimpleDnsCache() cdns.DnsCache {
	s := cache.New(minCleanupInterval)
	return &simpleDnsCache{
		storage: s,
	}
}

func packUint16(i uint16) []byte { return []byte{byte(i >> 8), byte(i)} }

func cacheKey(q dns.Question) string {
	return q.String()
}

func (c *simpleDnsCache) Query(payload []byte) ([]byte, error) {
	request := new(dns.Msg)
	e := request.Unpack(payload)
	if e != nil {
		return nil, e
	}
	if len(request.Question) == 0 {
		return nil, errors.New("simpleDnsCache: request.Question is empty")
	}

	key := cacheKey(request.Question[0])
	entryInterface := c.storage.Get(key)
	if entryInterface == nil {
		log.Debugf("simpleDnsCache: no entry found in DnsCache, key: %v", key)
		// not an error
		return nil, nil
	}
	entry := entryInterface.(*dnsCacheEntry)
	if entry == nil {
		return nil, errors.New("simpleDnsCache: nil pointer in DnsCache entry")
	}

	resp := new(dns.Msg)
	resp.Unpack(entry.msg)
	resp.Id = request.Id
	var buf [1024]byte
	dnsAnswer, err := resp.PackBuffer(buf[:])
	if err != nil {
		return nil, err
	}
	log.Debugf("simpleDnsCache: got dns answer from cache with key: %v", key)
	return append([]byte(nil), dnsAnswer...), nil
}

func (c *simpleDnsCache) Store(payload []byte) error {
	resp := new(dns.Msg)
	e := resp.Unpack(payload)
	if e != nil {
		return e
	}
	if resp.Rcode != dns.RcodeSuccess {
		return errors.New(fmt.Sprintf("simpleDnsCache: resp.Rcode not RcodeSuccess: DNS resp is: %v", resp.String()))
	}
	if len(resp.Question) == 0 || len(resp.Answer) == 0 {
		return errors.New("simpleDnsCache: resp.Question or resp.Answer is empty")
	}

	key := cacheKey(resp.Question[0])
	ttl := resp.Answer[0].Header().Ttl
	value := &dnsCacheEntry{
		msg: payload,
	}
	c.storage.Put(key, value, time.Duration(ttl)*time.Second)

	log.Debugf("simpleDnsCache: stored dns answer with key: %v, ttl: %v sec", key, ttl)
	return nil
}
