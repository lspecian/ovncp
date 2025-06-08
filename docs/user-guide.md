# OVN Control Platform User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Dashboard Overview](#dashboard-overview)
4. [Managing Logical Switches](#managing-logical-switches)
5. [Managing Logical Routers](#managing-logical-routers)
6. [Working with Ports](#working-with-ports)
7. [Configuring ACLs](#configuring-acls)
8. [Load Balancer Configuration](#load-balancer-configuration)
9. [Network Topology View](#network-topology-view)
10. [Monitoring and Metrics](#monitoring-and-metrics)
11. [Troubleshooting](#troubleshooting)
12. [Best Practices](#best-practices)

## Introduction

OVN Control Platform (OVNCP) provides a user-friendly interface for managing Open Virtual Network infrastructure. This guide will help you navigate the platform and perform common tasks efficiently.

### Key Concepts

Before using OVNCP, familiarize yourself with these OVN concepts:

- **Logical Switch**: A virtual L2 network segment
- **Logical Router**: A virtual router for L3 connectivity
- **Logical Port**: Connection point on a switch or router
- **ACL**: Access Control List for security policies
- **Load Balancer**: Distributes traffic across multiple backends
- **NAT**: Network Address Translation rules

## Getting Started

### Logging In

1. Navigate to your OVNCP instance (e.g., https://ovncp.example.com)
2. Click "Sign In" and choose your authentication provider
3. Authorize the application when prompted
4. You'll be redirected to the dashboard

### First-Time Setup

Upon first login, you should:

1. **Update your profile**: Click your avatar → Profile Settings
2. **Set preferences**: Configure timezone, notifications, and display options
3. **Generate API token**: For CLI/API access, go to Settings → API Tokens

### User Interface Overview

```
┌─────────────────────────────────────────────────────────┐
│  🔷 OVN Control Platform          [🔍] [🔔] [👤]         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐  ┌─────────────────────────────────────┐ │
│  │          │  │                                     │ │
│  │ Sidebar  │  │          Main Content Area          │ │
│  │          │  │                                     │ │
│  │ • Switch │  │                                     │ │
│  │ • Router │  │                                     │ │
│  │ • ACLs   │  │                                     │ │
│  │ • LBs    │  │                                     │ │
│  │          │  │                                     │ │
│  └──────────┘  └─────────────────────────────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Dashboard Overview

The dashboard provides a high-level view of your OVN infrastructure:

### Key Metrics

- **Resource Counts**: Total switches, routers, ports, and ACLs
- **Health Status**: Overall system health and alerts
- **Recent Activity**: Latest changes and operations
- **Performance Metrics**: Request latency and throughput

### Quick Actions

From the dashboard, you can:
- Create new resources using the "+" button
- Search for resources using the global search
- Access recent items from the activity feed
- View system notifications

## Managing Logical Switches

### Creating a Logical Switch

1. Navigate to **Network → Logical Switches**
2. Click **"Create Switch"**
3. Fill in the required fields:
   - **Name**: Unique identifier (e.g., "web-tier")
   - **Description**: Optional description
   - **Subnet**: CIDR notation (e.g., "10.0.1.0/24")
   - **DNS Servers**: Comma-separated IP addresses
4. Click **"Create"**

### Editing a Switch

1. Click on the switch name or select **Actions → Edit**
2. Modify the desired fields
3. Click **"Save Changes"**

### Deleting a Switch

⚠️ **Warning**: Deleting a switch removes all associated ports.

1. Select the switch(es) to delete
2. Click **Actions → Delete**
3. Confirm the operation

### Switch Operations

#### Viewing Switch Details

Click on a switch name to view:
- Basic information (UUID, name, subnet)
- Associated ports
- Connected routers
- Applied ACLs
- Recent events

#### Bulk Operations

Select multiple switches using checkboxes to:
- Delete multiple switches
- Export configurations
- Apply common settings

## Managing Logical Routers

### Creating a Router

1. Navigate to **Network → Logical Routers**
2. Click **"Create Router"**
3. Configure router settings:
   - **Name**: Unique identifier
   - **External Gateway**: Optional external IP
   - **Enable SNAT**: Source NAT for outbound traffic
4. Click **"Create"**

### Configuring Router Ports

1. Open router details
2. Go to **Ports** tab
3. Click **"Add Port"**
4. Choose port type:
   - **Gateway Port**: External connectivity
   - **Router Port**: Connect to switches

### Static Routes

To add static routes:

1. Open router details
2. Go to **Routes** tab
3. Click **"Add Route"**
4. Specify:
   - **Destination**: CIDR (e.g., "192.168.0.0/16")
   - **Next Hop**: Gateway IP
   - **Metric**: Route priority (lower = higher priority)

### NAT Rules

Configure NAT in the **NAT** tab:

#### SNAT (Source NAT)
```
Internal IP: 10.0.1.0/24
External IP: 203.0.113.1
```

#### DNAT (Destination NAT)
```
External IP: 203.0.113.1:80
Internal IP: 10.0.1.10:8080
```

## Working with Ports

### Port Types

- **VIF Ports**: Virtual machine interfaces
- **Router Ports**: Connect routers to switches
- **Localnet Ports**: Physical network connectivity
- **Patch Ports**: Connect logical switches

### Creating Ports

1. Navigate to the parent switch/router
2. Go to **Ports** tab
3. Click **"Add Port"**
4. Configure port settings:
   - **Name**: Unique identifier
   - **MAC Address**: Hardware address
   - **IP Address**: Static IP (optional)
   - **Security Groups**: Applied security policies

### Port Security

Enable port security to:
- Prevent MAC spoofing
- Restrict IP addresses
- Apply security groups

```yaml
Port Security Settings:
  ✅ Enable Port Security
  ✅ Allow MAC: 00:00:00:00:00:01
  ✅ Allow IPs: 10.0.1.10, 10.0.1.11
  ✅ Security Groups: web-servers, ssh-access
```

## Configuring ACLs

### Understanding ACL Priority

ACLs are processed in priority order (0-32767):
- **0-999**: System/infrastructure rules
- **1000-1999**: Security policies
- **2000-2999**: Application rules
- **3000+**: Default/catch-all rules

### Creating ACLs

1. Navigate to **Security → ACLs**
2. Click **"Create ACL"**
3. Configure ACL parameters:

```yaml
Name: allow-web-traffic
Priority: 1000
Direction: from-lport
Match: "tcp.dst == 80 || tcp.dst == 443"
Action: allow
Apply To: logical_switch: web-tier
```

### Common ACL Patterns

#### Allow SSH from Management Network
```
Priority: 1000
Direction: to-lport
Match: "ip4.src == 10.0.0.0/24 && tcp.dst == 22"
Action: allow
```

#### Block All Except Established
```
Priority: 2000
Direction: from-lport
Match: "ct.est"
Action: allow

Priority: 3000
Direction: from-lport
Match: "1"
Action: drop
```

#### Rate Limiting
```
Priority: 1500
Direction: from-lport
Match: "tcp.dst == 80"
Action: allow
Meter: http-rate-limit
```

## Load Balancer Configuration

### Creating a Load Balancer

1. Navigate to **Network → Load Balancers**
2. Click **"Create Load Balancer"**
3. Configure basic settings:
   - **Name**: Unique identifier
   - **Protocol**: TCP/UDP
   - **Algorithm**: Round-robin/Least connections

### Adding VIPs (Virtual IPs)

1. Open load balancer details
2. Go to **VIPs** tab
3. Click **"Add VIP"**
4. Configure:
   ```yaml
   VIP: 192.168.1.100:80
   Backends:
     - 10.0.1.10:8080
     - 10.0.1.11:8080
     - 10.0.1.12:8080
   Health Check:
     Type: HTTP
     Path: /health
     Interval: 5s
   ```

### Health Checks

Configure health checks to ensure traffic only goes to healthy backends:

- **HTTP**: Check specific endpoint
- **TCP**: Verify port connectivity
- **Custom**: Execute custom scripts

## Network Topology View

### Accessing Topology View

Click **Topology** in the main navigation to see:
- Visual network representation
- Real-time status updates
- Interactive navigation

### Topology Features

#### Filtering
- Filter by resource type
- Search for specific resources
- Show/hide resource labels

#### Layout Options
- **Hierarchical**: Tree structure
- **Force-directed**: Automatic positioning
- **Manual**: Drag and position nodes

#### Interactions
- **Click**: View resource details
- **Double-click**: Edit resource
- **Right-click**: Context menu
- **Drag**: Reposition nodes

### Understanding the Visualization

```
[🟦 Switch] ←→ [🟨 Router] ←→ [🟦 Switch]
     ↓              ↓              ↓
[🟩 Port]      [🟩 Port]      [🟩 Port]
```

- **Blue**: Logical switches
- **Yellow**: Logical routers
- **Green**: Ports
- **Red**: Errors/issues
- **Solid lines**: Active connections
- **Dashed lines**: Inactive/planned

## Monitoring and Metrics

### Resource Metrics

Each resource page shows:
- **Usage Statistics**: Traffic, connections, errors
- **Performance Metrics**: Latency, throughput
- **Historical Trends**: Time-series graphs

### Setting Up Alerts

1. Go to **Monitoring → Alerts**
2. Click **"Create Alert"**
3. Configure alert conditions:
   ```yaml
   Metric: port_traffic_bytes
   Condition: > 1000000000  # 1GB
   Duration: 5 minutes
   Actions:
     - Email: ops-team@example.com
     - Slack: #alerts
   ```

### Dashboards

Create custom dashboards:
1. Go to **Monitoring → Dashboards**
2. Click **"New Dashboard"**
3. Add widgets:
   - Metric graphs
   - Resource lists
   - Status indicators
   - Activity feeds

## Troubleshooting

### Common Issues

#### Cannot Create Resources
- **Check permissions**: Ensure you have write access
- **Verify quotas**: Check resource limits
- **Name conflicts**: Use unique names
- **Network connectivity**: Verify OVN connection

#### Port Not Receiving Traffic
1. Check ACLs on the switch
2. Verify port security settings
3. Ensure correct VLAN/network configuration
4. Check router connectivity

#### High Latency
1. Review topology for optimal routing
2. Check for ACL processing overhead
3. Monitor OVN controller load
4. Verify physical network performance

### Diagnostic Tools

#### Connection Test
Test connectivity between ports:
```
Source Port: web-01
Destination Port: db-01
Protocol: TCP
Port: 5432
Result: ✅ Connected (2.3ms)
```

#### Trace Route
View packet path through the network:
```
web-01 → switch-web → router-main → switch-db → db-01
```

#### ACL Analyzer
Check which ACLs affect traffic:
```
Source: 10.0.1.10
Destination: 10.0.2.20:5432
Matching ACLs:
  ✅ allow-web-to-db (Priority: 1000)
  ❌ deny-all (Priority: 32767)
Result: Allowed
```

## Best Practices

### Naming Conventions

Use consistent naming:
- **Switches**: `{tier}-{purpose}-{number}` (e.g., "prod-web-01")
- **Routers**: `{location}-{type}-{number}` (e.g., "dc1-edge-01")
- **Ports**: `{vm/host}-{interface}` (e.g., "web01-eth0")

### Security Best Practices

1. **Least Privilege**: Only allow necessary traffic
2. **Defense in Depth**: Multiple security layers
3. **Regular Audits**: Review ACLs periodically
4. **Change Management**: Document all changes

### Performance Optimization

1. **Minimize ACL Rules**: Combine where possible
2. **Use Specific Matches**: Avoid wildcard rules
3. **Optimize Topology**: Reduce routing hops
4. **Monitor Metrics**: Watch for bottlenecks

### Backup and Recovery

1. **Regular Exports**: Download configurations
2. **Version Control**: Track changes in Git
3. **Test Restores**: Verify backup procedures
4. **Document Procedures**: Maintain runbooks

## Keyboard Shortcuts

Improve efficiency with keyboard shortcuts:

| Shortcut | Action |
|----------|--------|
| `Ctrl+K` | Global search |
| `Ctrl+N` | New resource |
| `Ctrl+S` | Save changes |
| `Ctrl+D` | Duplicate resource |
| `Ctrl+/` | Show shortcuts |
| `Esc` | Close dialog |
| `?` | Help menu |

## Getting Help

If you need assistance:

1. **In-app Help**: Click the `?` icon
2. **Documentation**: Access from Help menu
3. **Support Tickets**: Submit via Help → Support
4. **Community**: Join our Slack channel
5. **Training**: Available video tutorials

## Conclusion

This guide covers the essential features of OVN Control Platform. As you become more familiar with the platform, explore advanced features like:

- API automation
- Bulk operations
- Custom integrations
- Advanced monitoring

For more information, refer to the [API Documentation](api-reference.md) and [Administrator Guide](admin-guide.md).