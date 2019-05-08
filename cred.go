package iglocparser

import (
	"github.com/ansel1/merry"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type IgApiCredentials struct {
	CSRFToken     string
	IgAppID       string
	InstagramAJAX string
}

func (self *IgApiCredentials) setToHeaders(h http.Header) {
	h.Set("X-CSRFToken", self.CSRFToken)
	h.Set("X-Ig-App-Id", self.IgAppID)
	h.Set("X-Instagram-Ajax", self.InstagramAJAX)
}

func NewAuthorizedClient(client *Client, creds *IgApiCredentials) *AuthorizedClient {
	return &AuthorizedClient{
		Client: client,
		creds:  creds,
	}
}

func ParseIgApiCredentialsFromPage(client *Client, link string) (*IgApiCredentials, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	client.SetHeaders(req.Header, "")

	res, err := client.Do(req)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, merry.WithHTTPCode(ErrInvalidResponseStatus, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	csrfTokenMatches := regexCSRFTokenFinder.FindSubmatch(body)
	if len(csrfTokenMatches) < 2 {
		return nil, merry.Wrap(merry.Errorf("failed to find CSRFToken on requsted page"))
	}
	csrfToken := string(csrfTokenMatches[1])

	instagramAjaxMatches := regexInstagramAjaxFinder.FindSubmatch(body)
	if len(instagramAjaxMatches) < 2 {
		return nil, merry.Wrap(merry.Errorf("failed to find Instagram-Ajax on requsted page"))
	}
	instagramAjax := string(instagramAjaxMatches[1])

	igAppIdScriptLinkMatches := regexIgAppIdScriptLinkFinder.FindSubmatch(body)
	if len(igAppIdScriptLinkMatches) < 2 {
		return nil, merry.Wrap(merry.Errorf("failed to find IG-App-ID script link"))
	}
	igAppIdScriptLink := string(igAppIdScriptLinkMatches[1])

	igAppId, err := parseIgAppIdFromScriptFile(client, GetIgLink(igAppIdScriptLink), link)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return &IgApiCredentials{
		CSRFToken:     csrfToken,
		IgAppID:       igAppId,
		InstagramAJAX: instagramAjax,
	}, nil
}

var regexCSRFTokenFinder = regexp.MustCompile(`csrf_token":"(\w+?)"`)
var regexInstagramAjaxFinder = regexp.MustCompile(`rollout_hash":"(\w+?)"`)
var regexIgAppIdScriptLinkFinder = regexp.MustCompile(`"(/.+?ConsumerLibCommons\.js.+?)"`)

var regexIgAppIdFinder = regexp.MustCompile(`e\.instagramWebDesktopFBAppId='(\w+?)'`)

func parseIgAppIdFromScriptFile(client *Client, link string, referer string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", client.getUserAgent())
	setReferrerToHeader(req.Header, referer)
	req.Header.Set("Origin", IgHost)

	res, err := client.Do(req)
	if err != nil {
		return "", merry.Wrap(err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", merry.WithHTTPCode(ErrInvalidResponseStatus, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", merry.Wrap(err)
	}

	igAppIdMatches := regexIgAppIdFinder.FindSubmatch(body)
	if len(igAppIdMatches) < 2 {
		return "", merry.Errorf("failed to find Ig-App-ID on requested page")
	}

	return string(igAppIdMatches[1]), nil
}

func CreateAuthorizedClient(client *Client) (*AuthorizedClient, error) {
	creds, err := ParseIgApiCredentialsFromPage(client, GetIgLinkWithLeadingSlash(IgExploreLocationsPath))
	if err != nil {
		return nil, err
	}

	return NewAuthorizedClient(client, creds), nil
}

func CreateDefaultAuthorizedClient(proxy *url.URL, timeout time.Duration) (*AuthorizedClient, error) {
	return CreateAuthorizedClient(NewClient(proxy, timeout))
}

func CreateDefaultIgApiClient(proxy *url.URL, timeout time.Duration) (*IgApiClient, error) {
	ac, err := CreateDefaultAuthorizedClient(proxy, timeout)
	if err != nil {
		return nil, err
	}

	return NewIgApiClient(ac), nil
}
