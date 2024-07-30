package models

import "time"

type Mail struct {
	From       string
	Subject    string
	Text       string
	ReceivedAt time.Time
}
