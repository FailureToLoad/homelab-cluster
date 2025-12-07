package cluster

import (
	"errors"
	"os"
)

type Config struct {
	clusterName          string
	controlPlaneEndpoint string
	controlPlanes        []NodeConfig
	workers              []NodeConfig
	secrets              Secrets
}

func NewConfig(clusterName string, controlPlaneEndpoint string, s Secrets, cp []NodeConfig, w []NodeConfig) (Config, error) {
	cc := Config{
		clusterName:          clusterName,
		controlPlaneEndpoint: controlPlaneEndpoint,
		controlPlanes:        cp,
		workers:              w,
		secrets:              s,
	}

	return cc, cc.Validate()
}

func (c Config) Validate() error {
	var err error
	if c.clusterName == "" {
		err = errors.Join(err, errors.New("cluster name is required"))
	}

	if c.controlPlaneEndpoint == "" {
		err = errors.Join(err, errors.New("control plane endpoint is required"))
	}

	if len(c.controlPlanes) == 0 {
		err = errors.Join(err, errors.New("at least one control plane is required"))
	}

	if validateErr := c.secrets.Validate(); validateErr != nil {
		err = errors.Join(err, validateErr)
	}

	seenAddresses := make(map[string]struct{})
	seenHostnames := make(map[string]struct{})

	for _, cp := range c.controlPlanes {
		if cpErr := cp.Validate(); cpErr != nil {
			err = errors.Join(err, cpErr)
		}
		if _, ok := seenAddresses[cp.Address]; ok {
			err = errors.Join(err, errors.New("duplicate node address"))
		}
		if _, ok := seenHostnames[cp.HostName]; ok {
			err = errors.Join(err, errors.New("duplicate node hostname"))
		}
		seenAddresses[cp.Address] = struct{}{}
		seenHostnames[cp.HostName] = struct{}{}
	}

	for _, worker := range c.workers {
		if wErr := worker.Validate(); wErr != nil {
			err = errors.Join(err, wErr)
		}
		if _, ok := seenAddresses[worker.Address]; ok {
			err = errors.Join(err, errors.New("duplicate node address"))
		}
		if _, ok := seenHostnames[worker.HostName]; ok {
			err = errors.Join(err, errors.New("duplicate node hostname"))
		}
		seenAddresses[worker.Address] = struct{}{}
		seenHostnames[worker.HostName] = struct{}{}
	}

	return err
}

func (c Config) GenerateConfigs(folderPath string) error {
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		return err
	}

	for _, controlPlane := range c.controlPlanes {
		cpPath := folderPath + "/" + c.clusterName + "-" + controlPlane.HostName + "-controlplane.yaml"
		if err := c.generateControlPlaneYAML(cpPath, controlPlane); err != nil {
			return err
		}
	}

	for _, worker := range c.workers {
		workerPath := folderPath + "/" + c.clusterName + "-" + worker.HostName + "-worker.yaml"
		if err := c.generateWorkerYAML(workerPath, worker, c.controlPlaneEndpoint); err != nil {
			return err
		}
	}

	return c.generateTalosconfig(folderPath + "/config")
}

type StorageType string

const (
	StorageTypeMMC  StorageType = "mmc"
	StorageTypeNVMe StorageType = "nvme"
)

func (s StorageType) InstallDisk() string {
	switch s {
	case StorageTypeMMC:
		return "/dev/mmcblk0"
	case StorageTypeNVMe:
		return "/dev/nvme0n1"
	default:
		return ""
	}
}

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
