/*
Copyright (c) Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the \"License\");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an \"AS IS\" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"slices"
	"testing"

	resourceapi "k8s.io/api/resource/v1"
)

func deviceNames(devices []resourceapi.Device) []string {
	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = d.Name
	}
	return names
}

// assertDeviceOrder repeats the call because map iteration order is not
// defined, so a single call could pass by chance without the sort.
func assertDeviceOrder(t *testing.T, allocatable AllocatableDevices, want []string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		got := deviceNames(resourceSliceDevices(allocatable))
		if !slices.Equal(got, want) {
			t.Fatalf("device order mismatch on call %d: got %v, want %v", i, got, want)
		}
	}
}

func gpu(card, render, numa int) *AllocatableDevice {
	return &AllocatableDevice{AmdGpu: &AmdGpuInfo{cardIndex: card, renderIndex: render, NumaNode: numa}}
}

func partition(card, render, numa int) *AllocatableDevice {
	return &AllocatableDevice{AmdPartition: &AmdPartitionInfo{Parent: &AmdGpuInfo{}, cardIndex: card, renderIndex: render, NumaNode: numa}}
}

func TestResourceSliceDevicesAreSortedByCardIndex(t *testing.T) {
	// Card indices compare as numbers: gpu-3 comes before gpu-11, which a
	// string comparison of the names would reverse.
	assertDeviceOrder(t, AllocatableDevices{
		"gpu-9-136":  gpu(9, 136, 0),
		"gpu-1-128":  gpu(1, 128, 0),
		"gpu-17-144": gpu(17, 144, 0),
		"gpu-3-130":  gpu(3, 130, 0),
		"gpu-11-138": gpu(11, 138, 0),
	}, []string{"gpu-1-128", "gpu-3-130", "gpu-9-136", "gpu-11-138", "gpu-17-144"})
}

func TestResourceSliceDevicesAreGroupedByNUMANode(t *testing.T) {
	tests := []struct {
		name        string
		allocatable AllocatableDevices
		want        []string
	}{
		{
			// An MI308X node: card indices step by 8, so sorting the names as
			// strings puts gpu-9 last and splits NUMA node 0.
			name: "MI308X card layout",
			allocatable: AllocatableDevices{
				"gpu-1-128": gpu(1, 128, 0), "gpu-9-136": gpu(9, 136, 0),
				"gpu-17-144": gpu(17, 144, 0), "gpu-25-152": gpu(25, 152, 0),
				"gpu-33-160": gpu(33, 160, 1), "gpu-41-168": gpu(41, 168, 1),
				"gpu-49-176": gpu(49, 176, 1), "gpu-57-184": gpu(57, 184, 1),
			},
			want: []string{
				"gpu-1-128", "gpu-9-136", "gpu-17-144", "gpu-25-152",
				"gpu-33-160", "gpu-41-168", "gpu-49-176", "gpu-57-184",
			},
		},
		{
			name: "NUMA node takes precedence over card index",
			allocatable: AllocatableDevices{
				"gpu-1-128": gpu(1, 128, 1), "gpu-2-129": gpu(2, 129, 1),
				"gpu-3-130": gpu(3, 130, 0), "gpu-4-131": gpu(4, 131, 0),
			},
			want: []string{"gpu-3-130", "gpu-4-131", "gpu-1-128", "gpu-2-129"},
		},
		{
			name: "partitions",
			allocatable: AllocatableDevices{
				"gpu-10-138": partition(10, 138, 1), "gpu-11-139": partition(11, 139, 1),
				"gpu-20-148": gpu(20, 148, 0),
			},
			want: []string{"gpu-20-148", "gpu-10-138", "gpu-11-139"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeviceOrder(t, tt.allocatable, tt.want)
		})
	}
}
