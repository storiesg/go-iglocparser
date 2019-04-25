package iglocparser

import (
	"encoding/json"
	"github.com/ansel1/merry"
	"github.com/buger/jsonparser"
	"io/ioutil"
	"net/http"
	"regexp"
)

type PlaceAddress struct {
	StreetAddress string `json:"street_address,omitempty"`
	ZipCode       string `json:"zip_code,omitempty"`
	CityName      string `json:"city_name,omitempty"`
	RegionName    string `json:"region_name,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
}

type Place struct {
	Id               string  `json:"id,omitempty"`
	Name             string  `json:"name,omitempty"`
	Latitude         float64 `json:"lat,omitempty"`
	Longitude        float64 `json:"lng,omitempty"`
	Slug             string  `json:"slug,omitempty"`
	Blurb            string  `json:"blurb,omitempty"`
	Website          string  `json:"website,omitempty"`
	Phone            string  `json:"phone,omitempty"`
	PrimaryAliasOnFb string  `json:"primary_alias_on_fb,omitempty"`
	ProfilePicUrl    string  `json:"profile_pic_url,omitempty"`

	Address          PlaceAddress

	Country *Country `json:"country"`
	City    *City    `json:"city"`
}

func ParsePlace(client *Client, id string, referrer string) (*Place, error) {
	link := getIgLinkWithLeadingSlash(IgExploreLocationsPath, id)

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	client.SetHeaders(req.Header, referrer)
	res, err := client.Do(req)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, merry.Wrap(ErrUndefinedLocation)
	} else if res.StatusCode != http.StatusOK {
		return nil, merry.WithHTTPCode(ErrInvalidResponseStatus, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	place, err := getPlaceInfoFromPageBody(body)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return place, nil
}

var regexPlaceJsonSharedDataFinder = regexp.MustCompile(`<script type="text/javascript">window\._sharedData = (.*);</script>`)

type placeLocationJsonResponse struct {
	Place
	AddressJson string `json:"address_json,omitempty"`
	Directory   *struct {
		Country *Country `json:"country,omitempty"`
		City    *City    `json:"city,omitempty"`
	} `json:"directory,omitempty"`
}

func getPlaceInfoFromPageBody(body []byte) (*Place, error) {
	jsonSharedDataMatches := regexPlaceJsonSharedDataFinder.FindSubmatch(body)
	if len(jsonSharedDataMatches) < 2 {
		return nil, merry.New("failed to find sharedData json in body")
	}

	jsonSharedData := jsonSharedDataMatches[1]

	var lastErr error
	var firstValue []byte
	_, err := jsonparser.ArrayEach(jsonSharedData, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if lastErr != nil || firstValue != nil {
			return
		}

		if err != nil {
			lastErr = err
		}

		firstValue = value
	}, "entry_data", "LocationsPage")
	if err != nil {
		return nil, merry.Wrap(err)
	} else if lastErr != nil {
		return nil, merry.Wrap(err)
	} else if firstValue == nil {
		return nil, merry.New("missing LocationsPage from sharedData json")
	}

	jsonLocationData, _, _, err := jsonparser.Get(firstValue, "graphql", "location")
	if err != nil {
		return nil, merry.Wrap(err)
	}

	res := placeLocationJsonResponse{}
	if err := json.Unmarshal(jsonLocationData, &res); err != nil {
		return nil, merry.Wrap(err)
	}

	addr := PlaceAddress{}
	if err := json.Unmarshal([]byte(res.AddressJson), &addr); err != nil {
		return nil, merry.Wrap(err)
	}

	place := &res.Place
	place.Address = addr
	if res.Directory != nil {
		if res.Directory.Country != nil {
			place.Country = res.Directory.Country
		}

		if res.Directory.City != nil {
			place.City = res.Directory.City
		}
	}

	return place, nil
}
