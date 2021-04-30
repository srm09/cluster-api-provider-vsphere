package helpers

import (
	"crypto/tls"
	"github.com/vmware/govmomi/simulator"
)

type VCSimulator struct {
	// Mocks for VC interactions
	Model  *simulator.Model
	Server *simulator.Server
}

func InitVCSim() (VCSimulator, error) {
	model := simulator.VPX()
	err := model.Create()
	if err != nil {
		return VCSimulator{}, err
	}

	model.Service.TLS = new(tls.Config)
	server := model.Service.NewServer()

	return VCSimulator{
		Model: model,
		Server: server,
	}, nil
}

func (vc VCSimulator) Username() string {
	return vc.Server.URL.User.Username()
}

func (vc VCSimulator) Password() string {
	pwd, _ := vc.Server.URL.User.Password()
	return pwd
}

func (vc VCSimulator) Stop() {
	vc.Server.Close()
	vc.Model.Remove()
}
