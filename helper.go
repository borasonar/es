package main

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func getTransport(proxy bool) http.RoundTripper {
	transport := http.Transport{}
	if proxy {
		url_i := url.URL{}
		url_proxy, _ := url_i.Parse("http://localhost:8080")
		transport.Proxy = http.ProxyURL(url_proxy)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &CustomTransport{
		Base: &transport,
	}
}

func addHeaders(h *http.Header) {
	h.Add("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8`)
	h.Add("Accept-Language", "en-US,en;q=0.7")
	h.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	//h.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:137.0) Gecko/20100101 Firefox/137.0")
	h.Add("Content-Type", "application/x-www-form-urlencoded")
}

type CustomTransport struct {
	Base http.RoundTripper
}

/**
* Kukiji koji sadrze [] u nazivu podrazumevano ne bivaju parsirani i ne dodaju se u Jar
* Ova funkcija prolazi kroz Set-Cookie hedere i dodaje one cije ime sadrzi [
 */
func cookieHack(c *http.Client, resp *http.Response) {
	u, _ := url.Parse(BASE_URL)
	bytes, _ := httputil.DumpResponse(resp, false)
	headers := string(bytes)

	r := regexp.MustCompile(`Set-Cookie:\s?(?P<name>[^=]+)=(?P<value>[^;]+)`)
	m := r.FindAllStringSubmatch(headers, -1)
	if m == nil {
		return
	}
	cookies := c.Jar.Cookies(u)
	for _, pair := range m {
		nameIdx := r.SubexpIndex("name")
		valueIdx := r.SubexpIndex("value")
		name := pair[nameIdx]
		if !strings.Contains(name, "[") {
			continue
		}
		value := pair[valueIdx]
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}
	c.Jar.SetCookies(u, cookies)
}

type contextKey string

const clientKey = contextKey("client-ref")

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	val := req.Context().Value(clientKey)
	c, ok := val.(*http.Client)
	if !ok {
		panic("nije prosledjen http klijent objekat")
	}
	resp, err := t.Base.RoundTrip(req)
	if err == nil {
		cookieHack(c, resp)
	}
	return resp, err
}

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	proxy := false
	parsedBaseUrl, err := url.Parse(BASE_URL)
	if err != nil {
		panic(err)
	}
	return &http.Client{
		Timeout:   time.Duration(3000) * time.Second,
		Jar:       jar,
		Transport: getTransport(proxy),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host != parsedBaseUrl.Host {
				return http.ErrUseLastResponse //errors.New("pokusaj redirekcije na domen " + req.URL.Host)
			}
			return nil //http.ErrUseLastResponse
		},
	}
}

func openFile(uri string) *os.File {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	path := baseDir + "/" + uri
	file, err := os.Open(path)
	if err != nil {
		panic("fajl " + uri + " nije nađen")
	}
	return file
}

func backupConf(uri string) {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	source, err := os.Open(baseDir + "/" + uri)
	if err != nil {
		panic(err.Error())
	}
	defer source.Close()

	destination, err := os.Create(baseDir + "/backup-cfg/" + uri + ".backup")
	if err != nil {
		panic(err.Error())
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		panic(err.Error())
	}
}
