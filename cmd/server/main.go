package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
	"github.com/xyersh/example-weather-service/cmd/server/internal/client/http/geocoding"
	"github.com/xyersh/example-weather-service/cmd/server/internal/client/http/open_meteo"
)

const (
	httpPort = ":3000"
)

func main() {
	wg := sync.WaitGroup{}

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	geocodingClient := geocoding.NewClient(httpClient)
	openMeteoClient := open_meteo.NewClient(httpClient)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/{city}", func(w http.ResponseWriter, r *http.Request) {

		city := chi.URLParam(r, "city")
		log.Printf("city is %s", city)

		// находим коордынаты переданного в запросе города
		geoRes, err := geocodingClient.GetCoords(city)
		if err != nil {
			log.Println(err)
			return
		}

		// по найденым координатам получаем погоду из ресурса open_meteo.com
		openMeteoRes, err := openMeteoClient.GetTemperature(geoRes.Latitude, geoRes.Longitude)
		if err != nil {
			log.Println(err)
			return
		}

		// маршалим ответ в байтовый срез
		raw, err := json.Marshal(openMeteoRes)
		if err != nil {
			log.Println(err)
		}

		// передаем срез ответом на запрос
		_, err = w.Write(raw)
		if err != nil {
			log.Println(err)
		}

	})

	shed, err := initCron()
	if err != nil {
		panic(err)
	}

	_, err = initJobs(shed)
	if err != nil {
		panic(err)
	}

	wg.Go(func() {
		err := http.ListenAndServe(httpPort, r)
		if err != nil {
			panic(err)
		}
	})

	wg.Go(func() {
		fmt.Println("sheduler starts")
		shed.Start()
	})

	wg.Wait()
}

func initCron() (gocron.Scheduler, error) {
	log.Println("initCron - START")
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func initJobs(shed gocron.Scheduler) ([]gocron.Job, error) {
	log.Println("initJobs - START")
	// add a job to the scheduler

	j, err := shed.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			func() {
				fmt.Printf("%v hello\n", time.Now())
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return []gocron.Job{j}, nil
}

func runCron(shed gocron.Scheduler) error {
	log.Println("runCron - START")
	// start the scheduler
	shed.Start()

	// block until you are ready to shut down
	<-time.After(time.Minute)

	// when you're done, shut it down
	err := shed.Shutdown()
	if err != nil {
		return err
	}
	return nil
}
