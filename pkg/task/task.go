package task

import (
	"errors"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/model/task"
	"sync"
	"time"
)

type TaskManager struct {
	Tasks map[string]*task.Task
	Locker sync.Mutex
}

var taskManager *TaskManager

func NewTaskManager() {
	taskManager = &TaskManager{
		Tasks: make(map[string]*task.Task),
	}
	go taskManager.Execute()
}

func GetTaskManager() *TaskManager {
	return taskManager
}

func (t *TaskManager) Init() {
	openTasks, err := task.GetOpenTasks()
	if err != nil {
		log.Errorf("Get open tasks found error:%#v", err)
		return
	}
	for _, task := range openTasks {
		t.Locker.Lock()
		t.Tasks[task.Upstream] = &task
		t.Locker.Unlock()
	}
}

func (t *TaskManager) TaskExists(upstream string) bool {
	return !(t.Tasks[upstream] == nil)
}

func (t *TaskManager) InsertTask(task task.Task) error {
	if t.TaskExists(task.Upstream) {
		return errors.New("task already exists")
	}
	t.Locker.Lock()
	t.Tasks[task.Upstream] = &task
	t.Locker.Unlock()
	return nil
}

func (t *TaskManager) DeleteTask(upstream string) error {
	if t.TaskExists(upstream) {
		return errors.New("task already removed")
	}
	t.Locker.Lock()
	delete(t.Tasks, upstream)
	t.Locker.Unlock()
	return nil
}

func (t *TaskManager) Execute() {
	for {
		log.Info("TASK LOOP")
		log.Infof("Queued tasks:%#v", t.Tasks)

		t.Init()
		for _, task := range t.Tasks {
			task.Handle()
		}

		time.Sleep(time.Second * 10)
	}
}
