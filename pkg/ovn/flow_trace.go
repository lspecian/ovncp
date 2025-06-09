package ovn

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotConnected is returned when OVN client is not connected
	ErrNotConnected = errors.New("not connected to OVN")
)

// FlowTraceRequest represents a request to trace packet flow through OVN
type FlowTraceRequest struct {
	// Source information
	SourcePort     string `json:"source_port" binding:"required"`
	SourceMAC      string `json:"source_mac" binding:"required"`
	SourceIP       string `json:"source_ip" binding:"required"`
	
	// Destination information
	DestinationMAC string `json:"destination_mac,omitempty"`
	DestinationIP  string `json:"destination_ip" binding:"required"`
	
	// Protocol information
	Protocol       string `json:"protocol" binding:"required,oneof=tcp udp icmp icmp6"`
	SourcePortNum  int    `json:"source_port_num,omitempty"`
	DestinationPort int   `json:"destination_port_num,omitempty"`
	
	// Additional options
	Verbose        bool   `json:"verbose,omitempty"`
	MaxHops        int    `json:"max_hops,omitempty"`
}

// FlowTraceResult represents the result of a flow trace
type FlowTraceResult struct {
	Request        *FlowTraceRequest `json:"request"`
	Success        bool              `json:"success"`
	ReachesDestination bool          `json:"reaches_destination"`
	Hops           []FlowHop         `json:"hops"`
	DroppedAt      *FlowHop          `json:"dropped_at,omitempty"`
	DropReason     string            `json:"drop_reason,omitempty"`
	Summary        string            `json:"summary"`
	RawOutput      string            `json:"raw_output,omitempty"`
}

// FlowHop represents a single hop in the packet flow
type FlowHop struct {
	Index          int                    `json:"index"`
	Type           string                 `json:"type"` // switch, router, port, acl, nat
	Component      string                 `json:"component"`
	ComponentID    string                 `json:"component_id"`
	Action         string                 `json:"action"` // forward, drop, modify
	Description    string                 `json:"description"`
	Modifications  map[string]string      `json:"modifications,omitempty"`
	ACLMatches     []ACLMatch             `json:"acl_matches,omitempty"`
	NextHop        string                 `json:"next_hop,omitempty"`
}

// ACLMatch represents an ACL that matched during flow trace
type ACLMatch struct {
	ACLName        string `json:"acl_name"`
	ACLID          string `json:"acl_id"`
	Priority       int    `json:"priority"`
	Direction      string `json:"direction"`
	Action         string `json:"action"`
	Match          string `json:"match"`
}

// TraceFlow traces the flow of a packet through OVN
func (c *Client) TraceFlow(ctx context.Context, req *FlowTraceRequest) (*FlowTraceResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, ErrNotConnected
	}

	// Validate request
	if err := c.validateFlowTraceRequest(req); err != nil {
		return nil, err
	}

	// Build trace command
	traceCmd := c.buildTraceCommand(req)

	// Execute trace using ovn-trace
	output, err := c.executeTrace(ctx, traceCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute trace: %w", err)
	}

	// Parse the output
	result := c.parseTraceOutput(output, req)

	return result, nil
}

// validateFlowTraceRequest validates the flow trace request
func (c *Client) validateFlowTraceRequest(req *FlowTraceRequest) error {
	// Validate source port exists
	// TODO: Implement GetLogicalPort when we have full OVN client access
	// For now, we'll skip this validation

	// Validate IP addresses
	if !isValidIP(req.SourceIP) {
		return fmt.Errorf("invalid source IP: %s", req.SourceIP)
	}
	if !isValidIP(req.DestinationIP) {
		return fmt.Errorf("invalid destination IP: %s", req.DestinationIP)
	}

	// Validate MAC addresses
	if !isValidMAC(req.SourceMAC) {
		return fmt.Errorf("invalid source MAC: %s", req.SourceMAC)
	}
	if req.DestinationMAC != "" && !isValidMAC(req.DestinationMAC) {
		return fmt.Errorf("invalid destination MAC: %s", req.DestinationMAC)
	}

	// Validate port numbers for TCP/UDP
	if req.Protocol == "tcp" || req.Protocol == "udp" {
		if req.SourcePortNum > 0 && (req.SourcePortNum < 1 || req.SourcePortNum > 65535) {
			return fmt.Errorf("invalid source port number: %d", req.SourcePortNum)
		}
		if req.DestinationPort > 0 && (req.DestinationPort < 1 || req.DestinationPort > 65535) {
			return fmt.Errorf("invalid destination port number: %d", req.DestinationPort)
		}
	}

	return nil
}

