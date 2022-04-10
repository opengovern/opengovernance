package auth

import (
	"fmt"
	"testing"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

func Test_hasAccess(t *testing.T) {
	type args struct {
		currRole api.Role
		minRole  api.Role
	}
	tests := []struct {
		args args
		want bool
	}{
		{
			args: args{
				currRole: api.AdminRole,
				minRole:  api.AdminRole,
			},
			want: true,
		},
		{
			args: args{
				currRole: api.ViewerRole,
				minRole:  api.ViewerRole,
			},
			want: true,
		},
		{
			args: args{
				currRole: api.EditorRole,
				minRole:  api.EditorRole,
			},
			want: true,
		},
		{
			args: args{
				currRole: api.AdminRole,
				minRole:  api.ViewerRole,
			},
			want: true,
		},
		{
			args: args{
				currRole: api.AdminRole,
				minRole:  api.EditorRole,
			},
			want: true,
		},
		{
			args: args{
				currRole: api.EditorRole,
				minRole:  api.AdminRole,
			},
			want: false,
		},
		{
			args: args{
				currRole: api.ViewerRole,
				minRole:  api.AdminRole,
			},
			want: false,
		},
		{
			args: args{
				currRole: api.EditorRole,
				minRole:  api.AdminRole,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		var name string
		if tt.want {
			name = fmt.Sprintf("User with role (%s) HAS access to minimum required role of (%s)", tt.args.currRole, tt.args.minRole)
		} else {
			name = fmt.Sprintf("User with role (%s) DOESN'T HAVE access to minimum required role of (%s)", tt.args.currRole, tt.args.minRole)
		}
		t.Run(name, func(t *testing.T) {
			if got := hasAccess(tt.args.currRole, tt.args.minRole); got != tt.want {
				t.Errorf("hasAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
