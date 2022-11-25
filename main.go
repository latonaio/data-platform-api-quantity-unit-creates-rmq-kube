package main

import (
	dpfm_api_caller "data-platform-api-quantity-unit-creates-rmq-kube/DPFM_API_Caller"
	dpfm_api_input_reader "data-platform-api-quantity-unit-creates-rmq-kube/DPFM_API_Input_Reader"
	"data-platform-api-quantity-unit-creates-rmq-kube/config"
	"encoding/json"
	"fmt"
	"time"

	"github.com/latonaio/golang-logging-library-for-data-platform/logger"
	rabbitmq "github.com/latonaio/rabbitmq-golang-client-for-data-platform"
)

func main() {
	l := logger.NewLogger()
	conf := config.NewConf()
	rmq, err := rabbitmq.NewRabbitmqClient(conf.RMQ.URL(), conf.RMQ.QueueFrom(), conf.RMQ.SessionControlQueue(), conf.RMQ.QueueToSQL(), 0)
	if err != nil {
		l.Fatal(err.Error())
	}
	defer rmq.Close()
	iter, err := rmq.Iterator()
	if err != nil {
		l.Fatal(err.Error())
	}
	defer rmq.Stop()

	caller := dpfm_api_caller.NewDPFMAPICaller(conf, rmq)

	for msg := range iter {
		start := time.Now()
		err = callProcess(rmq, caller, conf, msg)
		if err != nil {
			msg.Fail()
			continue
		}
		msg.Success()
		l.Info("process time %v\n", time.Since(start).Milliseconds())
	}
}
func getSessionID(data map[string]interface{}) string {
	id := fmt.Sprintf("%v", data["runtime_session_id"])
	return id
}

func callProcess(rmq *rabbitmq.RabbitmqClient, caller *dpfm_api_caller.DPFMAPICaller, conf *config.Conf, msg rabbitmq.RabbitmqMessage) (err error) {
	l := logger.NewLogger()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("error occurred: %w", e)
			l.Error(err)
			return
		}
	}()
	l.AddHeaderInfo(map[string]interface{}{"runtime_session_id": getSessionID(msg.Data())})
	var input dpfm_api_input_reader.SDC

	err = json.Unmarshal(msg.Raw(), &input)
	if err != nil {
		l.Error(err)
		input.APIProcessingResult = getBoolPtr(false)
		input.APIProcessingError = err.Error()
		return
	}

	accepter := getAccepter(&input)

	errs := caller.AsyncQuantityUnitCreates(accepter, &input, l)
	if len(errs) != 0 {
		for _, err := range errs {
			l.Error(err)
		}
		input.APIProcessingResult = getBoolPtr(false)
		input.APIProcessingError = errs[0].Error()
		rmq.Send(conf.RMQ.QueueToResponse(), input)
		return errs[0]
	}
	input.APIProcessingResult = getBoolPtr(true)
	rmq.Send(conf.RMQ.QueueToResponse(), input)
	l.JsonParseOut(input)

	return nil
}

func getAccepter(input *dpfm_api_input_reader.SDC) []string {
	accepter := input.Accepter
	if len(input.Accepter) == 0 {
		accepter = []string{"All"}
	}

	if accepter[0] == "All" {
		accepter = []string{
			"QuantityUnit",
		}
	}
	return accepter
}

func getBoolPtr(b bool) *bool {
	return &b
}