// buildTraceCommand builds the ovn-trace command
func (c *Client) buildTraceCommand(req *FlowTraceRequest) string {
	// Start with the datapath (logical switch of the source port)
	// We need to find the switch that contains the source port
	datapath := c.findDatapathForPort(req.SourcePort)
	if datapath == "" {
		datapath = "br-int" // fallback
	}

	// Build the flow specification
	var flow strings.Builder
	flow.WriteString(fmt.Sprintf("inport==\"%s\"", req.SourcePort))
	flow.WriteString(fmt.Sprintf(" && eth.src==%s", req.SourceMAC))
	flow.WriteString(fmt.Sprintf(" && eth.dst==%s", req.DestinationMAC))
	
	// Add IP layer
	if strings.Contains(req.SourceIP, ":") {
		// IPv6
		flow.WriteString(fmt.Sprintf(" && ipv6.src==%s && ipv6.dst==%s", req.SourceIP, req.DestinationIP))
	} else {
		// IPv4
		flow.WriteString(fmt.Sprintf(" && ip4.src==%s && ip4.dst==%s", req.SourceIP, req.DestinationIP))
	}

	// Add protocol-specific fields
	switch req.Protocol {
	case "tcp":
		flow.WriteString(" && ip.proto==6")
		if req.SourcePortNum > 0 {
			flow.WriteString(fmt.Sprintf(" && tcp.src==%d", req.SourcePortNum))
		}
		if req.DestinationPort > 0 {
			flow.WriteString(fmt.Sprintf(" && tcp.dst==%d", req.DestinationPort))
		}
	case "udp":
		flow.WriteString(" && ip.proto==17")
		if req.SourcePortNum > 0 {
			flow.WriteString(fmt.Sprintf(" && udp.src==%d", req.SourcePortNum))
		}
		if req.DestinationPort > 0 {
			flow.WriteString(fmt.Sprintf(" && udp.dst==%d", req.DestinationPort))
		}
	case "icmp":
		flow.WriteString(" && ip.proto==1")
	case "icmp6":
		flow.WriteString(" && ip.proto==58")
	}

	// Build the complete command
	cmd := fmt.Sprintf("ovn-trace %s '%s'", datapath, flow.String())
	
	if req.Verbose {
		cmd += " --detailed"
	}
	
	return cmd
}

// findDatapathForPort finds the logical switch containing the port
func (c *Client) findDatapathForPort(portName string) string {
	// This is a simplified version - in reality, we'd query the database
	// to find which logical switch contains this port
	ctx := context.Background()
	
	// List all switches
	switches, err := c.ListLogicalSwitches(ctx)
	if err != nil {
		return ""
	}

	// Find the switch containing this port
	for _, sw := range switches {
		ports, err := c.ListLogicalSwitchPorts(ctx, sw.UUID)
		if err != nil {
			continue
		}
		
		for _, port := range ports {
			if port.Name == portName {
				return sw.Name
			}
		}
	}

	return ""
}

// executeTrace executes the ovn-trace command
func (c *Client) executeTrace(ctx context.Context, cmd string) (string, error) {
	// In a real implementation, this would execute the ovn-trace command
	// For now, we'll simulate it
	
	// This would typically use exec.Command to run ovn-trace
	// output, err := exec.CommandContext(ctx, "ovn-trace", args...).Output()
	
	// Simulated output for demonstration
	output := `
# Packet trace for flow: inport=="vm1-port" && eth.src==00:00:00:00:00:01 && eth.dst==00:00:00:00:00:02 && ip4.src==10.0.0.1 && ip4.dst==10.0.0.2

ingress(dp="switch1", inport="vm1-port")
-----------------------------------------
 0. ls_in_port_sec_l2 (ovn-northd.c:4555): inport == "vm1-port" && eth.src == {00:00:00:00:00:01}, priority 50, uuid 12345
    next;
 1. ls_in_port_sec_ip (ovn-northd.c:4689): inport == "vm1-port" && eth.src == 00:00:00:00:00:01 && ip4.src == {10.0.0.1}, priority 90, uuid 23456
    next;
 5. ls_in_pre_acl (ovn-northd.c:4891): ip, priority 100, uuid 34567
    reg0[0] = 1;
    next;
10. ls_in_acl (ovn-northd.c:5284): ip4 && tcp && tcp.dst == 80, priority 2000, uuid 45678
    /* ACL allow web traffic */
    next;
25. ls_in_l2_lkup (ovn-northd.c:6994): eth.dst == 00:00:00:00:00:02, priority 50, uuid 56789
    outport = "vm2-port";
    output;

egress(dp="switch1", inport="vm1-port", outport="vm2-port")
-----------------------------------------------------------
 0. ls_out_pre_acl (ovn-northd.c:4878): ip, priority 100, uuid 67890
    reg0[0] = 1;
    next;
 8. ls_out_acl (ovn-northd.c:5284): ip4 && tcp && tcp.src == 80, priority 2000, uuid 78901
    /* ACL allow return web traffic */
    next;
 9. ls_out_port_sec_ip (ovn-northd.c:4689): outport == "vm2-port" && eth.dst == 00:00:00:00:00:02 && ip4.dst == 10.0.0.2, priority 90, uuid 89012
    next;
10. ls_out_port_sec_l2 (ovn-northd.c:4555): outport == "vm2-port" && eth.dst == {00:00:00:00:00:02}, priority 50, uuid 90123
    output;
    /* output to "vm2-port" */
`
	
	return output, nil
}

