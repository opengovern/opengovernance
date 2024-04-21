package ingestion

import (
	"encoding/csv"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"io"
	"net/http"
	"time"
)

type Service struct {
	dataAgeRepo repo.DataAgeRepo

	ec2InstanceRepo   repo.EC2InstanceTypeRepo
	ebsVolumeTypeRepo repo.EBSVolumeTypeRepo
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, dataAgeRepo repo.DataAgeRepo) *Service {
	return &Service{
		ec2InstanceRepo:   ec2InstanceRepo,
		ebsVolumeTypeRepo: ebsVolumeRepo,
		dataAgeRepo:       dataAgeRepo,
	}
}

func (s *Service) Start() error {
	ticker := time.NewTimer(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		dataAge, err := s.dataAgeRepo.List()
		if err != nil {
			fmt.Println(err)
			continue
		}

		var ec2InstaceData *model.DataAge
		for _, data := range dataAge {
			if data.DataType == "AWS::EC2::Instance" {
				ec2InstaceData = &data
			}
		}

		if ec2InstaceData == nil || ec2InstaceData.UpdatedAt.Before(time.Now().Add(-7*24*time.Hour)) {
			err = s.IngestEc2Instances()
			if err != nil {
				return err
			}
			if ec2InstaceData == nil {
				err = s.dataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			} else {
				err = s.dataAgeRepo.Update("AWS::EC2::Instance", model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) IngestEc2Instances() error {
	url := "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/index.csv"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	csvr := csv.NewReader(resp.Body)
	csvr.FieldsPerRecord = -1

	var columns map[string]int
	for {
		values, err := csvr.Read()
		if err != nil {
			return err
		}

		if len(values) > 2 {
			columns = readColumnPositions(values)
			break
		}
	}

	err = s.ec2InstanceRepo.Truncate()
	if err != nil {
		return err
	}
	// Read through each row in the CSV file and send a price.WithProduct on the results channel.
	for {
		row, err := csvr.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}

		switch row[columns["Product Family"]] {
		case "Compute Instance", "Compute Instance (bare metal)":
			v := model.EC2InstanceType{}
			v.PopulateFromMap(columns, row)

			if v.InstanceType == "" {
				continue
			}
			if v.TermType != "OnDemand" {
				continue
			}

			fmt.Println("Instance", v)
			err = s.ec2InstanceRepo.Create(&v)
			if err != nil {

				return err
			}
		case "Storage", "System Operation", "Provisioned Throughput":
			v := model.EBSVolumeType{}
			v.PopulateFromMap(columns, row)

			if v.VolumeType == "" {
				continue
			}
			if v.TermType != "OnDemand" {
				continue
			}
			fmt.Println("Volume", v)
			err = s.ebsVolumeTypeRepo.Create(&v)
			if err != nil {
				return err
			}
		}
	}
}

// readColumnPositions maps column names to their position in the CSV file.
func readColumnPositions(values []string) map[string]int {
	columns := make(map[string]int)
	for i, v := range values {
		columns[v] = i
	}
	return columns
}
