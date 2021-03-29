package commons

import (
	"testing"
)

func TestSingleGeneralMemoryView(t *testing.T) {
	correctViews := []struct {
		s    string
		want uint8
	}{
		{"", 0},
		{"foo:U", U_VAL},
		{"foo:P", P_VAL},
		{"foo:R", R_VAL},
		{"foo:RX", R_VAL | X_VAL},
		{"foo:XR", X_VAL | R_VAL},
		{"foo:RW", R_VAL | W_VAL},
		{"foo:WR", W_VAL | R_VAL},
		{"foo:RWX", R_VAL | W_VAL | X_VAL},
		{"foo:WRX", R_VAL | W_VAL | X_VAL},
		{"foo:XWR", R_VAL | W_VAL | X_VAL},
	}
	incorrectViews := []string{
		":",
		"foo",
		"foo:",
		":R",
		":W",
		":X",
		":U",
		":P",
		"foo:RP",
		"foo:WX",
		"foo:W",
		"foo:UR",
	}

	for _, c := range correctViews {
		res, err := ParseMemoryView(c.s)
		if err != nil {
			t.Errorf(err.Error())
		}
		if res != nil && len(res) == 0 && err == nil {
			continue
		}
		if len(res) != 1 {
			t.Errorf("Invalid len %v\n", len(res))
		}
		if res[0].Perm != c.want || res[0].Name != "foo" {
			t.Errorf("Invalid value %v, got %v\n", c, res)
		}
	}

	for _, c := range incorrectViews {
		_, err := ParseMemoryView(c)
		if err == nil {
			t.Errorf("Failed to catch bad entry %v\n", c)
		}
	}
}

func TestMultipleMemoryViews(t *testing.T) {
	correct := []struct {
		s     string
		l     int
		name  []string
		perms []uint8
	}{
		{"foo:R,bar:RW,farr:RX,dar:RWX", 4,
			[]string{"foo", "bar", "farr", "dar"},
			[]uint8{R_VAL, R_VAL | W_VAL, R_VAL | X_VAL, R_VAL | X_VAL | W_VAL},
		},
	}
	incorrect := []string{
		"foo:RWX,bar:RX,foo:RWX",
		"foo: RWX,,bitch:RX",
	}

	for _, c := range correct {
		res, err := ParseMemoryView(c.s)
		if err != nil {
			t.Errorf(err.Error())
		}
		if len(res) != c.l {
			t.Errorf("Wrong length, got %v expected %v\n", len(res), c.l)
		}
		for i := range res {
			if res[i].Name != c.name[i] {
				t.Errorf("Wrong name, got %v expected %v\n", res[i].Name, c.name[i])
			}
			if res[i].Perm != c.perms[i] {
				t.Errorf("Wrong perm, got %v expected %v\n", res[i].Perm, c.perms[i])
			}
		}
	}

	for _, c := range incorrect {
		_, err := ParseMemoryView(c)
		if err == nil {
			t.Errorf("Entry should have triggered error %v\n", c)
		}
	}
}
