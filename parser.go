package iglocparser

import (
	"github.com/ansel1/merry"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidResponseStatus = errors.New("invalid response status code")
var ErrUndefinedLocation = errors.New("undefined location")
var ErrInvalidIgApiResponseCode = errors.New("invalid ig api locations response code")

var DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.81 Safari/537.36"

const IgHost = "https://www.instagram.com"
const IgExploreLocationsPath = "explore/locations"

func init() {
	rand.Seed(time.Now().Unix())
}

func GetIgLink(paths ...string) string {
	return IgHost + "/" + strings.Trim(path.Join(paths...), "/")
}

func GetIgLinkWithLeadingSlash(paths ...string) string {
	return GetIgLink(paths...) + "/"
}

func getInMemoryCookieJar() *cookiejar.Jar {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(merry.Append(err, "failed to create cookies jar instance").Error())
	}

	return jar
}

type Client struct {
	*http.Client

	UserAgent string
	Headers   http.Header
}

func (self *Client) getUserAgent() string {
	if self.UserAgent == "" {
		return DefaultUserAgent
	}

	return self.UserAgent
}

func (self *Client) SetHeaders(h http.Header, referrer string) {
	h.Set("User-Agent", self.getUserAgent())
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	h.Set("Accept-Language", "en")
	h.Set("Cache-Control", "no-cache")
	h.Set("Pragma", "no-cache")
	h.Set("Origin", IgHost)
	setReferrerToHeader(h, referrer)

	if self.Headers != nil {
		for key, headers := range self.Headers {
			h.Del(key)

			if headers == nil {
				continue
			}

			for _, value := range headers {
				h.Add(key, value)
			}
		}
	}
}

func (self *Client) GetWithHeaders(url string, referrer string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	self.SetHeaders(req.Header, referrer)
	return self.Do(req)
}

func setReferrerToHeader(h http.Header, referrer string) {
	if referrer == "" {
		return
	}

	h.Set("Referer", referrer)
}

func NewClient(proxy *url.URL, timeout time.Duration) *Client {
	c := &http.Client{
		Jar: getInMemoryCookieJar(),
	}

	transport := &http.Transport{
		IdleConnTimeout:       timeout,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: timeout,
	}

	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}

	c.Transport = transport
	c.Timeout = timeout

	return &Client{
		Client: c,
	}
}

type AuthorizedClient struct {
	*Client
	creds *IgApiCredentials
}

func (self *AuthorizedClient) SetHeaders(h http.Header, referrer string) {
	self.Client.SetHeaders(h, referrer)
	self.creds.setToHeaders(h)
}

type igApiCursor struct {
	nextPage    int
	hasNextPage bool
}

func (self *igApiCursor) Has() bool {
	return self.hasNextPage
}

func (self *igApiCursor) setNextPage(nextPage *int) {
	if nextPage == nil {
		self.hasNextPage = false
	} else {
		self.nextPage = *nextPage
	}
}

type IgApiClient struct {
	client *AuthorizedClient
}

func (self *IgApiClient) Credentials() *IgApiCredentials {
	return self.client.creds
}

func (self *IgApiClient) GetClient() *Client {
	return self.client.Client
}

func NewIgApiClient(client *AuthorizedClient) *IgApiClient {
	return &IgApiClient{
		client: client,
	}
}

func CreateIgApiClient(client *Client) (*IgApiClient, error) {
	ac, err := CreateAuthorizedClient(client)
	if err != nil {
		return nil, err
	}

	return NewIgApiClient(ac), nil
}

func (self *IgApiClient) request(link string, page int, referer string) (*http.Response, error) {
	reqdata := url.Values{}
	reqdata.Set("page", strconv.Itoa(page))

	res, err := self.post(link, reqdata, referer)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return res, nil
}

func (self *IgApiClient) post(link string, data url.Values, referrer string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, link, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, merry.Wrap(err)
	}

	self.client.SetHeaders(req.Header, referrer)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := self.client.Do(req)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return res, nil
}

func (self *IgApiClient) do(link string, page int, referrer string) ([]byte, error) {
	resp, err := self.request(link, page, referrer)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, merry.WithHTTPCode(ErrInvalidResponseStatus, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return body, nil
}

type IgApiClientRotator struct {
	clients []*IgApiClient
	i       int
}

func NewIgApiClientRotator(clients []*IgApiClient) *IgApiClientRotator {
	return &IgApiClientRotator{
		clients: clients,
	}
}

func (self *IgApiClientRotator) Next() *IgApiClient {
	client := self.clients[self.i]
	self.i++
	if self.i > len(self.clients)-1 {
		self.i = 0
	}

	return client
}

func (self *IgApiClientRotator) Len() int {
	return len(self.clients)
}
