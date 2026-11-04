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
	network.Summary
	Subnet         string
	Gateway        string
	IsSystem       bool
	ConnectedCount int
	Containers     []NetworkEndpointInfo
}

// ListNetworks retrieves all Docker networks with detailed endpoint metadata
func (c *Client) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	listRes, err := c.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	ctrRes, _ := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	endpointsByNet := make(map[string][]NetworkEndpointInfo)

	for _, ctr := range ctrRes.Items {
		if ctr.State != StateRunning && ctr.State != StatePaused {
			continue
		}

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
		ports := "-"
		if len(portStrs) > 0 {
			ports = strings.Join(portStrs, ", ")
		}

		ctrName := ShortID(ctr.ID)
		if len(ctr.Names) > 0 {
			ctrName = strings.TrimPrefix(ctr.Names[0], "/")
		}

		if ctr.NetworkSettings != nil {
			for netName, ep := range ctr.NetworkSettings.Networks {
				if ep == nil {
					continue
				}
				ip := "-"
				if ep.IPAddress.IsValid() && !ep.IPAddress.IsUnspecified() {
					if ep.IPPrefixLen > 0 {
						ip = fmt.Sprintf("%s/%d", ep.IPAddress.String(), ep.IPPrefixLen)
					} else {
						ip = ep.IPAddress.String()
					}
				} else if ep.GlobalIPv6Address.IsValid() && !ep.GlobalIPv6Address.IsUnspecified() {
					if ep.GlobalIPv6PrefixLen > 0 {
						ip = fmt.Sprintf("%s/%d", ep.GlobalIPv6Address.String(), ep.GlobalIPv6PrefixLen)
					} else {
						ip = ep.GlobalIPv6Address.String()
					}
				}

				mac := "-"
				if len(ep.MacAddress) > 0 {
					mac = ep.MacAddress.String()
				}

				endpointInfo := NetworkEndpointInfo{
					ContainerID: ctr.ID,
					Name:        ctrName,
					IP:          ip,
					MacAddress:  mac,
					Ports:       ports,
				}

				if ep.NetworkID != "" {
					endpointsByNet[ep.NetworkID] = append(endpointsByNet[ep.NetworkID], endpointInfo)
				} else {
					endpointsByNet[netName] = append(endpointsByNet[netName], endpointInfo)
				}
			}
		}
	}

	networks := make([]NetworkInfo, 0, len(listRes.Items))
	for _, netSummary := range listRes.Items {
		var subnet string
		var gateway string
		if len(netSummary.IPAM.Config) > 0 {
			cfg := netSummary.IPAM.Config[0]
			if cfg.Subnet.IsValid() {
				subnet = cfg.Subnet.String()
			}
			if cfg.Gateway.IsValid() {
				gateway = cfg.Gateway.String()
			}
		}

		isSystem := netSummary.Name == "bridge" || netSummary.Name == "host" || netSummary.Name == "none"

		endpoints := endpointsByNet[netSummary.ID]
		if len(endpoints) == 0 {
			endpoints = endpointsByNet[netSummary.Name]
		}

		slices.SortStableFunc(endpoints, func(a, b NetworkEndpointInfo) int {
			return cmp.Compare(a.Name, b.Name)
		})

		networks = append(networks, NetworkInfo{
			Summary:        netSummary,
			Subnet:         subnet,
			Gateway:        gateway,
			IsSystem:       isSystem,
			ConnectedCount: len(endpoints),
			Containers:     endpoints,
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
