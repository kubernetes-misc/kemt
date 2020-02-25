package model

type Log struct {
	body string
	next *Log
}
