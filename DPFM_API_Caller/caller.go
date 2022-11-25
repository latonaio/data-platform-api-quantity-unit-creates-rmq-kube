package dpfm_api_caller

import (
	"context"
	dpfm_api_input_reader "data-platform-api-quantity-unit-creates-rmq-kube/DPFM_API_Input_Reader"
	"data-platform-api-quantity-unit-creates-rmq-kube/config"
	"sync"
	"time"

	"github.com/latonaio/golang-logging-library-for-data-platform/logger"
	rabbitmq "github.com/latonaio/rabbitmq-golang-client-for-data-platform"
	"golang.org/x/xerrors"
)

type DPFMAPICaller struct {
	ctx  context.Context
	conf *config.Conf
	rmq  *rabbitmq.RabbitmqClient
}

func NewDPFMAPICaller(
	conf *config.Conf, rmq *rabbitmq.RabbitmqClient,

) *DPFMAPICaller {
	return &DPFMAPICaller{
		ctx:  context.Background(),
		conf: conf,
		rmq:  rmq,
	}
}

func (c *DPFMAPICaller) AsyncQuantityUnitCreates(
	accepter []string,
	input *dpfm_api_input_reader.SDC,

	log *logger.Logger,

) []error {
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	errs := make([]error, 0, 5)

	sqlUpdateFin := make(chan error)

	for _, fn := range accepter {
		wg.Add(1)
		switch fn {
		case "QuantityUnit":
			go c.QuantityUnit(&wg, &mtx, sqlUpdateFin, log, &errs, input)
		default:
			wg.Done()
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	select {
	case e := <-sqlUpdateFin:
		if e != nil {
			mtx.Lock()
			errs = append(errs, e)
			return errs
		}
	case <-ticker.C:
		mtx.Lock()
		errs = append(errs, xerrors.New("time out"))
		return errs
	}

	return nil
}

func (c *DPFMAPICaller) QuantityUnit(wg *sync.WaitGroup, mtx *sync.Mutex, errFin chan error, log *logger.Logger, errs *[]error, sdc *dpfm_api_input_reader.SDC) {
	var err error = nil
	defer wg.Done()
	defer func() {
		errFin <- err
	}()
	sessionID := sdc.RuntimeSessionID
	ctx := context.Background()

	quantityUnitData := sdc.QuantityUnit
	res, err := c.rmq.SessionKeepRequest(ctx, c.conf.RMQ.QueueToSQL()[0], map[string]interface{}{"message": quantityUnitData, "function": "QuantityUnitQuantityUnit", "runtime_session_id": sessionID})
	if err != nil {
		err = xerrors.Errorf("rmq error: %w", err)
		return
	}
	res.Success()
	if !checkResult(res) {

		sdc.SQLUpdateResult = getBoolPtr(false)
		sdc.SQLUpdateError = "Quantity Unit Data cannot insert"
		return
	}

	sdc.SQLUpdateResult = getBoolPtr(true)
	return
}

func checkResult(msg rabbitmq.RabbitmqMessage) bool {
	data := msg.Data()
	_, ok := data["result"]
	if !ok {
		return false
	}
	result, ok := data["result"].(string)
	if !ok {
		return false
	}
	return result == "success"

}

func getBoolPtr(b bool) *bool {
	return &b
}
