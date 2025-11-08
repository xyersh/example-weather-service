package open_meteo

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Current struct {
		Temperature2m float64 `json:"temperature_2m"`
		Time          string  `json:"time"`
	}
}

type client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *client {
	return &client{
		httpClient: httpClient,
	}
}

func (c *client) GetTemperature(lat float64, lon float64) (Response, error) {
	resp, err := c.httpClient.Get(
		fmt.Sprintf("https://api.open-meteo.com/v1/forecast/?latitude=%f&longitude=%f&current=temperature_2m",
			lat,
			lon,
		),
	)

	if err != nil {
		return Response{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("status code: %d\n", resp.StatusCode)
	}

	defer resp.Body.Close()

	result := Response{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return Response{}, err
	}

	return result, nil
}
