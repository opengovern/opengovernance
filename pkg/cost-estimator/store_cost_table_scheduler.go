package cost_estimator

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"io"
	"net/http"
)

func (h *HttpHandler) StoreCostTableJob() {

}

func (h *HttpHandler) HandleStoreAzureCostTable() error {
	id, err := h.db.CreateStoreCostTableJob(source.CloudAzure)
	if err != nil {
		h.logger.Error("Unable to create job", zap.Error(err))
		return err
	}
	count, err := h.StoreAzureCostTable()
	if err != nil {
		err = h.db.UpdateStoreCostTableJob(id, db.StoreCostTableJobStatusFailed, err.Error(), count)
		if err != nil {
			h.logger.Error("Unable to update job", zap.Error(err))
			return err
		}
	}
	err = h.db.UpdateStoreCostTableJob(id, db.StoreCostTableJobStatusSucceeded, "", count)
	if err != nil {
		h.logger.Error("Unable to update job", zap.Error(err))
		return err
	}
	return nil
}

func (h *HttpHandler) StoreAzureCostTable() (int64, error) {
	hasPage := true
	nextPage := "https://prices.azure.com/api/retail/prices"
	var count int64
	for hasPage {
		req, err := http.NewRequest("GET", nextPage, nil)
		if err != nil {
			h.logger.Error(fmt.Sprintf("error in request to azure for giving the cost : %v ", err))
			return count, fmt.Errorf("error in request to azure for giving the cost : %v ", err)
		}

		client := http.Client{}
		res, err := client.Do(req)
		if err != nil {
			h.logger.Error(fmt.Sprintf("error in sending the request : %v ", err))
			return count, fmt.Errorf("error in sending the request : %v ", err)
		}

		if res.StatusCode != http.StatusOK {
			h.logger.Error(fmt.Sprintf("error status equal to : %v ", res.StatusCode))
			return count, fmt.Errorf("error status equal to : %v ", res.StatusCode)
		}

		responseBody, err := io.ReadAll(res.Body)
		if err != nil {
			h.logger.Error(fmt.Sprintf("error in read the response : %v ", err))
			return count, fmt.Errorf("error in read the response : %v ", err)
		}
		err = res.Body.Close()
		if err != nil {
			h.logger.Error(fmt.Sprintf("error in closing the respponse : %v ", err))
			return count, fmt.Errorf("error in closing the respponse : %v ", err)
		}

		var response es.AzureCostStr
		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			h.logger.Error(fmt.Sprintf("error in unmarshalling the response : %v ", err))
			return count, fmt.Errorf("error in unmarshalling the response : %v ", err)
		}
		var msgs []kafka.Doc
		for _, i := range response.Items {
			msgs = append(msgs, i)
		}
		i := 0
		for {
			if err := kafka.DoSend(h.kafkaProducer, h.kafkaTopic, -1, msgs, h.logger, nil); err != nil {
				if i > 10 {
					h.logger.Warn("send to kafka",
						zap.String("connector:", "aws"),
						zap.String("error message", err.Error()))
					return count, fmt.Errorf("send to kafka: %w", err)
				} else {
					i++
					continue
				}
			}
			break
		}
		count = count + int64(len(response.Items))
		if response.NextPageLink == nil {
			hasPage = false
		} else {
			nextPage = *response.NextPageLink
		}
	}
	return count, nil
}
