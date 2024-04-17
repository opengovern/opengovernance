package ingestion

import (
	"encoding/csv"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	dataAgeRepo repo.DataAgeRepo

	ec2InstanceRepo repo.EC2InstanceTypeRepo
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo, dataAgeRepo repo.DataAgeRepo) *Service {
	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
		dataAgeRepo:     dataAgeRepo,
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

		data := make(map[string]string)
		v := model.EC2InstanceType{}
		for col, index := range columns {
			switch col {
			case "vCPU":
				v.VCpu, _ = strconv.ParseInt(row[index], 10, 64)
			case "PricePerUnit":
				v.PricePerUnit, _ = strconv.ParseFloat(row[index], 64)
			case "Instance Type":
				v.InstanceType = row[index]
			case "Network Performance":
				bandwidth, upTo := parseNetworkPerformance(row[index])
				v.NetworkMaxBandwidth = bandwidth
				v.NetworkIsBandwidthUpTo = upTo
			case "Unit":
				v.Unit = row[index]
			case "TermType":
				v.TermType = row[index]
			case "Region Code":
				v.Region = row[index]
			case "Operating System":
				v.OperatingSystem = row[index]
			case "License Model":
				v.LicenseModel = row[index]
			case "usageType":
				v.UsageType = row[index]
			case "Pre Installed S/W":
				v.PreInstalledSW = row[index]
			case "CapacityStatus":
				v.CapacityStatus = row[index]
			case "Tenancy":
				v.Tenancy = row[index]
			}
			data[col] = row[index]
		}

		if v.InstanceType == "" {
			continue
		}
		if v.TermType != "OnDemand" {
			continue
		}
		//if v.PreInstalledSW != "NA" {
		//	continue
		//}
		//if v.Tenancy != "Shared" {
		//	continue
		//}
		//if v.CapacityStatus != "Used" {
		//	continue
		//}

		fmt.Println(v)
		err = s.ec2InstanceRepo.Create(&v)
		if err != nil {

			return err
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

func parseNetworkPerformance(v string) (int64, bool) {
	v = strings.ToLower(v)
	upTo := strings.HasPrefix(v, "up to ")
	v = strings.TrimPrefix(v, "up to ")

	factor := int64(0)
	if strings.HasSuffix(v, "gigabit") {
		factor = 1000000000
		v = strings.TrimSuffix(v, " gigabit")
	} else if strings.HasSuffix(v, "megabit") {
		factor = 1000000
		v = strings.TrimSuffix(v, " megabit")
	}
	b, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return b * factor, upTo
}
