package scheduler

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type jobType int

type scheduled interface {
	nextRun() (time.Duration, error)
}

type Job struct {
	Fn       func()
	Quit     chan bool
	schedule scheduled
}

type recurrent struct {
	units  time.Duration
	period time.Duration
}

func (r recurrent) nextRun() (time.Duration, error) {
	if r.units == 0 || r.period == 0 {
		return 0, errors.New("Cannot schedule with 0 time")
	}
	return r.units * r.period, nil
}

type daily struct {
	hour int
	min  int
	sec  int
}

func (d daily) nextRun() (time.Duration, error) {
	now := time.Now()
	year, month, day := now.Date()
	date := time.Date(year, month, day, d.hour, d.min, d.sec, 0, time.Local)
	if now.Before(date) {
		return date.Sub(now), nil
	}
	date = time.Date(year, month, day+1, d.hour, d.min, d.sec, 0, time.Local)
	return date.Sub(now), nil
}

func EverySeconds(t time.Duration) *Job {
	return &Job{schedule: recurrent{units: t, period: time.Second}}
}

func EveryDay() *Job {
	return &Job{schedule: daily{}}
}

func (j *Job) At(hourTime string) *Job {
	d := j.schedule.(daily)
	chunks := strings.Split(hourTime, ":")
	switch len(chunks) {
	case 1:
		d.hour, _ = strconv.Atoi(chunks[0])
	case 2:
		d.hour, _ = strconv.Atoi(chunks[0])
		d.min, _ = strconv.Atoi(chunks[1])
	case 3:
		d.hour, _ = strconv.Atoi(chunks[0])
		d.min, _ = strconv.Atoi(chunks[1])
		d.sec, _ = strconv.Atoi(chunks[2])
	}
	j.schedule = d
	return j
}

func (j *Job) Run(f func()) *Job {
	var next time.Duration
	var err error
	j.Quit = make(chan bool, 1)
	j.Fn = f
	go func(j *Job) {
		for {
			next, err = j.schedule.nextRun()
			if err != nil {
				break
			}
			select {
			case <-j.Quit:
				break
			case <-time.After(next):
				go j.Fn()
			}
		}
	}(j)
	return j
}
