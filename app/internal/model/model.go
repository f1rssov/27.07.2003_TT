package model

import(
	"sync"
)

type FileURLRequest struct {
	Url string `json:"file_url"`
}

type TaskStatus string

const (
	StatusP		TaskStatus = "pending"
	StatusR		TaskStatus = "running"
	StatusC		TaskStatus = "completed"
	StatusE    	TaskStatus = "error"
)


type Task struct {
	TaskName	string		`json:"task_name"`
	Links    	[]string	`json:"links"`
	Archive  	string		`json:"archive"`
	Status   	TaskStatus	`json:"status"`
	Errors   	[]string	`json:"errors"`
	Count		int			`json:"count_links"`
	TaskMutex 	sync.Mutex	`json:"-"`
}
