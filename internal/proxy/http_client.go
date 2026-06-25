package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

type ProxyClient struct {
	transportCache map[string]*http.Transport
	mu             sync.RWMutex
}

func NewProxyClient() *ProxyClient {
	return &ProxyClient{
		transportCache: make(map[string]*http.Transport),
	}
}

func (pc *ProxyClient) GetClient(proxyURL string) (*http.Client, error) {
	pc.mu.RLock()
	t, ok := pc.transportCache[proxyURL]
	pc.mu.RUnlock()

	if ok {
		return pc.clientForTransport(t), nil
	}

	return pc.createClient(proxyURL)
}

func (pc *ProxyClient) clientForTransport(transport *http.Transport) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
			}
			return nil
		},
	}
}

func (pc *ProxyClient) createClient(proxyURL string) (*http.Client, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if t, ok := pc.transportCache[proxyURL]; ok {
		return pc.clientForTransport(t), nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	var transport *http.Transport

	switch u.Scheme {
	case "http", "https":
		transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
	case "socks5", "socks", "socks4":
		dialer, err := proxy.FromURL(u, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("socks proxy error: %w", err)
		}
		transport = &http.Transport{
			Dial: dialer.Dial,
		}
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}

	pc.transportCache[proxyURL] = transport
	return pc.clientForTransport(transport), nil
}

type BunkerWebResult struct {
	Cookie string
	OK     bool
}

func SolveBunkerWebChallenge(body string) BunkerWebResult {
	prefix := "document.cookie"
	idx := strings.Index(body, prefix)
	if idx == -1 {
		return BunkerWebResult{OK: false}
	}

	rest := body[idx:]

	start := strings.Index(rest, "'")
	if start == -1 {
		return BunkerWebResult{OK: false}
	}
	start++
	end := strings.Index(rest[start:], "'")
	if end == -1 {
		return BunkerWebResult{OK: false}
	}
	cookieName := rest[start : start+end]

	afterName := rest[start+end:]
	valueStart := strings.Index(afterName, "'")
	if valueStart == -1 {
		return BunkerWebResult{OK: false}
	}
	valueStart++
	valueEnd := strings.Index(afterName[valueStart:], "'")
	if valueEnd == -1 {
		return BunkerWebResult{OK: false}
	}
	challengeValue := afterName[valueStart : valueStart+valueEnd]

	targetPrefix := "0000"
	for i := 0; i < 10000000; i++ {
		candidate := fmt.Sprintf("%s%d", challengeValue, i)
		hash := sha256.Sum256([]byte(candidate))
		hexHash := hex.EncodeToString(hash[:])
		if strings.HasPrefix(hexHash, targetPrefix) {
			return BunkerWebResult{
				Cookie: fmt.Sprintf("%s=%s%d", cookieName, challengeValue, i),
				OK:     true,
			}
		}
	}

	return BunkerWebResult{OK: false}
}

type DirectClient struct {
	*http.Client
}

func NewDirectClient() *DirectClient {
	jar, _ := cookiejar.New(nil)
	return &DirectClient{
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func FetchBody(client *http.Client, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body failed: %w", err)
	}
	return body, resp.StatusCode, nil
}
