package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}).
			Background(lipgloss.AdaptiveColor{Light: "#5A67D8", Dark: "#7C3AED"}).
			PaddingLeft(2).
			PaddingRight(2)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#F3F4F6"}).
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1F2937"}).
				Background(lipgloss.AdaptiveColor{Light: "#3B82F6", Dark: "#60A5FA"}).
				PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).
			PaddingLeft(2)
)

type model struct {
	choices   []string
	cursor    int
	search    string
	filtered  []string
	projectsDir string
}

func initialModel(projectsDir string) model {
	projects, _ := getProjects(projectsDir)
	return model{
		choices:     projects,
		filtered:    projects,
		projectsDir: projectsDir,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selectedProject := m.filtered[m.cursor]
				projectPath := filepath.Join(m.projectsDir, selectedProject)
				fmt.Printf("cd %q", projectPath)
				return m, tea.Quit
			}
			return m, nil
		case "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "ctrl+n":
			if len(m.search) > 0 {
				today := time.Now().Format("2006-01-02")
				newProjectName := fmt.Sprintf("%s-%s", today, m.search)
				newProjectPath := filepath.Join(m.projectsDir, newProjectName)
				
				if err := os.MkdirAll(newProjectPath, 0755); err == nil {
					fmt.Printf("cd %q", newProjectPath)
					return m, tea.Quit
				}
			}
			return m, nil
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.filtered = filterProjects(m.choices, m.search)
				if m.cursor >= len(m.filtered) {
					m.cursor = len(m.filtered) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
				m.filtered = filterProjects(m.choices, m.search)
				m.cursor = 0
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render("try - Project Selector")
	s += "\n\n"
	
	if len(m.search) > 0 {
		s += fmt.Sprintf("Search: %s\n", m.search)
	}
	
	if len(m.filtered) == 0 {
		if len(m.search) > 0 {
			s += helpStyle.Render("No projects found. Press 'C-n' to create a new project with this name.")
		} else {
			s += helpStyle.Render("No projects found.")
		}
	} else {
		for i, choice := range m.filtered {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
				s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
			} else {
				s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
			}
			s += "\n"
		}
	}

	s += "\n" + helpStyle.Render("C-k/C-j: navigate • enter: select • C-n: new project • C-q: quit")
	return s
}

func getProjects(projectsDir string) ([]string, error) {
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		os.MkdirAll(projectsDir, 0755)
		return []string{}, nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}
	
	sort.Strings(projects)
	return projects, nil
}

func filterProjects(projects []string, search string) []string {
	if search == "" {
		return projects
	}
	
	var filtered []string
	searchLower := strings.ToLower(search)
	for _, project := range projects {
		if strings.Contains(strings.ToLower(project), searchLower) {
			filtered = append(filtered, project)
		}
	}
	return filtered
}

func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	
	if path == "~" {
		return os.UserHomeDir()
	}
	
	return "", fmt.Errorf("expansion of ~user paths not supported, use absolute path instead: %s", path)
}

func getProjectsDir() string {
	if dir := os.Getenv("TRY_PROJECTS_DIR"); dir != "" {
		expanded, err := expandPath(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding TRY_PROJECTS_DIR: %v\n", err)
			os.Exit(1)
		}
		return expanded
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "projects")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  init  Output shell function\n")
		fmt.Fprintf(os.Stderr, "  cd    Interactive project selector\n")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "init":
		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf(`try() {
    local output
    output="$(%q cd 2>/dev/tty)"
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        eval "$output"
    else
        echo "$output" >&2
        return $exit_code
    fi
}
`, execPath)
	case "cd":
		projectsDir := getProjectsDir()
		
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening /dev/tty: %v", err)
			os.Exit(1)
		}
		defer tty.Close()
		
		// Force TrueColor to ensure colors work properly in subshells
		lipgloss.SetColorProfile(termenv.TrueColor)
		
		p := tea.NewProgram(initialModel(projectsDir), 
			tea.WithAltScreen(), 
			tea.WithInput(tty),
			tea.WithOutput(tty))
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}