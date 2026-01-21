package command

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/plan"
)

var addCmd = &cobra.Command{
	Use:   "add <change>",
	Short: "Add a new change to the plan",
	Long: `Add a new change to the plan file and create the deploy, revert,
and verify script files from templates.`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	addRequires          []string
	addConflicts         []string
	addNote              string
	addTemplateDirectory string
)

func init() {
	addCmd.Flags().StringSliceVarP(&addRequires, "requires", "r", nil, "Specify changes this change requires")
	addCmd.Flags().StringSliceVarP(&addConflicts, "conflicts", "c", nil, "Specify changes this change conflicts with")
	addCmd.Flags().StringVarP(&addNote, "note", "n", "", "A brief description of the change")
	addCmd.Flags().StringVar(&addTemplateDirectory, "template-directory", "", "Custom template directory")
}

func runAdd(cmd *cobra.Command, args []string) error {
	changeName := args[0]

	// Validate change name
	if err := validateChangeName(changeName); err != nil {
		return err
	}

	// Load plan
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'sqitch init' first", planPath)
		}
		return err
	}

	// Check for duplicate
	if p.HasChange(changeName) {
		return fmt.Errorf("change '%s' already exists in the plan", changeName)
	}

	// Parse dependencies
	var requires, conflicts []*plan.Depend
	for _, r := range addRequires {
		d, err := plan.ParseDepend(r)
		if err != nil {
			return fmt.Errorf("invalid requires: %w", err)
		}
		requires = append(requires, d)
	}
	for _, c := range addConflicts {
		d, err := plan.ParseDepend(c)
		if err != nil {
			return fmt.Errorf("invalid conflicts: %w", err)
		}
		d.IsConflict = true
		conflicts = append(conflicts, d)
	}

	note := addNote
	if note == "" {
		var err error
		note, err = getNoteFromEditor(changeName)
		if err != nil {
			return err
		}
	}

	// Create change
	change := &plan.Change{
		Name:         changeName,
		Timestamp:    time.Now().UTC(),
		PlannerName:  sqitch.UserName,
		PlannerEmail: sqitch.UserEmail,
		Requires:     requires,
		Conflicts:    conflicts,
		Note:         note,
	}

	// Add to plan
	p.AddChange(change)

	// Write plan
	if err := p.WriteFile(planPath); err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}
	sqitch.UI.Info("Added \"%s\" to %s", changeName, planPath)

	// Create script files
	ext := sqitch.Extension()
	scripts := []struct {
		dir      string
		template string
	}{
		{sqitch.DeployDir(), deployTemplate},
		{sqitch.RevertDir(), revertTemplate},
		{sqitch.VerifyDir(), verifyTemplate},
	}

	tmplData := templateData{
		Change:   changeName,
		Project:  p.Project,
		Author:   fmt.Sprintf("%s <%s>", sqitch.UserName, sqitch.UserEmail),
		Date:     change.Timestamp.Format("2006-01-02"),
		Requires: formatRequiresList(requires),
		Note:     note,
	}

	for _, s := range scripts {
		scriptPath := filepath.Join(s.dir, changeName+"."+ext)

		// Check if file already exists
		if _, err := os.Stat(scriptPath); err == nil {
			sqitch.UI.Comment("%s already exists", scriptPath)
			continue
		}

		// Create directory if needed
		if err := os.MkdirAll(s.dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write script from template
		if err := writeScript(scriptPath, s.template, tmplData); err != nil {
			return fmt.Errorf("failed to create %s: %w", scriptPath, err)
		}
		sqitch.UI.Info("Created %s", scriptPath)
	}

	return nil
}

func validateChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("change name cannot be empty")
	}

	// Check for invalid characters
	invalid := []string{" ", "\t", "\n", "@", "#", "%", ":", "[", "]"}
	for _, c := range invalid {
		if strings.Contains(name, c) {
			return fmt.Errorf("change name cannot contain '%s'", c)
		}
	}

	return nil
}

type templateData struct {
	Change   string
	Project  string
	Author   string
	Date     string
	Requires string
	Note     string
}

func writeScript(path, tmplStr string, data templateData) error {
	tmpl, err := template.New("script").Parse(tmplStr)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func formatRequiresList(deps []*plan.Depend) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = d.String()
	}
	return strings.Join(parts, ", ")
}

func getNoteFromEditor(changeName string) (string, error) {
	ext := sqitch.Extension()
	scripts := []string{
		filepath.Join("deploy", changeName+"."+ext),
		filepath.Join("revert", changeName+"."+ext),
		filepath.Join("verify", changeName+"."+ext),
	}

	template := fmt.Sprintf(`

# Please enter a note for your change. Lines starting with '#' will
# be ignored, and an empty message aborts the add.
# Change to add:
#
#   %s
#     %s
#     %s
#     %s
#
`, changeName, scripts[0], scripts[1], scripts[2])

	tmpFile, err := os.CreateTemp("", "sqitch-note-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(template); err != nil {
		return "", err
	}
	tmpFile.Close()

	editor := sqitch.Editor()
	parts := strings.Fields(editor)
	parts = append(parts, tmpFile.Name())

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	f, err := os.Open(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer f.Close()

	var result []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}
	}

	note := strings.TrimSpace(strings.Join(result, "\n"))
	if note == "" {
		return "", fmt.Errorf("aborting due to empty note")
	}

	return note, nil
}

var deployTemplate = `-- Deploy {{.Project}}:{{.Change}} to {{.Project}}
-- requires: {{.Requires}}

BEGIN;

-- XXX Add DDL here.

COMMIT;
`

var revertTemplate = `-- Revert {{.Project}}:{{.Change}} from {{.Project}}

BEGIN;

-- XXX Add DDL here.

COMMIT;
`

var verifyTemplate = `-- Verify {{.Project}}:{{.Change}} on {{.Project}}

BEGIN;

-- XXX Add verification code here.

ROLLBACK;
`
