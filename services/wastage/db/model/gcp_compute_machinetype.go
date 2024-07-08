package model

import (
	"google.golang.org/api/compute/v1"
	"gorm.io/gorm"
	"strings"
)

type GCPComputeMachineType struct {
	gorm.Model

	// Basic fields
	Name          string `gorm:"index"`
	MachineType   string `gorm:"index"`
	MachineFamily string `gorm:"index"`
	Zone          string `gorm:"index"`
	Preemptible   bool   `gorm:"index"`

	GuestCpus    int64
	MemoryMb     int64
	ImageSpaceGb int64
	Description  string
	Region       string

	UnitPrice float64
}

func (p *GCPComputeMachineType) PopulateFromObject(machineType *compute.MachineType, region string, preemptible bool) {
	p.Name = machineType.Name
	p.MachineType = machineType.Name
	mf := strings.ToLower(strings.Split(machineType.Name, "-")[0])
	p.MachineFamily = mf
	p.GuestCpus = machineType.GuestCpus
	p.MemoryMb = machineType.MemoryMb
	p.ImageSpaceGb = machineType.ImageSpaceGb
	p.Description = machineType.Description
	p.Zone = machineType.Zone
	p.Region = region
	p.Preemptible = preemptible
}
