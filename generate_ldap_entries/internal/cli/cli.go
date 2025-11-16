package cli

import (
	"encoding/json" // json is used to decode JSON input files
	"fmt"           // fmt is used to create readable error messages
	"os"            // os is used to open files from disk

	"github.com/alecthomas/kong" // kong is the library we use to parse command-line flags

	// This import path must match your module path from go.mod.
	"generate_ldap_entires/internal/generator"
)

///////////////////////////////////////////////////////////////////////////////
// CLI configuration
///////////////////////////////////////////////////////////////////////////////

// CLIConfig holds the command-line options for the fakeldap tool. Kong uses
// the struct tags to know which flags exist and how to parse them.
//
// Each field here will be turned into a flag, for example:
//
//	--suffix-dn, --count, --mode, --ldif-file, --ldap-url, --bind-dn, etc.
type CLIConfig struct {
	SuffixDN string `help:"DN suffix that comes after uid=<fakeuid>, e.g. 'ou=employee,ou=users,o=rtx'." required:"true" name:"suffix-dn"`
	Count    int    `help:"Number of fake entries to generate." default:"1"`

	Mode string `help:"Output mode: 'ldif' to write a file, 'ldap' to add entries to an LDAP server." default:"ldif"`

	LDIFFile string `help:"Path to LDIF file when mode is 'ldif'." default:"fake_users.ldif" name:"ldif-file"`

	LDAPURL      string `help:"LDAP URL when mode is 'ldap', e.g. 'ldaps://localhost:636'." name:"ldap-url"`
	BindDN       string `help:"Bind DN for LDAP when mode is 'ldap'." name:"bind-dn"`
	BindPassword string `help:"Bind password for LDAP when mode is 'ldap'." name:"bind-password"`

	InputFile string `help:"Optional JSON file that provides attribute values (uid, cn, sn, mail). Missing or empty fields are filled with fake data." name:"input-file"`
}

// NewCLIConfig is an initializer function for CLIConfig.
// It sets the same defaults we expect in the rest of the program so
// behavior stays consistent.
func NewCLIConfig() *CLIConfig {
	return &CLIConfig{
		Count:    1,
		Mode:     "ldif",
		LDIFFile: "fake_users.ldif",
	}
}

///////////////////////////////////////////////////////////////////////////////
// Template loading helper
///////////////////////////////////////////////////////////////////////////////

// loadTemplateFromFile reads a JSON file at the provided path and decodes it
// into a generator.AttributeTemplate. This lets the user define some or all
// attribute values in a separate file instead of writing them on the command
// line.
//
// The JSON file might look like:
//
//	{
//	  "uid": "jdoe",
//	  "cn": "John Doe",
//	  "sn": "Doe",
//	  "mail": "jdoe@example.com"
//	}
func loadTemplateFromFile(path string) (*generator.AttributeTemplate, error) {
	// Open the file for reading.
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %q: %w", path, err)
	}
	// Ensure the file is closed when we are done so the file descriptor
	// is not leaked.
	defer f.Close()

	// Create a new empty template using the generator's initializer function.
	tmpl := generator.NewAttributeTemplate()

	// Decode the JSON from the file into the template struct.
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(tmpl); err != nil {
		return nil, fmt.Errorf("failed to parse JSON in %q: %w", path, err)
	}

	return tmpl, nil
}

///////////////////////////////////////////////////////////////////////////////
// Top-level CLI runner
///////////////////////////////////////////////////////////////////////////////

// Run is the main entry point for the CLI layer. It:
//
//  1. Creates a CLIConfig and asks Kong to fill it from command-line flags.
//  2. Optionally loads a JSON template file if --input-file was provided.
//  3. Builds a generator.RunConfig and passes it to generator.Run.
//
// This keeps all CLI-related logic in one place and lets the generator
// package focus purely on generating and sending data.
func Run() error {
	// Create a new CLIConfig with defaults using the initializer.
	cfg := NewCLIConfig()

	// Parse flags into cfg. Kong reads os.Args automatically and fills the
	// struct fields based on the tags. If required flags are missing,
	// Kong will print a helpful message and exit the process.
	kctx := kong.Parse(cfg,
		kong.Name("fakeldap"),
		kong.Description("Generate fake LDAP entries and either write LDIF or send directly to LDAP."),
	)
	_ = kctx // we are not using kctx directly, but this avoids a compiler warning

	// Build a generator.RunConfig using values that came from the CLI.
	runCfg := generator.NewRunConfig()
	runCfg.SuffixDN = cfg.SuffixDN
	runCfg.Count = cfg.Count
	runCfg.Mode = cfg.Mode
	runCfg.LDIFFile = cfg.LDIFFile
	runCfg.LDAPURL = cfg.LDAPURL
	runCfg.BindDN = cfg.BindDN
	runCfg.BindPassword = cfg.BindPassword

	// If the user specified an input file, we load it into a template.
	// Any attributes included in this file will override the generated ones.
	if cfg.InputFile != "" {
		tmpl, err := loadTemplateFromFile(cfg.InputFile)
		if err != nil {
			return err
		}
		runCfg.Template = tmpl
	}

	// If the user selected "ldap" mode, we make sure they provided enough
	// information to actually connect and bind to the LDAP server.
	if runCfg.Mode == "ldap" {
		if runCfg.LDAPURL == "" || runCfg.BindDN == "" || runCfg.BindPassword == "" {
			return fmt.Errorf("mode 'ldap' requires --ldap-url, --bind-dn, and --bind-password")
		}
	}

	// Finally, we hand the fully-populated RunConfig to the generator
	// package, which will generate the entries and either write LDIF
	// or talk to LDAP, depending on Mode.
	return generator.Run(runCfg)
}
