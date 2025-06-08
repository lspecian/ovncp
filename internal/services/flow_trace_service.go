package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn"
	"go.uber.org/zap"
)

// FlowTraceService provides flow tracing capabilities
type FlowTraceService struct {
	ovnClient *ovn.Client
	logger    *zap.Logger
}

// NewFlowTraceService creates a new flow trace service
func NewFlowTraceService(ovnClient *ovn.Client, logger *zap.Logger) *FlowTraceService {
	return &FlowTraceService{
		ovnClient: ovnClient,
		logger:    logger,
	}
}

// TraceFlow traces a packet flow through the OVN network
func (s *FlowTraceService) TraceFlow(ctx context.Context, req *ovn.FlowTraceRequest) (*ovn.FlowTraceResult, error) {
	s.logger.Info("Tracing flow",
		zap.String("source_port", req.SourcePort),
		zap.String("source_ip", req.SourceIP),
		zap.String("destination_ip", req.DestinationIP),
		zap.String("protocol", req.Protocol))

	// Perform the trace
	result, err := s.ovnClient.TraceFlow(ctx, req)
	if err != nil {
		s.logger.Error("Flow trace failed", zap.Error(err))
		return nil, fmt.Errorf("flow trace failed: %w", err)
	}

	// Log the result
	if result.Success {
		if result.ReachesDestination {
			s.logger.Info("Flow trace completed - packet reaches destination",
				zap.Int("hops", len(result.Hops)))
		} else {
			s.logger.Warn("Flow trace completed - packet dropped",
				zap.Int("dropped_at_hop", result.DroppedAt.Index),
				zap.String("drop_reason", result.DropReason))
		}
	}

	return result, nil
}

// TraceMultiplePaths traces multiple paths between two endpoints
func (s *FlowTraceService) TraceMultiplePaths(ctx context.Context, req *MultiPathTraceRequest) (*MultiPathTraceResult, error) {
	result := &MultiPathTraceResult{
		SourcePort:     req.SourcePort,
		DestinationIP:  req.DestinationIP,
		Paths:          []PathResult{},
	}

	// Trace different protocols
	protocols := req.Protocols
	if len(protocols) == 0 {
		protocols = []string{"tcp", "udp", "icmp"}
	}

	for _, protocol := range protocols {
		// For TCP/UDP, trace common ports
		if protocol == "tcp" || protocol == "udp" {
			ports := req.Ports
			if len(ports) == 0 {
				// Default common ports
				ports = []int{22, 80, 443, 3306, 5432, 6379, 8080}
			}

			for _, port := range ports {
				traceReq := &ovn.FlowTraceRequest{
					SourcePort:      req.SourcePort,
					SourceMAC:       req.SourceMAC,
					SourceIP:        req.SourceIP,
					DestinationMAC:  req.DestinationMAC,
					DestinationIP:   req.DestinationIP,
					Protocol:        protocol,
					DestinationPort: port,
				}

				trace, err := s.ovnClient.TraceFlow(ctx, traceReq)
				if err != nil {
					s.logger.Warn("Failed to trace path",
						zap.String("protocol", protocol),
						zap.Int("port", port),
						zap.Error(err))
					continue
				}

				path := PathResult{
					Protocol:           protocol,
					Port:               port,
					ReachesDestination: trace.ReachesDestination,
					Blocked:            !trace.ReachesDestination,
					BlockedBy:          s.extractBlockingRule(trace),
					HopCount:           len(trace.Hops),
				}

				result.Paths = append(result.Paths, path)
			}
		} else {
			// For ICMP, just trace once
			traceReq := &ovn.FlowTraceRequest{
				SourcePort:     req.SourcePort,
				SourceMAC:      req.SourceMAC,
				SourceIP:       req.SourceIP,
				DestinationMAC: req.DestinationMAC,
				DestinationIP:  req.DestinationIP,
				Protocol:       protocol,
			}

			trace, err := s.ovnClient.TraceFlow(ctx, traceReq)
			if err != nil {
				s.logger.Warn("Failed to trace ICMP path", zap.Error(err))
				continue
			}

			path := PathResult{
				Protocol:           protocol,
				ReachesDestination: trace.ReachesDestination,
				Blocked:            !trace.ReachesDestination,
				BlockedBy:          s.extractBlockingRule(trace),
				HopCount:           len(trace.Hops),
			}

			result.Paths = append(result.Paths, path)
		}
	}

	// Generate summary
	result.Summary = s.generateMultiPathSummary(result)

	return result, nil
}