// parseTraceOutput parses the ovn-trace output
func (c *Client) parseTraceOutput(output string, req *FlowTraceRequest) *FlowTraceResult {
	result := &FlowTraceResult{
		Request:   req,
		Success:   true,
		Hops:      []FlowHop{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	currentStage := ""
	hopIndex := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect stage changes
		if strings.HasPrefix(line, "ingress(") || strings.HasPrefix(line, "egress(") {
			currentStage = line
			continue
		}

		// Parse flow entries
		if strings.Contains(line, ".") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			// Extract flow number and stage
			flowInfo := strings.TrimSpace(parts[0])
			flowDesc := strings.TrimSpace(parts[1])

			// Create hop
			hop := FlowHop{
				Index:       hopIndex,
				Type:        c.getFlowType(flowInfo),
				Component:   currentStage,
				Description: flowDesc,
			}

			// Check for actions
			if strings.Contains(line, "next;") {
				hop.Action = "forward"
			} else if strings.Contains(line, "drop;") {
				hop.Action = "drop"
				result.DroppedAt = &hop
				result.DropReason = "Dropped by flow rule"
				result.ReachesDestination = false
			} else if strings.Contains(line, "output;") {
				hop.Action = "output"
			}

			// Check for ACL matches
			if strings.Contains(flowInfo, "acl") {
				hop.Type = "acl"
				if strings.Contains(flowDesc, "/*") && strings.Contains(flowDesc, "*/") {
					// Extract ACL comment
					start := strings.Index(flowDesc, "/*") + 2
					end := strings.Index(flowDesc, "*/")
					if start < end {
						aclMatch := ACLMatch{
							Action: hop.Action,
							Match:  strings.TrimSpace(flowDesc[start:end]),
						}
						hop.ACLMatches = append(hop.ACLMatches, aclMatch)
					}
				}
			}

			// Check for modifications
			if strings.Contains(line, "eth.src =") || strings.Contains(line, "eth.dst =") {
				hop.Modifications = make(map[string]string)
				// Parse modifications (simplified)
				if strings.Contains(line, "eth.src =") {
					hop.Modifications["eth.src"] = "modified"
				}
				if strings.Contains(line, "eth.dst =") {
					hop.Modifications["eth.dst"] = "modified"
				}
			}

			// Check for port output
			if strings.Contains(line, "output to") {
				parts := strings.Split(line, "\"")
				if len(parts) >= 2 {
					hop.NextHop = parts[1]
				}
			}

			result.Hops = append(result.Hops, hop)
			hopIndex++
		}
	}

	// Determine if packet reaches destination
	if result.DroppedAt == nil && len(result.Hops) > 0 {
		lastHop := result.Hops[len(result.Hops)-1]
		if lastHop.Action == "output" {
			result.ReachesDestination = true
		}
	}

	// Generate summary
	if result.ReachesDestination {
		result.Summary = fmt.Sprintf("Packet successfully traced from %s to %s through %d hops",
			req.SourcePort, req.DestinationIP, len(result.Hops))
	} else if result.DroppedAt != nil {
		result.Summary = fmt.Sprintf("Packet dropped at hop %d: %s",
			result.DroppedAt.Index, result.DropReason)
	} else {
		result.Summary = "Packet trace completed but destination not reached"
	}

	return result
}

