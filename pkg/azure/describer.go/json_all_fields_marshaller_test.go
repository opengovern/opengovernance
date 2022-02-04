package describer

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/synapse/mgmt/2021-03-01/synapse"
	"github.com/gofrs/uuid"
)

func TestJSONAllFieldsMarshaller(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name: "Struct/Pointer",
			value: compute.VirtualMachine{
				ID:   String("MyVirtualMachine"),
				Type: String("MyVirtualMachineType"),
			},
			want: `{"id":"MyVirtualMachine","tags":null,"type":"MyVirtualMachineType"}`,
		},
		{
			name: "Struct/Pointer 2",
			value: compute.VirtualMachine{
				ID:   String("MyVirtualMachine"),
				Type: String("MyVirtualMachineType"),
				Plan: &compute.Plan{
					Name:      String("MyPlan"),
					Publisher: String("MyPublisher"),
				},
			},
			want: `{"id":"MyVirtualMachine","plan":{"name":"MyPlan","publisher":"MyPublisher"},"tags":null,"type":"MyVirtualMachineType"}`,
		},
		{
			name: "Struct/Pointer/Slice",
			value: compute.VirtualMachine{
				ID:   String("MyVirtualMachine"),
				Type: String("MyVirtualMachineType"),
				Plan: &compute.Plan{
					Name:      String("MyPlan"),
					Publisher: String("MyPublisher"),
				},
				Resources: &[]compute.VirtualMachineExtension{
					{
						ID: String("MyVirtualMachineExtension"),
					},
				},
			},
			want: `{"id":"MyVirtualMachine","plan":{"name":"MyPlan","publisher":"MyPublisher"},"resources":[{"id":"MyVirtualMachineExtension","tags":null}],"tags":null,"type":"MyVirtualMachineType"}`,
		},
		{
			name: "Array/Slice",
			value: compute.VirtualMachine{
				ID:   String("MyVirtualMachine"),
				Type: String("MyVirtualMachineType"),
				Resources: &[]compute.VirtualMachineExtension{
					{
						ID:   String("MyVirtualMachineExtension"),
						Name: String("MyVirtualMachineExtensionName"),
						VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
							Publisher: String("MyPublisher"),
						},
					},
				},
			},
			want: `{"id":"MyVirtualMachine","resources":[{"id":"MyVirtualMachineExtension","name":"MyVirtualMachineExtensionName","properties":{"publisher":"MyPublisher"},"tags":null}],"tags":null,"type":"MyVirtualMachineType"}`,
		},
		{
			name: "UUID",
			value: synapse.Workspace{
				ID: String("MyWorkspace"),
				WorkspaceProperties: &synapse.WorkspaceProperties{
					WorkspaceUID: UUID(uuid.Must(uuid.FromString("7eae5af9-b353-4d53-89b6-15a1a664b2c2"))),
				},
			},
			want: `{"id":"MyWorkspace","properties":{"connectivityEndpoints":null,"extraProperties":null,"workspaceUID":"7eae5af9-b353-4d53-89b6-15a1a664b2c2"},"tags":null}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := JSONAllFieldsMarshaller{
				Value: tt.value,
			}
			got, err := x.MarshalJSON()
			if err != nil {
				t.Errorf("JSONAllFieldsMarshaller.MarshalJSON() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("JSONAllFieldsMarshaller.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func String(s string) *string {
	return &s
}

func UUID(u uuid.UUID) *uuid.UUID {
	return &u
}
