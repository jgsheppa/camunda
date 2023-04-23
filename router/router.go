package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jgsheppa/camunda/task_runner"
)

func ServeHTTP(client task_runner.Camunda) {
	r := chi.NewRouter()
	r.Get("/start", func(w http.ResponseWriter, r *http.Request) {
		process := client.CreateProcess()
		client.RunJobWorker()
		w.Write([]byte(process))
	})
	http.ListenAndServe(":3000", r)
}
