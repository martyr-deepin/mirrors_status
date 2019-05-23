package task

import (
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/service"
	checker2 "mirrors_status/pkg/mirror/checker"
	"mirrors_status/pkg/model"
	"mirrors_status/pkg/task/jenkins"
	"mirrors_status/pkg/utils"
	"time"
)

func TaskHandler() {
	taskService := service.NewTaskService(configs.GetMySQLClient())
	checker := checker2.NewCDNChecker(configs.NewServerConfig().CdnChecker)
	mirrorService := service.NewMirrorService(*configs.GetMySQLClient())
	for {
		log.Info("Start execution loop handler")
		openTasks, err := taskService.GetOpenTasks()
		if err != nil {
			log.Errorf("Get open tasks found error:%#v", err)
			continue
		}
		for _, task := range openTasks {
			var mirrorOperation model.MirrorOperation
			if task.MirrorOperationIndex != "" {
				mirrorOperation, err = service.GetOperationByIndex(task.MirrorOperationIndex)
				if err != nil {
					log.Errorf("Get mirror operation by index:[%s] found error:%#v", task.MirrorOperationIndex, err)
					continue
				}
			}
			if mirrorOperation.Status == model.STATUS_RUNNING {
				continue
			} else if mirrorOperation.Status == model.STATUS_WAITING {
				mirrors, err := mirrorService.GetMirrorsByUpstream(task.Upstream)
				if err != nil {
					if err != nil {
						log.Errorf("Get mirrors by upstream:[%s] found error:%#v", task.Upstream, err)
						continue
					}
				}
				index := checker.CheckMirrors(mirrors, task.Creator)
				err = taskService.UpdateMirrorOperationIndex(task.Id, index)
				if err != nil {
					log.Errorf("Update mirror task index:[%s] found error:%#v", index, err)
					continue
				}
				for {
					mirrorOperation, _ = service.GetOperationByIndex(index)
					if mirrorOperation.Status == model.STATUS_FINISHED {
						break
					}
					time.Sleep(time.Second * 10)
					continue
				}
			}
			ciTasks, err := taskService.GetCiTasksById(task.Id)
			if err != nil {
				log.Errorf("Get ci tasks found error:%#v", err)
				continue
			}
			for _, ciTask := range ciTasks {
				jobInfo := &configs.JobInfo{
					Name: ciTask.Name,
					URL: ciTask.JobUrl,
					Token: ciTask.Token,
					Description: ciTask.Description,
				}
				params := make(map[string]string)
				params["UPSTREAM"] = task.Upstream
				params["MIRRORS_API"] = ciTask.JobUrl
				_, err := jenkins.TriggerBuild(jobInfo, params)
				if err != nil {
					utils.SendMail("warning", "<h3>Jenkins task trigger failed. Error messages are as follow:</h3><br><p>" + err.Error() + "</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", task.ContactMail)
					taskService.UpdateTaskStatus(task.Id, model.STATUS_FAILURE)
					continue
				}
				for {
					buildInfo, err := jenkins.LastBuildInfo(jobInfo)
					//buildInfo, err := jenkins.GetBuildInfo(jobInfo, queueId)
					if err != nil {
						utils.SendMail("warning", "<h3>Jenkins task build info fetch got error. Error messages are as follow:</h3><br><p>" + err.Error() + "</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", task.ContactMail)
						taskService.UpdateCiTaskResult(ciTask.Id, "FAILURE")
						break
					}
					err = taskService.UpdateCiTaskJenkinsUrl(ciTask.Id, buildInfo.URL)
					if err != nil {
						log.Errorf("Update ci task jenkins url found error:%#v", err)
						break
					}
					if buildInfo.Result == "SUCCESS" {
						err = taskService.UpdateCiTaskResult(ciTask.Id, buildInfo.Result)
						if err != nil {
							log.Errorf("Update ci task result url found error:%#v", err)
							break
						}
						err = taskService.TaskProceed(ciTask.TaskId)
						if err != nil {
							log.Errorf("Update task progress url found error:%#v", err)
							break
						}
						err = taskService.UpdateMirrorOperationStatus(ciTask.TaskId, model.STATUS_WAITING)
						if err != nil {
							log.Errorf("Update mirror operation status url found error:%#v", err)
							break
						}
						delay := configs.NewServerConfig().Jenkins.Delay
						time.Sleep(time.Minute * time.Duration(delay))
						break
					} else {
						time.Sleep(time.Second * 10)
						continue
					}
				}

			}
		}
		time.Sleep(time.Second * 10)
	}
}

func Execute() {
	go TaskHandler()
}