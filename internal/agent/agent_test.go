package agent

import "testing"

func TestValidateTask(t *testing.T) {
	ok := []string{"fix-login", "a", "Task_1", "123"}
	bad := []string{"", "has space", "a/b", "a.b", "a:b", "-lead", "_lead"}
	for _, s := range ok {
		if err := ValidateTask(s); err != nil {
			t.Errorf("ValidateTask(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateTask(s); err == nil {
			t.Errorf("ValidateTask(%q) expected error", s)
		}
	}
}
