package iglocparser

import (
	"encoding/json"
	"github.com/ansel1/merry"
)

type Location struct {
	Id   string
	Name string
	Slug string
}

type igApiLocationsResponse struct {
	LocationList      []*Location `json:"location_list,omitempty"`
	CityDirectoryPage bool        `json:"city_directory_page,omitempty"`
	CityInfo          *City       `json:"city_info,omitempty"`
	CountryInfo       *Country    `json:"country_info,omitempty"`
	NextPage          *int        `json:"next_page,omitempty"`
	Status            string      `json:"status,omitempty"`
}

type IgApiLocationsCursor struct {
	*igApiCursor

	city *City
}

func (self *IgApiLocationsCursor) Has() bool {
	return self.hasNextPage
}

func (self *IgApiLocationsCursor) Next() ([]*Location, error) {
	link := getIgLinkWithLeadingSlash(IgExploreLocationsPath, self.city.Id)
	referrer := getIgLinkWithLeadingSlash(IgExploreLocationsPath, self.city.Id, self.city.Slug)

	var locations []*Location

	body, err := self.client.do(link, self.nextPage, referrer)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	res := &igApiLocationsResponse{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, merry.Wrap(err)
	}

	if res.Status != "ok" {
		return nil, merry.Errorf("invalid ig api locations response code: '%v'", string(body))
	}

	for _, l := range res.LocationList {
		locations = append(locations, l)
	}

	self.setNextPage(res.NextPage)
	return locations, nil
}

func GetLocationsCursors(client *IgApiClient, city *City) *IgApiLocationsCursor {
	return &IgApiLocationsCursor{
		igApiCursor: &igApiCursor{
			client:      client,
			nextPage:    1,
			hasNextPage: true,
		},
		city: city,
	}
}

func ParseAllLocations(client *IgApiClient, city *City, callback func(page int, locations []*Location)) ([]*Location, error) {
	var locations []*Location

	cursor := GetLocationsCursors(client, city)

	for cursor.Has() {
		page := cursor.nextPage
		list, err := cursor.Next()
		if err != nil {
			return nil, merry.Wrap(err)
		}

		if callback != nil {
			callback(page, list)
		}

		locations = append(locations, list...)
	}

	return locations, nil
}
