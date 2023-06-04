package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var (
	defaultIo2Thresholds              = []int{32000, 64000}
	defaultGp3FreeIOPSThreshold       = 3000
	defaultGp3FreeThroughputThreshold = 125
)

type EBSCostDescription struct {
	Region        string `json:"region"`
	Gp2Size       int    `json:"gp2Size"`
	Gp3Size       int    `json:"gp3Size"`
	Gp3Throughput int    `json:"gp3Throughput"`
	Gp3IOPS       int    `json:"gp3IOPS"`
	Io1Size       int    `json:"io1Size"`
	Io1IOPS       int    `json:"io1IOPS"`
	Io2Size       int    `json:"io2Size"`
	Io2IOPS       int    `json:"io2IOPS"`
	Sc1Size       int    `json:"sc1Size"`
	St1Size       int    `json:"st1Size"`
	StandardSize  int    `json:"standardSize"`
	StandardIOPS  int    `json:"standardIOPS"`

	CostValue float64 `json:"costValue"`
}

func (e EBSCostDescription) CalculateCostFromPriceJSON() float64 {
	const GBtoMBRatio = 1024
	costManifest, ok := GetEbsCosts()[strings.ToLower(e.Region)]
	if !ok {
		return 0
	}
	total := float64(0)

	// GP2
	total += costManifest.Gp2.PricePerGBMonth.GetPerDayFloat64() * float64(e.Gp2Size)

	// GP3
	total += costManifest.Gp3.PricePerGBMonth.GetPerDayFloat64() * float64(e.Gp3Size)
	total += costManifest.Gp3.PricePerIOPSMonth.GetPerDayFloat64() * float64(e.Gp3IOPS)
	total += costManifest.Gp3.PricePerGiBpsMonth.GetPerDayFloat64() / GBtoMBRatio * float64(e.Gp3Throughput)

	//Io1
	total += costManifest.Io1.PricePerGBMonth.GetPerDayFloat64() * float64(e.Io1Size)
	total += costManifest.Io1.PricePerIOPSMonth.GetPerDayFloat64() * float64(e.Io1IOPS)

	//Io2
	total += costManifest.Io2.PricePerGBMonth.GetPerDayFloat64() * float64(e.Io2Size)
	switch {
	case e.Io2IOPS <= costManifest.Io2.PricePerTierThresholds[0]:
		total += costManifest.Io2.PricePerTier1IOPSMonth.GetPerDayFloat64() * float64(e.Io2IOPS)
	case costManifest.Io2.PricePerTierThresholds[0] < e.Io2IOPS && e.Io2IOPS <= costManifest.Io2.PricePerTierThresholds[1]:
		total += costManifest.Io2.PricePerTier2IOPSMonth.GetPerDayFloat64() * float64(e.Io2IOPS)
	case costManifest.Io2.PricePerTierThresholds[1] < e.Io2IOPS:
		total += costManifest.Io2.PricePerTier3IOPSMonth.GetPerDayFloat64() * float64(e.Io2IOPS)
	}

	//Sc1
	total += costManifest.Sc1.PricePerGBMonth.GetPerDayFloat64() * float64(e.Sc1Size)

	//St1
	total += costManifest.St1.PricePerGBMonth.GetPerDayFloat64() * float64(e.St1Size)

	//Standard
	total += costManifest.Standard.PricePerGBMonth.GetPerDayFloat64() * float64(e.StandardSize)
	total += costManifest.Standard.PricePerIOs.GetPerDayFloat64() * float64(e.StandardIOPS)

	return total
}

func (e EBSCostDescription) GetCost() float64 {
	return e.CostValue
}

type PricePerMonth struct {
	USD string `json:"USD"`
}

func (p PricePerMonth) GetFloat64() float64 {
	res, _ := strconv.ParseFloat(p.USD, 64)
	return res
}

func (p PricePerMonth) GetPerDayFloat64() float64 {
	return p.GetFloat64() / 30
}

type EbsCost struct {
	Gp2 struct {
		PricePerGBMonth PricePerMonth `json:"pricePerGBMonth"`
	} `json:"gp2,omitempty"`
	Gp3 struct {
		PricePerGBMonth    PricePerMonth `json:"pricePerGBMonth"`
		PricePerGiBpsMonth PricePerMonth `json:"pricePerGiBpsMonth"`
		PricePerIOPSMonth  PricePerMonth `json:"pricePerIOPSMonth"`

		FreeIOPSThreshold       int `json:"freeIOPSThreshold"`
		FreeThroughputThreshold int `json:"freeThroughputThreshold"`
	} `json:"gp3,omitempty"`
	Io1 struct {
		PricePerGBMonth   PricePerMonth `json:"pricePerGBMonth"`
		PricePerIOPSMonth PricePerMonth `json:"pricePerIOPSMonth"`
	} `json:"io1,omitempty"`
	Io2 struct {
		PricePerGBMonth        PricePerMonth `json:"pricePerGBMonth"`
		PricePerTier1IOPSMonth PricePerMonth `json:"pricePerTier1IOPSMonth"`
		PricePerTier2IOPSMonth PricePerMonth `json:"pricePerTier2IOPSMonth"`
		PricePerTier3IOPSMonth PricePerMonth `json:"pricePerTier3IOPSMonth"`

		PricePerTierThresholds []int `json:"pricePerTierThresholds"`
	} `json:"io2,omitempty"`
	Sc1 struct {
		PricePerGBMonth PricePerMonth `json:"pricePerGBMonth"`
	} `json:"sc1,omitempty"`
	St1 struct {
		PricePerGBMonth PricePerMonth `json:"pricePerGBMonth"`
	} `json:"st1,omitempty"`
	Standard struct {
		PricePerGBMonth PricePerMonth `json:"pricePerGBMonth"`
		PricePerIOs     PricePerMonth `json:"pricePerIOs"`
	} `json:"standard,omitempty"`
}

type JSONEbsCosts struct {
	EbsPrices EbsCost `json:"ebs_prices"`
	Location  string  `json:"location"`
	Partition string  `json:"partition"`
	RzCode    string  `json:"rzCode"`
	RzType    string  `json:"rzType"`
}

// RegionCode to EBS cost map
var ebsCosts = map[string]EbsCost{}

// Singleton pattern for getting ebsCosts
func GetEbsCosts() map[string]EbsCost {
	if len(ebsCosts) == 0 {
		err := initEbsCosts()
		if err != nil {
			fmt.Printf("Error initializing EBS costs: %v", err)
		}
	}
	return ebsCosts
}

func initEbsCosts() error {
	// read from file
	jsonFile, err := os.Open("/config/ebs-costs.json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	jsonBytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	costsArr := make([]JSONEbsCosts, 0)
	err = json.Unmarshal(jsonBytes, &costsArr)
	if err != nil {
		return err
	}
	for _, cost := range costsArr {
		if cost.EbsPrices.Io2.PricePerTierThresholds == nil {
			cost.EbsPrices.Io2.PricePerTierThresholds = defaultIo2Thresholds
		}
		if cost.EbsPrices.Gp3.FreeIOPSThreshold == 0 {
			cost.EbsPrices.Gp3.FreeIOPSThreshold = defaultGp3FreeIOPSThreshold
		}
		if cost.EbsPrices.Gp3.FreeThroughputThreshold == 0 {
			cost.EbsPrices.Gp3.FreeThroughputThreshold = defaultGp3FreeThroughputThreshold
		}
		ebsCosts[strings.ToLower(cost.RzCode)] = cost.EbsPrices
	}
	return nil
}
