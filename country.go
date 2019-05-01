package iglocparser

import (
	"encoding/json"
	"github.com/ansel1/merry"
)

type Country struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type igApiCountriesResponse struct {
	CountryList []*Country `json:"country_list,omitempty"`
	NextPage    *int       `json:"next_page,omitempty"`
	Status      string     `json:"status,omitempty"`
}

type IgApiCountriesCursor struct {
	*igApiCursor
}

func (self *IgApiCountriesCursor) Next() ([]*Country, error) {
	body, err := self.client.do(getIgLinkWithLeadingSlash(IgExploreLocationsPath), self.nextPage, getIgLinkWithLeadingSlash(IgExploreLocationsPath))
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

func GetCountriesCursor(client *IgApiClient) *IgApiCountriesCursor {
	return &IgApiCountriesCursor{
		igApiCursor: &igApiCursor{
			client:      client,
			nextPage:    1,
			hasNextPage: true,
		},
	}
}

func ParseAllCountries(client *IgApiClient, callback func(page int, countries []*Country)) ([]*Country, error) {
	var countries []*Country

	cursor := GetCountriesCursor(client)

	for cursor.Has() {
		page := cursor.nextPage
		list, err := cursor.Next()
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
