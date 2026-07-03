package jobs

import "testing"

func TestExampleCreateTaskRoundTrip(t *testing.T) {
	task, err := NewExampleCreateTask("  async demo  ")
	if err != nil {
		t.Fatalf("NewExampleCreateTask() error = %v", err)
	}

	if task.Type() != TypeExampleCreate {
		t.Fatalf("task type = %s, want %s", task.Type(), TypeExampleCreate)
	}

	payload, err := ParseExampleCreateTask(task)
	if err != nil {
		t.Fatalf("ParseExampleCreateTask() error = %v", err)
	}

	if payload.Name != "async demo" {
		t.Fatalf("payload.Name = %q, want %q", payload.Name, "async demo")
	}
}

func TestExampleCreateTaskRejectsEmptyName(t *testing.T) {
	if _, err := NewExampleCreateTask("   "); err == nil {
		t.Fatal("expected error for empty task name")
	}
}
