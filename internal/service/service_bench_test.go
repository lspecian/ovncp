package service

import (
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// Benchmark service operations
func BenchmarkService_ListLogicalSwitches(b *testing.B) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	// Prepare test data
	switches := make([]LogicalSwitch, 100)
	for i := range switches {
		switches[i] = LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "switch-" + string(rune(i)),
			Description: "Benchmark switch",
		}
	}
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.ListLogicalSwitches("user1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkService_GetLogicalSwitch(b *testing.B) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switchID := uuid.New().String()
	ls := &LogicalSwitch{
		UUID:        switchID,
		Name:        "bench-switch",
		Description: "Benchmark switch",
	}
	
	mockClient.On("GetLogicalSwitch", switchID).Return(ls, nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.GetLogicalSwitch("user1", switchID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkService_CreateLogicalSwitch(b *testing.B) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	mockClient.On("CreateLogicalSwitch", mock.Anything, mock.Anything).
		Return(&LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "new-switch",
			Description: "Created switch",
		}, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := CreateLogicalSwitchRequest{
			Name:        "bench-switch-" + string(rune(i)),
			Description: "Benchmark created switch",
		}
		
		_, err := svc.CreateLogicalSwitch("user1", req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_GetNetworkTopology(b *testing.B) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	// Prepare topology data
	switches := make([]LogicalSwitch, 50)
	for i := range switches {
		switches[i] = LogicalSwitch{
			UUID: uuid.New().String(),
			Name: "switch-" + string(rune(i)),
		}
	}
	
	routers := make([]LogicalRouter, 10)
	for i := range routers {
		routers[i] = LogicalRouter{
			UUID:    uuid.New().String(),
			Name:    "router-" + string(rune(i)),
			Enabled: true,
		}
	}
	
	ports := make([]LogicalPort, 100)
	for i := range ports {
		ports[i] = LogicalPort{
			UUID: uuid.New().String(),
			Name: "port-" + string(rune(i)),
		}
	}
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	mockClient.On("ListLogicalRouters").Return(routers, nil)
	mockClient.On("ListLogicalPorts").Return(ports, nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.GetNetworkTopology("user1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark concurrent operations
func BenchmarkService_ConcurrentMixedOperations(b *testing.B) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	// Setup mock responses
	switches := make([]LogicalSwitch, 10)
	for i := range switches {
		switches[i] = LogicalSwitch{
			UUID: uuid.New().String(),
			Name: "switch-" + string(rune(i)),
		}
	}
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	mockClient.On("GetLogicalSwitch", mock.Anything).Return(&switches[0], nil)
	mockClient.On("CreateLogicalSwitch", mock.Anything, mock.Anything).
		Return(&LogicalSwitch{UUID: uuid.New().String()}, nil)
	mockClient.On("UpdateLogicalSwitch", mock.Anything, mock.Anything).
		Return(&LogicalSwitch{UUID: uuid.New().String()}, nil)
	mockClient.On("DeleteLogicalSwitch", mock.Anything).Return(nil)
	
	operations := []func(){
		func() { svc.ListLogicalSwitches("user1") },
		func() { svc.GetLogicalSwitch("user1", switches[0].UUID) },
		func() {
			svc.CreateLogicalSwitch("user1", CreateLogicalSwitchRequest{
				Name: "new-switch",
			})
		},
		func() {
			svc.UpdateLogicalSwitch("user1", switches[0].UUID, UpdateLogicalSwitchRequest{
				Name: stringPtr("updated"),
			})
		},
		func() { svc.DeleteLogicalSwitch("user1", switches[0].UUID) },
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			op := operations[b.N%len(operations)]
			op()
		}
	})
}