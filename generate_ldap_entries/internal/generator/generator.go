package generator

import (
	"crypto/tls" // tls is used to configure secure LDAP connections
	"fmt"        // fmt is used to build human-readable error messages and strings
	"os"         // os is used for file operations such as writing LDIF files

	"github.com/brianvoe/gofakeit/v6" // gofakeit generates realistic-looking fake data
	"github.com/go-ldap/ldap/v3"      // ldap/v3 provides LDAP client and entry types
	ldif "github.com/go-ldap/ldif"    // ldif converts LDAP entries to LDIF text
)

///////////////////////////////////////////////////////////////////////////////
// Configuration types
///////////////////////////////////////////////////////////////////////////////

// RunConfig holds all the options needed to generate fake entries and decide
// whether to write them to an LDIF file or send them directly to an LDAP
// server. This struct is designed to be independent of the CLI library
// (Kong) so it can be reused in tests or from other callers.
type RunConfig struct {
	SuffixDN     string             // SuffixDN is everything after "uid=<id>,", for example "ou=employee,ou=users,o=rtx"
	Count        int                // Count is how many fake entries to generate
	Mode         string             // Mode selects behavior: "ldif" or "ldap"
	LDIFFile     string             // LDIFFile is the path to write LDIF to when Mode == "ldif"
	LDAPURL      string             // LDAPURL is the LDAP server URL for Mode == "ldap", e.g. "ldaps://localhost:636"
	BindDN       string             // BindDN is the DN used to authenticate to the LDAP server
	BindPassword string             // BindPassword is the password used with BindDN
	Template     *AttributeTemplate // Template holds optional attribute values loaded from a file
}

// NewRunConfig is an initializer function for RunConfig.
// It sets safe default values so the caller only has to override
// what they care about.
func NewRunConfig() *RunConfig {
	return &RunConfig{
		Count:    1,                 // default to one generated entry
		Mode:     "ldif",            // default to writing LDIF instead of hitting LDAP
		LDIFFile: "fake_users.ldif", // default LDIF output file path
	}
}

// AttributeTemplate represents optional attribute values that can be read
// from a user-provided JSON file. Each field can override the corresponding
// generated value. If a field is empty, the generator will supply a fake
// value using gofakeit.
type AttributeTemplate struct {
	UID  string `json:"uid"`  // UID read from file; empty means "generate one"
	CN   string `json:"cn"`   // CN read from file; empty means "build from first and last name"
	SN   string `json:"sn"`   // SN read from file; empty means "generate last name"
	Mail string `json:"mail"` // Mail read from file; empty means "generate email"
}

// NewAttributeTemplate is an initializer function for AttributeTemplate.
// It returns an empty template that is ready to be filled from JSON.
func NewAttributeTemplate() *AttributeTemplate {
	return &AttributeTemplate{}
}

///////////////////////////////////////////////////////////////////////////////
// Fake entry representation
///////////////////////////////////////////////////////////////////////////////

// FakeEntry represents a single fake LDAP user. This struct is deliberately
// small so that it is easy to reason about and easy to convert into an
// *ldap.Entry that the LDAP and LDIF libraries understand.
type FakeEntry struct {
	DN   string // DN is the full distinguished name, for example "uid=jdoe,ou=employee,ou=users,o=rtx"
	UID  string // UID will become the "uid" attribute in LDAP
	CN   string // CN is the common name, for example "John Doe"
	SN   string // SN is the surname / last name, for example "Doe"
	Mail string // Mail is the email address
}

// NewFakeEntry is an initializer function for FakeEntry.
// It builds a new FakeEntry from explicit values, which makes the
// relationship between fields very clear and easy to test.
func NewFakeEntry(dn, uid, cn, sn, mail string) *FakeEntry {
	return &FakeEntry{
		DN:   dn,
		UID:  uid,
		CN:   cn,
		SN:   sn,
		Mail: mail,
	}
}

// NewFakeEntryWithTemplate creates a FakeEntry using gofakeit, then applies
// overrides from an AttributeTemplate if one is provided.
//
// suffixDN should be everything after "uid=<id>,", for example
// "ou=employee,ou=users,o=rtx". tmpl can be nil, in which case all
// attributes are generated.
func NewFakeEntryWithTemplate(suffixDN string, tmpl *AttributeTemplate) *FakeEntry {
	// Generate baseline fake values for the different pieces.
	// We do this first so we always have values to fall back on.
	first := gofakeit.FirstName() // random first name
	last := gofakeit.LastName()   // random last name
	email := gofakeit.Email()     // random-looking email address
	uid := gofakeit.Username()    // random username for uid

	// If a template was provided, we use its non-empty fields to override
	// the generated values. This lets the user control as much or as
	// little of the data as they want.
	if tmpl != nil {
		if tmpl.UID != "" {
			uid = tmpl.UID
		}
		if tmpl.SN != "" {
			last = tmpl.SN
		}
		if tmpl.Mail != "" {
			email = tmpl.Mail
		}
	}

	// Determine what the CN (common name) should be. If the template
	// specified a CN, we trust it. Otherwise, we build a simple CN by
	// combining the first and last names with a space.
	cn := first + " " + last
	if tmpl != nil && tmpl.CN != "" {
		cn = tmpl.CN
	}

	// Build the full DN in the format:
	//   uid=<uid>,<suffixDN>
	// This matches common LDAP patterns such as
	//   uid=jdoe,ou=employee,ou=users,o=rtx
	dn := fmt.Sprintf("uid=%s,%s", uid, suffixDN)

	// Use the explicit initializer so construction is obvious and consistent.
	return NewFakeEntry(dn, uid, cn, last, email)
}

