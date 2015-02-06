package scheduler

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

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

type weekly struct {
	day time.Weekday
	d   daily
}

func (w weekly) nextRun() (time.Duration, error) {
	now := time.Now()
	year, month, day := now.Date()
	numDays := w.day - now.Weekday()
	if numDays == 0 {
		date := time.Date(year, month, day, w.d.hour, w.d.min, w.d.sec, 0, time.Local)
		if now.Before(date) {
			return date.Sub(now), nil
		}
		numDays = 7
	} else if numDays < 0 {
		numDays += 7
	}
	date := time.Date(year, month, day+int(numDays), w.d.hour, w.d.min, w.d.sec, 0, time.Local)
	return date.Sub(now), nil
}

func Every(t time.Duration) *Job {
	return &Job{schedule: recurrent{units: t}}
}

func (j *Job) Minutes() *Job {
	r := j.schedule.(recurrent)
	r.period = time.Minute
	j.schedule = r
	return j
}

func (j *Job) Hours() *Job {
	r := j.schedule.(recurrent)
	r.period = time.Hour
	j.schedule = r
	return j
}

func (j *Job) Seconds() *Job {
	r := j.schedule.(recurrent)
	r.period = time.Second
	j.schedule = r
	return j
}

func EveryDay() *Job {
	return &Job{schedule: daily{}}
}

func EveryMonday() *Job {
	return &Job{schedule: weekly{day: time.Monday}}
}

func (j *Job) At(hourTime string) *Job {
	var d *daily
	aux, ok := j.schedule.(daily)
	if !ok {
		w := j.schedule.(weekly)
		d = &w.d
	} else {
		d = &aux
	}
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
