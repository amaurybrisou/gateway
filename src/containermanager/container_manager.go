package containermanager

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerConfig struct {
	Name         string
	Image        string
	NetworkMode  string
	ExposedPorts map[string]string
	Env          []string
}

type ContainerManager struct {
	cli             *client.Client
	MaxRestartCount int
	GatewayNetwork  string
}

func NewContainerManager() (*ContainerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &ContainerManager{
		cli: cli,
	}, nil
}

func (cm *ContainerManager) CreateContainer(configHandler func() *ContainerConfig) (string, error) {
	config := configHandler()

	containerConfig := &types.ContainerCreateConfig{
		Name: config.Name,
		Config: &container.Config{
			Hostname:    config.Name,
			Env:         config.Env,
			Healthcheck: &container.HealthConfig{},
			Image:       config.Image,
		},
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cm.GatewayNetwork: {},
			},
		},
		HostConfig: &container.HostConfig{
			NetworkMode: client.DefaultDockerHost,
			RestartPolicy: container.RestartPolicy{
				MaximumRetryCount: cm.MaxRestartCount,
			},
			PortBindings: nat.PortMap{
				// Adjust according to your needs
				"8080": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
			},
		},
	}

	resp, err := cm.cli.ContainerCreate(
		context.Background(),
		containerConfig.Config,
		containerConfig.HostConfig,
		containerConfig.NetworkingConfig,
		&v1.Platform{},
		config.Name,
	)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (cm *ContainerManager) StartContainer(containerID string) error {
	err := cm.cli.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cm *ContainerManager) StopContainer(containerID string) error {
	err := cm.cli.ContainerStop(context.Background(), containerID, container.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cm *ContainerManager) DeleteContainer(containerID string) error {
	err := cm.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return err
	}

	return nil
}

func (cm *ContainerManager) GetContainerStatus(containerID string) (string, error) {
	resp, err := cm.cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "", err
	}

	return resp.State.Status, nil
}
