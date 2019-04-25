package iglocparser

import (
	"encoding/json"
	"github.com/ansel1/merry"
)

type City struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type igApiCitiesResponse struct {
	CityList             []*City  `json:"city_list,omitempty"`
	CountryDirectoryPage bool     `json:"country_directory_page,omitempty"`
	CountryInfo          *Country `json:"country_info,omitempty"`
	NextPage             *int     `json:"next_page,omitempty"`
	Status               string   `json:"status,omitempty"`
}

type IgApiCitiesCursor struct {
	*igApiCursor

	country *Country
}

func (self *IgApiCitiesCursor) Next() ([]*City, error) {
	link := getIgLinkWithLeadingSlash(IgExploreLocationsPath, self.country.Id)
	referrer := getIgLinkWithLeadingSlash(IgExploreLocationsPath, self.country.Id, self.country.Slug)

	body, err := self.client.do(link, self.nextPage, referrer)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	res := &igApiCitiesResponse{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, merry.Wrap(err)
	}

	if res.Status != "ok" {
		return nil, merry.Errorf("invalid ig api locations response code: '%v'", string(body))
	}

	var cities []*City
	for _, c := range res.CityList {
		cities = append(cities, c)
	}

	self.setNextPage(res.NextPage)
	return cities, nil
}

func GetCitiesCursor(client *IgApiClient, country *Country) *IgApiCitiesCursor {
	return &IgApiCitiesCursor{
		igApiCursor: &igApiCursor{
			client:      client,
			nextPage:    1,
			hasNextPage: true,
		},
		country: country,
	}
}

func ParseAllCities(client *IgApiClient, country *Country, callback func(page int, cities []*City)) ([]*City, error) {
	var cities []*City

	cursor := GetCitiesCursor(client, country)

	for cursor.Has() {
		page := cursor.nextPage
		list, err := cursor.Next()
		if err != nil {
			return nil, merry.Wrap(err)
		}

		if callback != nil {
			callback(page, list)
		}

		cities = append(cities, list...)
	}

	return cities, nil
}
