package iglocparser

import (
	"encoding/json"
	"fmt"
	"github.com/ansel1/merry"
)

type Country struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (self *Country) String() string {
	return fmt.Sprintf("[id=%v; name=%v; slug=%v]", self.Id, self.Name, self.Slug)
}

type igApiCountriesResponse struct {
	CountryList []*Country `json:"country_list,omitempty"`
	NextPage    *int       `json:"next_page,omitempty"`
	Status      string     `json:"status,omitempty"`
}

type IgApiCountriesCursor struct {
	*igApiCursor
}

func (self *IgApiCountriesCursor) Next(client *IgApiClient) ([]*Country, error) {
	body, err := client.do(GetIgLinkWithLeadingSlash(IgExploreLocationsPath), self.nextPage, GetIgLinkWithLeadingSlash(IgExploreLocationsPath))
	if err != nil {
		return nil, merry.Wrap(err)
	}

	res := &igApiCountriesResponse{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, merry.Wrap(err)
	}

	if res.Status != "ok" {
		return nil, merry.WithUserMessage(ErrInvalidIgApiResponseCode, string(body))
	}

	self.setNextPage(res.NextPage)
	return res.CountryList, nil
}

func GetCountriesCursor() *IgApiCountriesCursor {
	return &IgApiCountriesCursor{
		igApiCursor: &igApiCursor{
			nextPage:    1,
			hasNextPage: true,
		},
	}
}

func ParseAllCountries(client *IgApiClient, callback func(page int, countries []*Country)) ([]*Country, error) {
	var countries []*Country

	cursor := GetCountriesCursor()

	for cursor.Has() {
		page := cursor.nextPage
		list, err := cursor.Next(client)
		if err != nil {
			return nil, merry.Wrap(err)
		}

		if callback != nil {
			callback(page, list)
		}

		countries = append(countries, list...)
	}

	return countries, nil
}
