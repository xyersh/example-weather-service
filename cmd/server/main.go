package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
)

const (
	httpPort = ":3000"
)

func main() {
	wg := sync.WaitGroup{}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("welcome"))
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
	fmt.Println("initCron - START")
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func initJobs(shed gocron.Scheduler) ([]gocron.Job, error) {
	fmt.Println("initJobs - START")
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
	fmt.Println("runCron - START")
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