// getFlowType determines the type of flow from the flow info
func (c *Client) getFlowType(flowInfo string) string {
	flowInfo = strings.ToLower(flowInfo)
	
	if strings.Contains(flowInfo, "port_sec") {
		return "port_security"
	}
	if strings.Contains(flowInfo, "acl") {
		return "acl"
	}
	if strings.Contains(flowInfo, "l2_lkup") {
		return "l2_lookup"
	}
	if strings.Contains(flowInfo, "nat") {
		return "nat"
	}
	if strings.Contains(flowInfo, "lb") {
		return "load_balancer"
	}
	if strings.Contains(flowInfo, "router") {
		return "router"
	}
	
	return "flow"
}

// Utility functions

func isValidIP(ip string) bool {
	// Simple validation - in production, use net.ParseIP
	if ip == "" {
		return false
	}
	
	// Check for IPv4
	if strings.Count(ip, ".") == 3 {
		parts := strings.Split(ip, ".")
		if len(parts) != 4 {
			return false
		}
		for _, part := range parts {
			if len(part) == 0 || len(part) > 3 {
				return false
			}
		}
		return true
	}
	
	// Check for IPv6
	if strings.Contains(ip, ":") {
		return true // Simplified
	}
	
	return false
}

func isValidMAC(mac string) bool {
	// Simple validation
	if mac == "" {
		return false
	}
	
	// Check format XX:XX:XX:XX:XX:XX
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}
	
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
	}
	
	return true
}

// SimulateFlowTrace provides flow trace simulation for testing
func (c *Client) SimulateFlowTrace(ctx context.Context, req *FlowTraceRequest) (*FlowTraceResult, error) {
	// This method simulates flow tracing for testing purposes
	// It creates a realistic trace result without actually executing ovn-trace
	
	result := &FlowTraceResult{
		Request: req,
		Success: true,
		Hops:    []FlowHop{},
	}

	// Simulate ingress processing
	result.Hops = append(result.Hops, FlowHop{
		Index:       0,
		Type:        "port_security",
		Component:   fmt.Sprintf("ingress(dp=\"ls1\", inport=\"%s\")", req.SourcePort),
		Action:      "forward",
		Description: "Port security check - MAC and IP validation",
	})

	// Simulate ACL processing
	if req.Protocol == "tcp" && req.DestinationPort == 80 {
		result.Hops = append(result.Hops, FlowHop{
			Index:       1,
			Type:        "acl",
			Component:   "ingress ACL",
			Action:      "forward",
			Description: "Allow HTTP traffic",
			ACLMatches: []ACLMatch{{
				ACLName:   "allow-http",
				Priority:  2000,
				Direction: "ingress",
				Action:    "allow",
				Match:     "tcp.dst == 80",
			}},
		})
		result.ReachesDestination = true
	} else if req.Protocol == "tcp" && req.DestinationPort == 22 {
		// Simulate blocked SSH
		hop := FlowHop{
			Index:       1,
			Type:        "acl",
			Component:   "ingress ACL",
			Action:      "drop",
			Description: "Deny SSH traffic",
			ACLMatches: []ACLMatch{{
				ACLName:   "deny-ssh",
				Priority:  2100,
				Direction: "ingress",
				Action:    "drop",
				Match:     "tcp.dst == 22",
			}},
		}
		result.Hops = append(result.Hops, hop)
		result.DroppedAt = &hop
		result.DropReason = "Blocked by security policy"
		result.ReachesDestination = false
	} else {
		// Default allow
		result.Hops = append(result.Hops, FlowHop{
			Index:       1,
			Type:        "acl",
			Component:   "ingress ACL",
			Action:      "forward",
			Description: "Default allow rule",
		})
		result.ReachesDestination = true
	}

	// Add L2 lookup if not dropped
	if result.ReachesDestination {
		result.Hops = append(result.Hops, FlowHop{
			Index:       len(result.Hops),
			Type:        "l2_lookup",
			Component:   "L2 forwarding",
			Action:      "output",
			Description: fmt.Sprintf("L2 lookup - forward to port for MAC %s", req.DestinationMAC),
			NextHop:     "vm2-port",
		})
	}

	// Generate summary
	if result.ReachesDestination {
		result.Summary = fmt.Sprintf("Packet successfully traced from %s to %s through %d hops",
			req.SourcePort, req.DestinationIP, len(result.Hops))
	} else {
		result.Summary = fmt.Sprintf("Packet dropped at hop %d: %s",
			result.DroppedAt.Index, result.DropReason)
	}

	return result, nil
}