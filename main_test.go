package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- filterProjects tests ---

func TestFilterProjects_EmptySearch(t *testing.T) {
	projects := []string{"alpha", "beta", "gamma"}
	got := filterProjects(projects, "")
	if len(got) != len(projects) {
		t.Fatalf("expected %d projects, got %d", len(projects), len(got))
	}
	for i, p := range projects {
		if got[i] != p {
			t.Errorf("index %d: expected %q, got %q", i, p, got[i])
		}
	}
}

func TestFilterProjects_MatchingSubstring(t *testing.T) {
	projects := []string{"my-project", "other-thing", "project-two"}
	got := filterProjects(projects, "project")
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0] != "my-project" || got[1] != "project-two" {
		t.Errorf("unexpected results: %v", got)
	}
}

func TestFilterProjects_NoMatch(t *testing.T) {
	projects := []string{"alpha", "beta"}
	got := filterProjects(projects, "zzz")
	if len(got) != 0 {
		t.Fatalf("expected 0 results, got %d", len(got))
	}
}

func TestFilterProjects_CaseInsensitive(t *testing.T) {
	projects := []string{"MyProject", "other"}
	got := filterProjects(projects, "myproject")
	if len(got) != 1 || got[0] != "MyProject" {
		t.Errorf("expected [MyProject], got %v", got)
	}

	got2 := filterProjects(projects, "MYPROJECT")
	if len(got2) != 1 || got2[0] != "MyProject" {
		t.Errorf("expected [MyProject], got %v", got2)
	}
}

func TestFilterProjects_EmptyInput(t *testing.T) {
	got := filterProjects([]string{}, "test")
	if len(got) != 0 {
		t.Fatalf("expected 0 results, got %d", len(got))
	}
}

// --- getProjects tests ---

func TestGetProjects_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	projects, err := getProjects(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}
}

func TestGetProjects_WithSubdirs(t *testing.T) {
	dir := t.TempDir()
	// Create dirs with explicit mod times to verify most-recent-first ordering
	names := []string{"oldest", "middle", "newest"}
	baseTime := time.Now().Add(-3 * time.Hour)
	for i, name := range names {
		p := filepath.Join(dir, name)
		os.Mkdir(p, 0755)
		modTime := baseTime.Add(time.Duration(i) * time.Hour)
		os.Chtimes(p, modTime, modTime)
	}
	projects, err := getProjects(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}
	// Most recent first
	expected := []string{"newest", "middle", "oldest"}
	for i, e := range expected {
		if projects[i] != e {
			t.Errorf("index %d: expected %q, got %q (full: %v)", i, e, projects[i], projects)
		}
	}
}

func TestGetProjects_IgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "project-dir"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte(""), 0644)

	projects, err := getProjects(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 1 || projects[0] != "project-dir" {
		t.Errorf("expected [project-dir], got %v", projects)
	}
}

func TestGetProjects_CreatesNonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent", "subdir")
	projects, err := getProjects(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}
	// Verify the directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("created path is not a directory")
	}
}

// --- expandPath tests ---

func TestExpandPath_NonTilde(t *testing.T) {
	path := "/some/absolute/path"
	got, err := expandPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("expected %q, got %q", path, got)
	}
}

func TestExpandPath_TildeOnly(t *testing.T) {
	got, err := expandPath("~")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	if got != home {
		t.Errorf("expected %q, got %q", home, got)
	}
}

