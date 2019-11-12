package iglocparser

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var IgAuthenticateUrl = "https://www.instagram.com/accounts/login/ajax/"

type IgAuthenticateResponse struct {
	Authenticated bool   `json:"authenticated"`
	User          bool   `json:"user"`
	UserId        string `json:"user_id"`
	OneTapPrompt  bool   `json:"one_tap_prompt"`
	Status        string `json:"status"`
}

type IgAuthenticateCred struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func IgAuthenticate(c *AuthorizedClient, cred IgAuthenticateCred) (*IgAuthenticateResponse, error) {
	data := url.Values{}
	data.Set("login", cred.Login)
	data.Set("password", cred.Password)
	data.Set("enc_password", cred.Password)
	data.Set("queryParams", `{"source":"auth_switcher"}`)
	data.Set("optIntoOneTap", `false`)

	req, err := http.NewRequest("POST", IgAuthenticateUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	c.SetHeaders(req.Header, "https://www.instagram.com/accounts/login/?source=auth_switcher")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Ig-Www-Claim", "0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var authResp IgAuthenticateResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}
