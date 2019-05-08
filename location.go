package iglocparser

import (
	"encoding/json"
	"fmt"
	"github.com/ansel1/merry"
)

type Location struct {
	Id   string
	Name string
	Slug string
}

func (self *Location) String() string {
	return fmt.Sprintf("[id=%v; name=%v; slug=%v]", self.Id, self.Name, self.Slug)
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

func (self *IgApiLocationsCursor) Next(client *IgApiClient) ([]*Location, error) {
	link := GetIgLinkWithLeadingSlash(IgExploreLocationsPath, self.city.Id)
	referrer := GetIgLinkWithLeadingSlash(IgExploreLocationsPath, self.city.Id, self.city.Slug)

	var locations []*Location

	body, err := client.do(link, self.nextPage, referrer)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	res := &igApiLocationsResponse{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, merry.Wrap(err)
	}

	if res.Status != "ok" {
		return nil, merry.WithUserMessage(ErrInvalidIgApiResponseCode, string(body))
	}

	for _, l := range res.LocationList {
		locations = append(locations, l)
	}

	self.setNextPage(res.NextPage)
	return locations, nil
}

func GetLocationsCursors(city *City) *IgApiLocationsCursor {
	return &IgApiLocationsCursor{
		igApiCursor: &igApiCursor{
			nextPage:    1,
			hasNextPage: true,
		},
		city: city,
	}
}

func ParseAllLocations(client *IgApiClient, city *City, callback func(page int, locations []*Location)) ([]*Location, error) {
	var locations []*Location

	cursor := GetLocationsCursors(city)

	for cursor.Has() {
		page := cursor.nextPage
		list, err := cursor.Next(client)
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