// AnalyzeConnectivity performs comprehensive connectivity analysis
func (s *FlowTraceService) AnalyzeConnectivity(ctx context.Context, req *ConnectivityAnalysisRequest) (*ConnectivityAnalysisResult, error) {
	result := &ConnectivityAnalysisResult{
		SourcePort:    req.SourcePort,
		TargetPorts:   []PortConnectivity{},
		FullyReachable: true,
	}

	// Get source port details
	// TODO: Implement when GetLogicalPort is available
	// sourcePort, err := s.ovnClient.GetLogicalPort(ctx, req.SourcePort)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to get source port: %w", err)
	// }
	// For now, create a dummy source port
	sourcePort := &models.LogicalSwitchPort{
		Name: req.SourcePort,
	}

	// Extract source IP and MAC
	sourceIP, sourceMAC := s.extractPortAddresses(sourcePort)
	if sourceIP == "" || sourceMAC == "" {
		return nil, fmt.Errorf("source port missing IP or MAC address")
	}

	// Test connectivity to each target
	for _, targetPortName := range req.TargetPorts {
		// TODO: Implement when GetLogicalPort is available
		// targetPort, err := s.ovnClient.GetLogicalPort(ctx, targetPortName)
		// if err != nil {
		//     s.logger.Warn("Failed to get target port", 
		//         zap.String("port", targetPortName),
		//         zap.Error(err))
		//     continue
		// }
		// For now, create a dummy target port
		targetPort := &models.LogicalSwitchPort{
			Name: targetPortName,
		}

		targetIP, targetMAC := s.extractPortAddresses(targetPort)
		if targetIP == "" {
			s.logger.Warn("Target port missing IP address",
				zap.String("port", targetPortName))
			continue
		}

		// Test different protocols
		portConn := PortConnectivity{
			PortName:  targetPortName,
			PortIP:    targetIP,
			Protocols: make(map[string]bool),
		}

		// Test ICMP
		icmpReq := &ovn.FlowTraceRequest{
			SourcePort:     req.SourcePort,
			SourceMAC:      sourceMAC,
			SourceIP:       sourceIP,
			DestinationMAC: targetMAC,
			DestinationIP:  targetIP,
			Protocol:       "icmp",
		}

		icmpTrace, err := s.ovnClient.TraceFlow(ctx, icmpReq)
		if err == nil {
			portConn.Protocols["icmp"] = icmpTrace.ReachesDestination
			portConn.Reachable = icmpTrace.ReachesDestination
		}

		// Test common TCP ports
		for _, port := range []int{22, 80, 443} {
			tcpReq := &ovn.FlowTraceRequest{
				SourcePort:      req.SourcePort,
				SourceMAC:       sourceMAC,
				SourceIP:        sourceIP,
				DestinationMAC:  targetMAC,
				DestinationIP:   targetIP,
				Protocol:        "tcp",
				DestinationPort: port,
			}

			tcpTrace, err := s.ovnClient.TraceFlow(ctx, tcpReq)
			if err == nil && tcpTrace.ReachesDestination {
				portConn.Protocols[fmt.Sprintf("tcp:%d", port)] = true
				portConn.Reachable = true
			}
		}

		if !portConn.Reachable {
			result.FullyReachable = false
		}

		result.TargetPorts = append(result.TargetPorts, portConn)
	}

	// Generate recommendations
	result.Recommendations = s.generateConnectivityRecommendations(result)

	return result, nil
}

// extractBlockingRule extracts the blocking rule from a trace result
func (s *FlowTraceService) extractBlockingRule(trace *ovn.FlowTraceResult) string {
	if trace.DroppedAt == nil {
		return ""
	}

	if len(trace.DroppedAt.ACLMatches) > 0 {
		acl := trace.DroppedAt.ACLMatches[0]
		return fmt.Sprintf("ACL: %s (Priority: %d)", acl.ACLName, acl.Priority)
	}

	return trace.DropReason
}

// extractPortAddresses extracts IP and MAC from a port
func (s *FlowTraceService) extractPortAddresses(port *models.LogicalSwitchPort) (string, string) {
	if len(port.Addresses) == 0 {
		return "", ""
	}

	// Parse the first address (format: "MAC IP")
	parts := strings.Fields(port.Addresses[0])
	if len(parts) >= 2 {
		return parts[1], parts[0]
	}

	return "", ""
}

