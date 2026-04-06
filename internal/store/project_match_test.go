package store

import "testing"

func TestProjectMatchKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"element-adds", "elementadds"},
		{"elementAdds", "elementadds"},
		{"Element_Adds", "elementadds"},
		{"  my app  ", "myapp"},
		{"my\tapp\nname", "myappname"},
		{"my.app/v2", "my.app/v2"},
		{"", ""},
		{"!!!", "!!!"},
	}
	for _, tt := range tests {
		if got := ProjectMatchKey(tt.in); got != tt.want {
			t.Errorf("ProjectMatchKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
