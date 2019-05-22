package service

import "mirrors_status/pkg/db/client/mysql"

type TaskService struct {
	client *mysql.Client
}

func NewTaskService(client *mysql.Client) *TaskService {
	return &TaskService{
		client: client,
	}
}