// generateMultiPathSummary generates a summary for multi-path trace results
func (s *FlowTraceService) generateMultiPathSummary(result *MultiPathTraceResult) string {
	totalPaths := len(result.Paths)
	blockedPaths := 0
	allowedProtocols := []string{}

	for _, path := range result.Paths {
		if path.Blocked {
			blockedPaths++
		} else {
			if path.Port > 0 {
				allowedProtocols = append(allowedProtocols, 
					fmt.Sprintf("%s:%d", path.Protocol, path.Port))
			} else {
				allowedProtocols = append(allowedProtocols, path.Protocol)
			}
		}
	}

	if blockedPaths == 0 {
		return fmt.Sprintf("All %d paths tested are allowed. Allowed protocols: %s",
			totalPaths, strings.Join(allowedProtocols, ", "))
	} else if blockedPaths == totalPaths {
		return fmt.Sprintf("All %d paths tested are blocked", totalPaths)
	} else {
		return fmt.Sprintf("%d of %d paths blocked. Allowed: %s",
			blockedPaths, totalPaths, strings.Join(allowedProtocols, ", "))
	}
}

// generateConnectivityRecommendations generates recommendations based on analysis
func (s *FlowTraceService) generateConnectivityRecommendations(result *ConnectivityAnalysisResult) []string {
	recommendations := []string{}

	if result.FullyReachable {
		recommendations = append(recommendations, 
			"All target ports are reachable from the source port")
		return recommendations
	}

	// Find unreachable ports
	unreachablePorts := []string{}
	for _, port := range result.TargetPorts {
		if !port.Reachable {
			unreachablePorts = append(unreachablePorts, port.PortName)
		}
	}

	if len(unreachablePorts) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("The following ports are unreachable: %s",
				strings.Join(unreachablePorts, ", ")))
		
		recommendations = append(recommendations,
			"Check ACL rules and ensure proper routing between network segments")
	}

	// Check for partial connectivity
	for _, port := range result.TargetPorts {
		if port.Reachable && len(port.Protocols) < 3 {
			recommendations = append(recommendations,
				fmt.Sprintf("Port %s has limited protocol access. Consider reviewing ACL rules",
					port.PortName))
		}
	}

	return recommendations
}

// Types for multi-path and connectivity analysis

// MultiPathTraceRequest requests tracing of multiple paths
type MultiPathTraceRequest struct {
	SourcePort     string   `json:"source_port" binding:"required"`
	SourceMAC      string   `json:"source_mac" binding:"required"`
	SourceIP       string   `json:"source_ip" binding:"required"`
	DestinationMAC string   `json:"destination_mac,omitempty"`
	DestinationIP  string   `json:"destination_ip" binding:"required"`
	Protocols      []string `json:"protocols,omitempty"`
	Ports          []int    `json:"ports,omitempty"`
}

// MultiPathTraceResult contains results for multiple path traces
type MultiPathTraceResult struct {
	SourcePort    string       `json:"source_port"`
	DestinationIP string       `json:"destination_ip"`
	Paths         []PathResult `json:"paths"`
	Summary       string       `json:"summary"`
}

// PathResult represents a single path trace result
type PathResult struct {
	Protocol           string `json:"protocol"`
	Port               int    `json:"port,omitempty"`
	ReachesDestination bool   `json:"reaches_destination"`
	Blocked            bool   `json:"blocked"`
	BlockedBy          string `json:"blocked_by,omitempty"`
	HopCount           int    `json:"hop_count"`
}

// ConnectivityAnalysisRequest requests connectivity analysis
type ConnectivityAnalysisRequest struct {
	SourcePort  string   `json:"source_port" binding:"required"`
	TargetPorts []string `json:"target_ports" binding:"required"`
}

// ConnectivityAnalysisResult contains connectivity analysis results
type ConnectivityAnalysisResult struct {
	SourcePort      string              `json:"source_port"`
	TargetPorts     []PortConnectivity  `json:"target_ports"`
	FullyReachable  bool                `json:"fully_reachable"`
	Recommendations []string            `json:"recommendations"`
}

// PortConnectivity represents connectivity to a specific port
type PortConnectivity struct {
	PortName  string            `json:"port_name"`
	PortIP    string            `json:"port_ip"`
	Reachable bool              `json:"reachable"`
	Protocols map[string]bool   `json:"protocols"`
}