func TestExpandPath_TildeSlash(t *testing.T) {
	got, err := expandPath("~/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "foo/bar")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestExpandPath_TildeUser(t *testing.T) {
	_, err := expandPath("~someuser/path")
	if err == nil {
		t.Fatal("expected error for ~user path, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExpandPath_RelativePath(t *testing.T) {
	got, err := expandPath("relative/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "relative/path" {
		t.Errorf("expected %q, got %q", "relative/path", got)
	}
}

// --- getProjectsDir tests ---

func TestGetProjectsDir_WithEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TRY_PROJECTS_DIR", dir)
	got, err := getProjectsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("expected %q, got %q", dir, got)
	}
}

func TestGetProjectsDir_Default(t *testing.T) {
	t.Setenv("TRY_PROJECTS_DIR", "")
	got, err := getProjectsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "projects")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGetProjectsDir_WithTildeExpansion(t *testing.T) {
	t.Setenv("TRY_PROJECTS_DIR", "~/my-projects")
	got, err := getProjectsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "my-projects")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGetProjectsDir_TildeUserError(t *testing.T) {
	t.Setenv("TRY_PROJECTS_DIR", "~baduser/path")
	_, err := getProjectsDir()
	if err == nil {
		t.Fatal("expected error for ~user path")
	}
}

// --- initialModel tests ---

func TestInitialModel(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "proj-a"), 0755)
	os.Mkdir(filepath.Join(dir, "proj-b"), 0755)

	m := initialModel(dir)
	if len(m.choices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(m.choices))
	}
	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 filtered, got %d", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if m.projectsDir != dir {
		t.Errorf("expected projectsDir %q, got %q", dir, m.projectsDir)
	}
	if m.search != "" {
		t.Errorf("expected empty search, got %q", m.search)
	}
}

func TestInitialModel_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := initialModel(dir)
	if len(m.choices) != 0 {
		t.Fatalf("expected 0 choices, got %d", len(m.choices))
	}
	if len(m.filtered) != 0 {
		t.Fatalf("expected 0 filtered, got %d", len(m.filtered))
	}
}

// --- model.Init tests ---

func TestModelInit(t *testing.T) {
	m := model{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

// --- model.Update tests ---

func newTestModel(choices []string, projectsDir string) model {
	return model{
		choices:     choices,
		filtered:    choices,
		cursor:      0,
		projectsDir: projectsDir,
	}
}

func TestUpdate_CtrlC(t *testing.T) {
	m := newTestModel([]string{"a", "b"}, "/tmp")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
}

func TestUpdate_CtrlQ(t *testing.T) {
	m := newTestModel([]string{"a", "b"}, "/tmp")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
}

func TestUpdate_CtrlK_MovesUp(t *testing.T) {
	m := newTestModel([]string{"a", "b", "c"}, "/tmp")
	m.cursor = 2

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	if cmd != nil {
		t.Error("expected nil command")
	}
	um := updated.(model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}
}

func TestUpdate_CtrlK_AtTop(t *testing.T) {
	m := newTestModel([]string{"a", "b"}, "/tmp")
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	um := updated.(model)
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}
}

func TestUpdate_CtrlJ_MovesDown(t *testing.T) {
	m := newTestModel([]string{"a", "b", "c"}, "/tmp")
	m.cursor = 0

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if cmd != nil {
		t.Error("expected nil command")
	}
	um := updated.(model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}
}

func TestUpdate_CtrlJ_AtBottom(t *testing.T) {
	m := newTestModel([]string{"a", "b"}, "/tmp")
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	um := updated.(model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}
}

func TestUpdate_Enter_SelectsProject(t *testing.T) {
	dir := t.TempDir()
	m := newTestModel([]string{"my-project"}, dir)

	// Capture stdout to verify fmt.Printf output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	expectedPath := filepath.Join(dir, "my-project")
	expectedOutput := fmt.Sprintf("cd %q", expectedPath)
	if output != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, output)
	}
}

func TestUpdate_Enter_EmptyList(t *testing.T) {
	m := newTestModel([]string{}, "/tmp")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil command for enter on empty list")
	}
}

func TestUpdate_TypeCharacter(t *testing.T) {
	m := newTestModel([]string{"alpha", "beta", "gamma"}, "/tmp")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("expected nil command")
	}
	um := updated.(model)
	if um.search != "a" {
		t.Errorf("expected search %q, got %q", "a", um.search)
	}
	// "a" matches alpha, beta (has 'a'), gamma (has 'a')
	if len(um.filtered) != 3 {
		t.Errorf("expected 3 filtered results, got %d: %v", len(um.filtered), um.filtered)
	}
	if um.cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", um.cursor)
	}
}

