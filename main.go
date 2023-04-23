package main

import (
	"github.com/jgsheppa/camunda/router"
	"github.com/jgsheppa/camunda/task_runner"
)

func main() {
	camunda := task_runner.SetUpClient()
	camunda.Deploy()
	camunda.GetStatus()

	router.ServeHTTP(camunda)
}