///////////////////////////////////////////////////////////////////////////////
// Conversion helpers
///////////////////////////////////////////////////////////////////////////////

// ToLDAPEntry converts a FakeEntry into an *ldap.Entry. This is necessary
// because the LDIF library and the LDAP client library both expect
// *ldap.Entry values rather than custom structs. By keeping the conversion
// here, changes to attributes only have to be made once.
func (f *FakeEntry) ToLDAPEntry() *ldap.Entry {
	attrs := map[string][]string{
		"objectClass": {"inetOrgPerson"}, // inetOrgPerson is a common objectClass for user entries
		"uid":         {f.UID},           // uid attribute
		"cn":          {f.CN},            // cn attribute
		"sn":          {f.SN},            // sn attribute
		"mail":        {f.Mail},          // mail attribute
	}

	return ldap.NewEntry(f.DN, attrs)
}

///////////////////////////////////////////////////////////////////////////////
// LDIF writing
///////////////////////////////////////////////////////////////////////////////

// writeLDIFFile takes a set of *ldap.Entry values and writes them into an
// LDIF file whose path is specified in cfg.LDIFFile. It uses the ldif
// package to convert the entries into proper LDIF text.
func writeLDIFFile(cfg *RunConfig, entries []*ldap.Entry) error {
	// ldif.ToLDIF wraps the entries into a structure that ldif.Marshal
	// knows how to convert into a string.
	ldifData, err := ldif.ToLDIF(entries)
	if err != nil {
		return fmt.Errorf("failed to build LDIF struct: %w", err)
	}

	// ldif.Marshal turns the LDIF structure into a text LDIF string that
	// can be written to a file or printed on screen.
	ldifText, err := ldif.Marshal(ldifData)
	if err != nil {
		return fmt.Errorf("failed to marshal LDIF: %w", err)
	}

	// Write the LDIF string to disk. The permission 0644 means the owner
	// can read and write, while group and others can only read.
	if err := os.WriteFile(cfg.LDIFFile, []byte(ldifText), 0o644); err != nil {
		return fmt.Errorf("failed to write LDIF file: %w", err)
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// LDAP writing
///////////////////////////////////////////////////////////////////////////////

// writeToLDAP connects to an LDAP server using the information in RunConfig,
// then sends Add requests for each entry. This function is intentionally
// simple and meant for test/demo purposes rather than production.
//
// In a real environment, you would:
//   - validate TLS certificates instead of skipping verification,
//   - handle errors more gracefully,
//   - avoid passing plain text passwords around.
func writeToLDAP(cfg *RunConfig, entries []*ldap.Entry) error {
	// Dial the LDAP server using the URL from the config.
	// For LDAPS, use a URL like "ldaps://localhost:636".
	l, err := ldap.DialURL(cfg.LDAPURL, ldap.DialWithTLSConfig(&tls.Config{
		InsecureSkipVerify: true, // WARNING: For testing only. Do not use in production.
	}))
	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	// Ensure the connection is closed when we are done so resources
	// are not leaked.
	defer l.Close()

	// Authenticate (bind) to the server using the provided BindDN and
	// BindPassword so the server knows who we are.
	if err := l.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	// Loop over each entry and send an Add request for it.
	for _, e := range entries {
		req := ldap.NewAddRequest(e.DN, nil)

		// Copy all attributes from the ldap.Entry into the AddRequest.
		for _, attr := range e.Attributes {
			req.Attribute(attr.Name, attr.Values)
		}

		// Send the Add request to the server.
		if err := l.Add(req); err != nil {
			return fmt.Errorf("failed to add entry %s: %w", e.DN, err)
		}
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Top-level runner
///////////////////////////////////////////////////////////////////////////////

// Run is the main entry point for this package. It:
//
//  1. Validates the provided configuration.
//  2. Seeds the fake data generator.
//  3. Generates cfg.Count FakeEntry values, using cfg.Template if present.
//  4. Converts them into *ldap.Entry values.
//  5. Either writes an LDIF file or sends them to LDAP, based on cfg.Mode.
//
// This function does not depend on Kong or any CLI library; it only uses
// the RunConfig struct. That makes it easier to test and reuse.
func Run(cfg *RunConfig) error {
	// Basic validation so we catch configuration errors early and provide
	// clear messages to the caller.
	if cfg.SuffixDN == "" {
		return fmt.Errorf("SuffixDN must not be empty")
	}
	if cfg.Count < 1 {
		return fmt.Errorf("Count must be at least 1")
	}
	if cfg.Mode != "ldif" && cfg.Mode != "ldap" {
		return fmt.Errorf("Mode must be either 'ldif' or 'ldap'")
	}

	// Seed gofakeit once for the entire run. Using 0 gives deterministic
	// results, which is helpful while developing and testing. If you later
	// want different fake data every time, you can switch this to use
	// time.Now().UnixNano().
	gofakeit.Seed(0)

	// Generate the requested number of entries and convert them into
	// ldap.Entry values so they can be written to LDIF or LDAP.
	var ldapEntries []*ldap.Entry
	for i := 0; i < cfg.Count; i++ {
		fake := NewFakeEntryWithTemplate(cfg.SuffixDN, cfg.Template)
		ldapEntries = append(ldapEntries, fake.ToLDAPEntry())
	}

	// Decide what to do with the generated entries based on the Mode.
	switch cfg.Mode {
	case "ldif":
		return writeLDIFFile(cfg, ldapEntries)
	case "ldap":
		return writeToLDAP(cfg, ldapEntries)
	default:
		// This should never be reached because we validate Mode above,
		// but we keep it as a safety net.
		return fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}
}
