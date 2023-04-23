package task_runner

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/pb"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/joho/godotenv"
)

type Camunda struct {
	Client zbc.Client
}

func SetUpClient() Camunda {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress: os.Getenv("ZEEBE_ADDRESS"),
	})

	if err != nil {
		panic(err)
	}

	return Camunda{Client: client}
}

func roleToString(role pb.Partition_PartitionBrokerRole) string {
	switch role {
	case pb.Partition_LEADER:
		return "Leader"
	case pb.Partition_FOLLOWER:
		return "Follower"
	default:
		return "Unknown"
	}
}

func (c *Camunda) GetStatus() {
	ctx := context.Background()
	topology, err := c.Client.NewTopologyCommand().Send(ctx)
	if err != nil {
		panic(err)
	}

	for _, broker := range topology.Brokers {
		fmt.Println("Broker", broker.Host, ":", broker.Port)
		for _, partition := range broker.Partitions {
			fmt.Println("  Partition", partition.PartitionId, ":", roleToString(partition.Role))
		}
	}
}

func (c *Camunda) Deploy() {
	ctx := context.Background()
	response, err := c.Client.NewDeployResourceCommand().AddResourceFile("order-process.bpmn").Send(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.String())
}

func (c *Camunda) CreateProcess() string {
	// After the process is deployed.
	variables := make(map[string]interface{})
	variables["orderId"] = "31243"

	request, err := c.Client.NewCreateInstanceCommand().BPMNProcessId("Process_44feff77-e35c-4a6c-9fd5-f1876212f8a5").LatestVersion().VariablesFromMap(variables)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	msg, err := request.Send(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return msg.String()
}

func (c *Camunda) RunJobWorker() {
	jobWorker := c.Client.NewJobWorker().JobType("payment-service").Handler(HandleJob).Open()

	<-readyClose
	jobWorker.Close()
	jobWorker.AwaitClose()
}

var readyClose = make(chan struct{})

func HandleJob(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	headers, err := job.GetCustomHeadersAsMap()
	if err != nil {
		// failed to handle job as we require the custom job headers
		FailJob(client, job)
		return
	}

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		// failed to handle job as we require the variables
		FailJob(client, job)
		return
	}

	variables["totalPrice"] = 46.50
	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		// failed to set the updated variables
		FailJob(client, job)
		return
	}

	log.Println("Complete job", jobKey, "of type", job.Type)
	log.Println("Processing order:", variables["orderId"])
	log.Println("Collect money using payment method:", headers["method"])

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		panic(err)
	}

	log.Println("Successfully completed job")
	close(readyClose)
}

func FailJob(client worker.JobClient, job entities.Job) {
	log.Println("Failed to complete job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send(ctx)
	if err != nil {
		panic(err)
	}
}
