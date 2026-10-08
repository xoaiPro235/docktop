package docker

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// NetworkEndpointInfo represents a container connected to a network
type NetworkEndpointInfo struct {
	ContainerID string
	Name        string
	IP          string
	MacAddress  string
	Ports       string
}

type NetworkInfo struct {
	ID             string
	Name           string
	Driver         string
	Scope          string
	Subnet         string
	Gateway        string
	Internal       bool
	IsSystem       bool
	ConnectedCount int
	Containers     []NetworkEndpointInfo
	RawInspect     *network.Inspect
}

// ListNetworks retrieves all Docker networks with detailed endpoint metadata
func (c *Client) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	listRes, err := c.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	ctrRes, _ := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	containerPortsMap := make(map[string]string)
	for _, ctr := range ctrRes.Items {
		var portStrs []string
		for _, p := range ctr.Ports {
			if p.PublicPort > 0 {
				if p.IP.Is6() || (p.IP.IsValid() && !p.IP.IsUnspecified()) {
					portStrs = append(portStrs, fmt.Sprintf("%s:%d", p.IP.String(), p.PublicPort))
				} else {
					portStrs = append(portStrs, fmt.Sprintf("%d", p.PublicPort))
				}
			} else if p.PrivatePort > 0 {
				portStrs = append(portStrs, fmt.Sprintf("%d", p.PrivatePort))
			}
		}
		if len(portStrs) > 0 {
			containerPortsMap[ctr.ID] = strings.Join(portStrs, ", ")
		}
	}

	var networks []NetworkInfo
	for _, netSummary := range listRes.Items {
		// Deep inspect each network to retrieve connected containers
		inspectRes, err := c.cli.NetworkInspect(ctx, netSummary.ID, client.NetworkInspectOptions{Verbose: true})
		if err != nil {
			continue
		}

		netInspect := inspectRes.Network

		var subnet string
		var gateway string
		if len(netInspect.IPAM.Config) > 0 {
			cfg := netInspect.IPAM.Config[0]
			if cfg.Subnet.IsValid() {
				subnet = cfg.Subnet.String()
			}
			if cfg.Gateway.IsValid() {
				gateway = cfg.Gateway.String()
			}
		}

		isSystem := netInspect.Name == "bridge" || netInspect.Name == "host" || netInspect.Name == "none"

		var endpoints []NetworkEndpointInfo
		for ctrID, endpoint := range netInspect.Containers {
			ip := "-"
			if endpoint.IPv4Address.IsValid() {
				ip = endpoint.IPv4Address.String()
			} else if endpoint.IPv6Address.IsValid() {
				ip = endpoint.IPv6Address.String()
			}
			mac := "-"
			if len(endpoint.MacAddress) > 0 {
				mac = endpoint.MacAddress.String()
			}

			ports := containerPortsMap[ctrID]
			if ports == "" {
				ports = "-"
			}

			name := endpoint.Name
			if name == "" {
				name = shortID(ctrID)
			}

			endpoints = append(endpoints, NetworkEndpointInfo{
				ContainerID: ctrID,
				Name:        name,
				IP:          ip,
				MacAddress:  mac,
				Ports:       ports,
			})
		}

		// Sort endpoints by name
		slices.SortStableFunc(endpoints, func(a, b NetworkEndpointInfo) int {
			return cmp.Compare(a.Name, b.Name)
		})

		connectedCount := len(endpoints)
		networks = append(networks, NetworkInfo{
			ID:             netInspect.ID,
			Name:           netInspect.Name,
			Driver:         netInspect.Driver,
			Scope:          netInspect.Scope,
			Subnet:         subnet,
			Gateway:        gateway,
			Internal:       netInspect.Internal,
			IsSystem:       isSystem,
			ConnectedCount: connectedCount,
			Containers:     endpoints,
			RawInspect:     &netInspect,
		})
	}

	// Sort networks:
	// 1. User networks with attached containers (score 1)
	// 2. User networks without attached containers (score 2)
	// 3. System networks (bridge, host, none) (score 3)
	// Then by name alphabetical
	slices.SortStableFunc(networks, func(a, b NetworkInfo) int {
		if c := cmp.Compare(networkScore(a), networkScore(b)); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	return networks, nil
}

func networkScore(n NetworkInfo) int {
	if n.IsSystem {
		return 3
	}
	if n.ConnectedCount > 0 {
		return 1
	}
	return 2
}

// InspectNetwork fetches full inspection metadata for a network
func (c *Client) InspectNetwork(ctx context.Context, networkID string) (*network.Inspect, error) {
	res, err := c.cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{Verbose: true})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network %s: %w", networkID, err)
	}
	return &res.Network, nil
}

// RemoveNetwork deletes a network by ID or name
func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	_, err := c.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	return err
}

// PruneNetworks removes unused networks
func (c *Client) PruneNetworks(ctx context.Context) (network.PruneReport, error) {
	res, err := c.cli.NetworkPrune(ctx, client.NetworkPruneOptions{})
	if err != nil {
		return network.PruneReport{}, err
	}
	return res.Report, nil
}
