package plan

import (
	"testing"
)

func TestParseDepend(t *testing.T) {
	tests := []struct {
		input   string
		want    *Depend
		wantErr bool
	}{
		{
			input: "users",
			want:  &Depend{Change: "users"},
		},
		{
			input: "users@v1.0.0",
			want:  &Depend{Change: "users", Tag: "v1.0.0"},
		},
		{
			input: "other:users",
			want:  &Depend{Project: "other", Change: "users"},
		},
		{
			input: "other:users@v1.0.0",
			want:  &Depend{Project: "other", Change: "users", Tag: "v1.0.0"},
		},
		{
			input: "!users",
			want:  &Depend{Change: "users", IsConflict: true},
		},
		{
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDepend(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDepend(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Project != tt.want.Project {
				t.Errorf("Project = %q, want %q", got.Project, tt.want.Project)
			}
			if got.Change != tt.want.Change {
				t.Errorf("Change = %q, want %q", got.Change, tt.want.Change)
			}
			if got.Tag != tt.want.Tag {
				t.Errorf("Tag = %q, want %q", got.Tag, tt.want.Tag)
			}
			if got.IsConflict != tt.want.IsConflict {
				t.Errorf("IsConflict = %v, want %v", got.IsConflict, tt.want.IsConflict)
			}
		})
	}
}

func TestDependString(t *testing.T) {
	tests := []struct {
		dep  *Depend
		want string
	}{
		{
			dep:  &Depend{Change: "users"},
			want: "users",
		},
		{
			dep:  &Depend{Change: "users", Tag: "v1.0.0"},
			want: "users@v1.0.0",
		},
		{
			dep:  &Depend{Project: "other", Change: "users"},
			want: "other:users",
		},
		{
			dep:  &Depend{Change: "users", IsConflict: true},
			want: "!users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.dep.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
