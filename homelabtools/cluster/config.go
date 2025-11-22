package cluster

import (
	"errors"
	"os"

	"github.com/failuretoload/homelabtools/vault"
)

type Config struct {
	clusterName  string
	controlPlane NodeConfig
	workers      []NodeConfig
	secrets      vault.ClusterSecrets
}

func NewConfig(clusterName string, cp NodeConfig, s vault.ClusterSecrets, w ...NodeConfig) (Config, error) {
	cc := Config{
		clusterName:  clusterName,
		controlPlane: cp,
		workers:      w,
		secrets:      s,
	}

	return cc, cc.Validate()
}

func (c Config) Validate() error {
	var err error
	if c.clusterName == "" {
		err = errors.Join(err, errors.New("cluster name is required"))
	}

	if validateErr := c.secrets.Validate(); validateErr != nil {
		err = errors.Join(err, validateErr)
	}

	if cpErr := c.controlPlane.Validate(); cpErr != nil {
		err = errors.Join(err, cpErr)
	}

	for i, worker := range c.workers {
		if wErr := worker.Validate(); wErr != nil {
			err = errors.Join(err, wErr)
		}
		if worker.Address == c.controlPlane.Address {
			err = errors.Join(err, errors.New("worker and control plane cannot have the same address"))
		}
		if worker.HostName == c.controlPlane.HostName {
			err = errors.Join(err, errors.New("worker and control plane cannot have the same hostname"))
		}
		for j := range i {
			if worker.Address == c.workers[j].Address {
				err = errors.Join(err, errors.New("duplicate worker node address"))
			}
			if worker.HostName == c.workers[j].HostName {
				err = errors.Join(err, errors.New("duplicate worker node hostname"))
			}
		}
	}

	return err
}

func (c Config) GenerateConfigs(folderPath string) error {
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		return err
	}

	cpPath := folderPath + "/" + c.clusterName + "-" + c.controlPlane.HostName + "-controlplane.yaml"
	if err := c.generateControlPlaneYAML(cpPath); err != nil {
		return err
	}

	for _, worker := range c.workers {
		workerPath := folderPath + "/" + c.clusterName + "-" + worker.HostName + "-worker.yaml"
		if err := c.generateWorkerYAML(workerPath, worker); err != nil {
			return err
		}
	}

	talosconfigPath := folderPath + "/config"
	if err := c.generateTalosconfig(talosconfigPath); err != nil {
		return err
	}

	return nil
}

type StorageType string

const (
	StorageTypeMMC  StorageType = "mmc"
	StorageTypeNVMe StorageType = "nvme"
)

type NodeConfig struct {
	HostName     string
	Address      string
	StorageType  StorageType
	EphemeralGB  int
	PersistentGB int
}

func (n NodeConfig) Validate() error {
	var err error
	if n.HostName == "" {
		err = errors.Join(err, errors.New("host name is required"))
	}
	if n.Address == "" {
		err = errors.Join(err, errors.New("node address is required"))
	}
	if n.StorageType != StorageTypeMMC && n.StorageType != StorageTypeNVMe {
		err = errors.Join(err, errors.New("storage type must be either mmc or nvme"))
	}
	if n.EphemeralGB < 0 {
		err = errors.Join(err, errors.New("ephemeral volume size cannot be negative"))
	}
	if n.PersistentGB < 0 {
		err = errors.Join(err, errors.New("persistent volume size cannot be negative"))
	}

	return err
}

func NewNodeConfig(h, a string, st StorageType, e, p int) (NodeConfig, error) {
	nc := NodeConfig{
		HostName:     h,
		Address:      a,
		StorageType:  st,
		EphemeralGB:  e,
		PersistentGB: p,
	}

	return nc, nc.Validate()
}
