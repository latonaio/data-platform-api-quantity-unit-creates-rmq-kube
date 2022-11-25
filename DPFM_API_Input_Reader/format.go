package dpfm_api_input_reader

import (
	"data-platform-api-quantity-unit-creates-rmq-kube/DPFM_API_Caller/requests"
)

func (sdc *SDC) ConvertToQuantityUnitn() *requests.QuantityUnit {
	data := sdc.QuantityUnit
	return &requests.QuantityUnit{
		QuantityUnit: data.QuantityUnit,
	}
}
