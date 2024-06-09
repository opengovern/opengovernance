package model

import (
	"google.golang.org/api/compute/v1"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type GCPComputeMachineType struct {
	gorm.Model

	// Basic fields
	Id            string `gorm:"index"`
	Name          string `gorm:"index"`
	MachineType   string `gorm:"index"`
	MachineFamily string `gorm:"index"`

	GuestCpus    int64
	MemoryMb     int64
	ImageSpaceGb int64
	Description  string
	Zone         string
}

func (p *GCPComputeMachineType) PopulateFromObject(machineType compute.MachineType) {
	p.Id = strconv.FormatUint(machineType.Id, 10)
	p.Name = machineType.Name
	p.MachineType = machineType.Name
	mf := strings.ToLower(strings.Split(machineType.Name, "-")[0])
	p.MachineFamily = mf
	p.GuestCpus = machineType.GuestCpus
	p.MemoryMb = machineType.MemoryMb
	p.ImageSpaceGb = machineType.ImageSpaceGb
	p.Description = machineType.Description
	p.Zone = machineType.Zone
}
