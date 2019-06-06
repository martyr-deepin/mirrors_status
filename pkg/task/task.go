package task

import (
	"errors"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/model/task"
	"sync"
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

func (t *TaskManager) Init() {
	log.Info("Init tasks of global task manager")
	openTasks, err := task.GetOpenTasks()
	if err != nil {
		log.Errorf("Get open tasks found error:%#v", err)
		return
	}
	t.Tasks = make(map[string]*task.Task)
	for _, task := range openTasks {
		t.Locker.Lock()
		t.Tasks[task.Upstream] = &task
		log.Infof("Task:%#v", task)
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
	task.Handle(t.DeleteTask)
	t.Locker.Unlock()
	return nil
}

func (t *TaskManager) DeleteTask(upstream string) {
	if t.TaskExists(upstream) {
		return
	}
	t.Locker.Lock()
	delete(t.Tasks, upstream)
	t.Locker.Unlock()
}

func (t *TaskManager) Execute() {
	log.Info("TASK LOOP")
	log.Infof("Queued tasks:%#v", t.Tasks)

	t.Init()
	if len(t.Tasks) <= 0 {
		log.Info("No task in queue")
	} else {
		for _, tk := range t.Tasks {
			log.Infof("%#v", t)
			tk.Handle(t.DeleteTask)
			tsk, _ := task.GetTaskById(tk.Id)
			log.Infof("%#v", tsk)
			if !tsk.IsOpen {
				t.DeleteTask(tk.Upstream)
				break
			}
		}
	}
}