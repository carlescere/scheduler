// Replacement for cron based on:
// http://adam.herokuapp.com/past/2010/4/13/rethinking_cron/
// and
// https://github.com/dbader/schedule
//
// Uses include:
// scheduler.Every(5).Seconds().Run(function)
// scheduler.Every().Day().Run(function)
// scheduler.Every().Sunday().At("08:30").Run(function)
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

// Defines a Job and allows to stop a scheduled job or run it.
type Job struct {
	fn       func()
	Quit     <-chan bool
	SkipWait <-chan bool
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

// Defines when to run a job. For a recurrent jobs (n seconds/minutes/hours) you should // specify the unit and then call to the correspondent period method.
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

// Let's you define a specific time when the job would be run. Does not work with
// recurrent jobs.
// Time should be defined as a string separated by a colon. Could be used as "08:35:30",
// "08:35" or "8" for only the hours.
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

// Set the job to the schedule and returns the pointer to the job so it may be stopped
// or executed without waiting or an error.
func (j *Job) Run(f func()) (*Job, error) {
	if j.err != nil {
		return nil, j.err
	}
	var next time.Duration
	var err error
	j.Quit = make(chan bool, 1)
	j.fn = f
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
			case <-j.SkipWait:
				go j.fn()
			case <-time.After(next):
				go j.fn()
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

// Sets the job to run every Monday.
func (j *Job) Monday() *Job {
	return j.dayOfWeek(time.Monday)
}

// Sets the job to run every Tuesday.
func (j *Job) Tuesday() *Job {
	return j.dayOfWeek(time.Tuesday)
}

// Sets the job to run every Wednesday.
func (j *Job) Wednesday() *Job {
	return j.dayOfWeek(time.Wednesday)
}

// Sets the job to run every Thursday.
func (j *Job) Thursday() *Job {
	return j.dayOfWeek(time.Thursday)
}

// Sets the job to run every Friday.
func (j *Job) Friday() *Job {
	return j.dayOfWeek(time.Friday)
}

// Sets the job to run every Saturday.
func (j *Job) Saturday() *Job {
	return j.dayOfWeek(time.Saturday)
}

// Sets the job to run every Sunday.
func (j *Job) Sunday() *Job {
	return j.dayOfWeek(time.Sunday)
}

// Sets the job to run every day.
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

// Sets the job to run every n Seconds where n was defined in the Every function.
func (j *Job) Seconds() *Job {
	return j.timeOfDay(time.Second)
}

// Sets the job to run every n Minutes where n was defined in the Every function.
func (j *Job) Minutes() *Job {
	return j.timeOfDay(time.Minute)
}

// Sets the job to run every n Hours where n was defined in the Every function.
func (j *Job) Hours() *Job {
	return j.timeOfDay(time.Hour)
}