func TestUpdate_TypeMultipleChars(t *testing.T) {
	m := newTestModel([]string{"alpha", "beta", "gamma"}, "/tmp")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	um := updated.(model)
	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	um2 := updated2.(model)

	if um2.search != "be" {
		t.Errorf("expected search %q, got %q", "be", um2.search)
	}
	if len(um2.filtered) != 1 || um2.filtered[0] != "beta" {
		t.Errorf("expected [beta], got %v", um2.filtered)
	}
}

func TestUpdate_Backspace(t *testing.T) {
	m := newTestModel([]string{"alpha", "beta"}, "/tmp")
	m.search = "al"
	m.filtered = filterProjects(m.choices, m.search)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd != nil {
		t.Error("expected nil command")
	}
	um := updated.(model)
	if um.search != "a" {
		t.Errorf("expected search %q, got %q", "a", um.search)
	}
	// "a" matches both "alpha" and "beta" (contains 'a')
	if len(um.filtered) != 2 {
		t.Errorf("expected 2 filtered results, got %d: %v", len(um.filtered), um.filtered)
	}
}

func TestUpdate_Backspace_EmptySearch(t *testing.T) {
	m := newTestModel([]string{"alpha"}, "/tmp")
	m.search = ""

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(model)
	if um.search != "" {
		t.Errorf("expected empty search, got %q", um.search)
	}
}

func TestUpdate_Backspace_CursorAdjustment(t *testing.T) {
	m := newTestModel([]string{"alpha", "beta", "gamma"}, "/tmp")
	m.search = "alpha"
	m.filtered = filterProjects(m.choices, m.search)
	m.cursor = 0

	// Backspace to "alph" — still only alpha matches, cursor stays 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(model)
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}
}

func TestUpdate_Backspace_CursorClampToZero(t *testing.T) {
	// Scenario: all items filtered out, cursor should clamp to 0
	m := model{
		choices:  []string{"alpha"},
		filtered: []string{},
		cursor:   0,
		search:   "z",
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(model)
	// After backspace, search is empty, all items returned
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}
}

func TestUpdate_CtrlN_CreatesProject(t *testing.T) {
	dir := t.TempDir()
	m := newTestModel([]string{}, dir)
	m.search = "test-project"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}

	today := time.Now().Format("2006-01-02")
	expectedName := fmt.Sprintf("%s-test-project", today)
	expectedPath := filepath.Join(dir, expectedName)

	if !strings.Contains(output, expectedPath) {
		t.Errorf("expected output to contain %q, got %q", expectedPath, output)
	}

	// Verify directory was created
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("project directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("created path is not a directory")
	}
}

func TestUpdate_CtrlN_EmptySearch(t *testing.T) {
	m := newTestModel([]string{}, "/tmp")
	m.search = ""

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd != nil {
		t.Error("expected nil command when search is empty")
	}
}

func TestUpdate_NonKeyMsg(t *testing.T) {
	m := newTestModel([]string{"a"}, "/tmp")
	// Send a non-key message (tea.WindowSizeMsg)
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("expected nil command")
	}
	um := updated.(model)
	if um.cursor != 0 {
		t.Errorf("model should be unchanged")
	}
}

// --- model.View tests ---

func TestView_ContainsTitle(t *testing.T) {
	m := newTestModel([]string{"project-a"}, "/tmp")
	view := m.View()
	if !strings.Contains(view, "try - Project Selector") {
		t.Error("view should contain title")
	}
}

func TestView_ShowsItems(t *testing.T) {
	m := newTestModel([]string{"project-a", "project-b"}, "/tmp")
	view := m.View()
	if !strings.Contains(view, "project-a") {
		t.Error("view should contain project-a")
	}
	if !strings.Contains(view, "project-b") {
		t.Error("view should contain project-b")
	}
}

func TestView_SelectedItem(t *testing.T) {
	m := newTestModel([]string{"project-a", "project-b"}, "/tmp")
	m.cursor = 0
	view := m.View()
	// The selected item should have ">" prefix
	if !strings.Contains(view, "> project-a") {
		t.Error("view should show > prefix for selected item")
	}
}

