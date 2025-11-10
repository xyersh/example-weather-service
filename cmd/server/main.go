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
	city     = "moscow"
)

type Reading struct {
	Timestamp  time.Time
	Temprature float64
}

type Storage struct {
	data map[string][]Reading
	mu   sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		data: make(map[string][]Reading, 1000),
	}
}

func main() {
	wg := sync.WaitGroup{}

	storage := NewStorage()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/{city}", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("storage	.data: %v\n", storage.data)
		cityName := chi.URLParam(r, "city")
		log.Printf("city is %s\n", cityName)

		storage.mu.RLock()
		reading, ok := storage.data[cityName]
		log.Printf("reading = %v\n", reading)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}
		storage.mu.RUnlock()

		// маршалим ответ в байтовый срез
		raw, err := json.Marshal(reading)
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

	_, err = initJobs(shed, storage)
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

func initJobs(shed gocron.Scheduler, storage *Storage) ([]gocron.Job, error) {
	log.Println("initJobs - START")
	// add a job to the scheduler
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	geocodingClient := geocoding.NewClient(httpClient)
	openMeteoClient := open_meteo.NewClient(httpClient)

	j, err := shed.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			func() {

				// находим координаты переданного в запросе города
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

				// сохранение данных по расписанию в in-memoty
				storage.mu.Lock()
				defer storage.mu.Unlock()
				timeStamp, err := time.Parse("2006-01-02T15:04", openMeteoRes.Current.Time)
				if err != nil {
					log.Println(err)
					return
				}

				storage.data[city] = append(storage.data[city], Reading{
					// "time": "2025-11-09T06:00",
					Timestamp:  timeStamp,
					Temprature: openMeteoRes.Current.Temperature2m,
				})
				log.Printf("in cache: %v\n", storage.data)
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
