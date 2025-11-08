package geocoding

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type client struct {
	httpClient *http.Client
}

type Response struct {
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NewClient(httpClient *http.Client) *client {
	return &client{
		httpClient: httpClient,
	}
}

func (c *client) GetCoords(city string) (Response, error) {
	log.Printf("GetCoords START with param: %s", city)
	resp, err := c.httpClient.Get(
		fmt.Sprintf(
			"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=ru&format=json",
			city,
		),
	)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("status code: %d\n", resp.StatusCode)

	}
	defer resp.Body.Close()

	var geoResp struct {
		Results []Response `json:"results"`
	}

	err = json.NewDecoder(resp.Body).Decode(&geoResp)
	if err != nil {
		return Response{}, err
	}

	return geoResp.Results[0], nil
}