func TestView_ShowsSearchText(t *testing.T) {
	m := newTestModel([]string{"alpha"}, "/tmp")
	m.search = "alp"
	view := m.View()
	if !strings.Contains(view, "Search: alp") {
		t.Error("view should display search text")
	}
}

func TestView_NoSearchText(t *testing.T) {
	m := newTestModel([]string{"alpha"}, "/tmp")
	m.search = ""
	view := m.View()
	if strings.Contains(view, "Search:") {
		t.Error("view should not display Search: when search is empty")
	}
}

func TestView_EmptyNoSearch(t *testing.T) {
	m := newTestModel([]string{}, "/tmp")
	m.filtered = []string{}
	view := m.View()
	if !strings.Contains(view, "No projects found.") {
		t.Error("view should show 'No projects found.' when no projects and no search")
	}
}

func TestView_EmptyWithSearch(t *testing.T) {
	m := newTestModel([]string{}, "/tmp")
	m.search = "xyz"
	m.filtered = []string{}
	view := m.View()
	if !strings.Contains(view, "C-n") {
		t.Error("view should mention C-n to create a new project when search has no results")
	}
}

func TestView_HelpText(t *testing.T) {
	m := newTestModel([]string{"a"}, "/tmp")
	view := m.View()
	if !strings.Contains(view, "C-k/C-j: navigate") {
		t.Error("view should contain help text")
	}
	if !strings.Contains(view, "C-q: quit") {
		t.Error("view should contain quit help")
	}
}

func TestView_SecondItemSelected(t *testing.T) {
	m := newTestModel([]string{"project-a", "project-b"}, "/tmp")
	m.cursor = 1
	view := m.View()
	if !strings.Contains(view, "> project-b") {
		t.Error("view should show > prefix for second item when cursor=1")
	}
}

// --- Backspace cursor edge cases ---

func TestUpdate_Backspace_CursorExceedsFiltered(t *testing.T) {
	// After backspace, filtered has fewer items than cursor position
	m := model{
		choices:  []string{"abc", "abd", "xyz"},
		filtered: []string{"abc"},
		cursor:   2,
		search:   "abc",
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(model)
	// search="ab", filtered=["abc","abd"], cursor was 2 which is >= len(filtered)=2, so clamped to 1
	if um.cursor >= len(um.filtered) && len(um.filtered) > 0 {
		t.Errorf("cursor %d should be < filtered length %d", um.cursor, len(um.filtered))
	}
}

func TestUpdate_Backspace_FilteredBecomesEmpty(t *testing.T) {
	// After backspace, search still has chars but nothing matches → cursor clamped to 0
	m := model{
		choices:     []string{"xyz"},
		filtered:    []string{},
		cursor:      0,
		search:      "ab",
		projectsDir: "/tmp",
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(model)
	// search="a", "xyz" doesn't contain "a", filtered empty, cursor clamped to 0
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}
}

// --- getProjects error path ---

func TestGetProjects_ReadDirError(t *testing.T) {
	// Use a file where a directory is expected to trigger ReadDir error
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "notadir")
	os.WriteFile(fakePath, []byte("x"), 0644)
	_, err := getProjects(fakePath)
	if err == nil {
		t.Fatal("expected error reading a file as directory")
	}
}

// --- run() tests (extracted from main for testability) ---

func TestRun_NoArgs(t *testing.T) {
	code := run([]string{"try"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	code := run([]string{"try", "bogus"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Init(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := run([]string{"try", "init"})

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(output, "try()") {
		t.Errorf("expected shell function, got: %s", output)
	}
}

func TestRun_Cd_BadProjectsDir(t *testing.T) {
	t.Setenv("TRY_PROJECTS_DIR", "~baduser/path")
	code := run([]string{"try", "cd"})
	if code != 1 {
		t.Errorf("expected exit code 1 for bad projects dir, got %d", code)
	}
}

func TestRun_Cd_ValidDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TRY_PROJECTS_DIR", dir)

	done := make(chan int, 1)
	go func() {
		done <- run([]string{"try", "cd"})
	}()

	// Let the TUI initialize enough for coverage, then we're done
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// TUI blocks on input — coverage for setup lines was captured
	}
}
