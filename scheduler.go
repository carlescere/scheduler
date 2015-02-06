package scheduler

import (
	"errors"
	"log"
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
	err      error
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

func (d *daily) setTime(h, m, s int) {
	d.hour = h
	d.min = m
	d.sec = s
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

func Every(times ...time.Duration) *Job {
	switch len(times) {
	case 0:
		return &Job{}
	case 1:
		return &Job{schedule: recurrent{units: times[0]}}
	default:
		return &Job{err: errors.New("Too many arguments in Every")}
	}
}

func (j *Job) At(hourTime string) *Job {
	if j.err != nil {
		return j
	}
	hour, min, sec := parseTime(hourTime)
	d, ok := j.schedule.(daily)
	if !ok {
		w, ok := j.schedule.(weekly)
		if !ok {
			j.err = errors.New("Bad function chaining")
			return j
		}
		w.d.setTime(hour, min, sec)
		j.schedule = w
	} else {
		d.setTime(hour, min, sec)
		j.schedule = d
	}
	return j
}

func (j *Job) Run(f func()) (*Job, error) {
	if j.err != nil {
		return nil, j.err
	}
	var next time.Duration
	var err error
	j.Quit = make(chan bool, 1)
	j.Fn = f
	go func(j *Job) {
		for {
			next, err = j.schedule.nextRun()
			log.Printf("Next: %v\n", next)
			if err != nil {
				log.Printf("Job schedule error: %v\n", err)
				return
			}
			select {
			case <-j.Quit:
				return
			case <-time.After(next):
				go j.Fn()
			}
		}
	}(j)
	return j, nil
}

func parseTime(str string) (hour, min, sec int) {
	chunks := strings.Split(str, ":")
	switch len(chunks) {
	case 1:
		hour, _ = strconv.Atoi(chunks[0])
	case 2:
		hour, _ = strconv.Atoi(chunks[0])
		min, _ = strconv.Atoi(chunks[1])
	case 3:
		hour, _ = strconv.Atoi(chunks[0])
		min, _ = strconv.Atoi(chunks[1])
		sec, _ = strconv.Atoi(chunks[2])
	}

	return
}

func (j *Job) dayOfWeek(d time.Weekday) *Job {
	if j.schedule != nil {
		j.err = errors.New("Bad function chaining")
	}
	j.schedule = weekly{day: time.Monday}
	return j
}

func (j *Job) Monday() *Job {
	return j.dayOfWeek(time.Monday)
}

func (j *Job) Tuesday() *Job {
	return j.dayOfWeek(time.Tuesday)
}

func (j *Job) Wednesday() *Job {
	return j.dayOfWeek(time.Wednesday)
}

func (j *Job) Thursday() *Job {
	return j.dayOfWeek(time.Thursday)
}

func (j *Job) Friday() *Job {
	return j.dayOfWeek(time.Friday)
}

func (j *Job) Saturday() *Job {
	return j.dayOfWeek(time.Saturday)
}

func (j *Job) Sunday() *Job {
	return j.dayOfWeek(time.Sunday)
}

func (j *Job) Day() *Job {
	if j.schedule != nil {
		j.err = errors.New("Bad function chaining")
	}
	j.schedule = daily{}
	return j
}

func (j *Job) timeOfDay(d time.Duration) *Job {
	if j.err != nil {
		return j
	}
	r := j.schedule.(recurrent)
	r.period = d
	j.schedule = r
	return j
}

func (j *Job) Seconds() *Job {
	return j.timeOfDay(time.Second)
}

func (j *Job) Minutes() *Job {
	return j.timeOfDay(time.Minute)
}

func (j *Job) Hours() *Job {
	return j.timeOfDay(time.Hour)
}
