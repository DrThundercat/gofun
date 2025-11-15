package main

import (
	"fmt" // fmt is used for printing messages to the terminal
	"os"  // os is used for writing the LDIF string to a file

	"github.com/go-ldap/ldap/v3"   // ldap/v3 gives us LDAP types like Entry
	ldif "github.com/go-ldap/ldif" // ldif works with ldap types to produce LDIF text
)

// LDIFUser is a simple struct that represents the data we want to put into LDAP.
// We create this so our "business data" is decoupled from the ldap.Entry type.
// This makes it easier to test, change, or reuse later.
type LDIFUser struct {
	DN   string // DN is the full distinguished name, for example: "uid=jdoe,ou=people,dc=example,dc=com"
	UID  string // UID is the user's uid attribute
	CN   string // CN is the common name
	SN   string // SN is the surname (last name)
	Mail string // Mail is the email address
}

// NewLDIFUser is an initializer function for LDIFUser.
// It creates a new LDIFUser pointer so we can easily build user objects in a consistent way.
func NewLDIFUser(dn, uid, cn, sn, mail string) *LDIFUser {
	return &LDIFUser{
		DN:   dn,
		UID:  uid,
		CN:   cn,
		SN:   sn,
		Mail: mail,
	}
}

// ToLDAPEntry converts our LDIFUser into an *ldap.Entry.
//
// We do this because the ldif package expects ldap types (like *ldap.Entry)
// instead of our custom struct. By having this method, the conversion logic
// lives in one place and is easy to understand and update.
func (u *LDIFUser) ToLDAPEntry() *ldap.Entry {
	// ldap.NewEntry takes a DN and a map of attribute name -> []values.
	// We build the attribute map using standard LDAP attributes.
	attrs := map[string][]string{
		"objectClass": {"inetOrgPerson"}, // inetOrgPerson is a common objectClass for user entries
		"uid":         {u.UID},           // uid attribute gets the UID value
		"cn":          {u.CN},            // cn attribute gets the common name
		"sn":          {u.SN},            // sn attribute gets the surname (last name)
		"mail":        {u.Mail},          // mail attribute gets the email address
	}

	// NewEntry builds an ldap.Entry with the DN and attributes.
	// Using NewEntry ensures a stable attribute order, which makes
	// your LDIF more predictable across runs.
	entry := ldap.NewEntry(u.DN, attrs)
	return entry
}

// WriteLDIFFile takes a slice of *ldap.Entry, converts them into an LDIF,
// and writes the result into a file on disk.
//
// filename: name of the file we want to create, for example "output.ldif".
// entries:  list of LDAP entries that should be included in the LDIF file.
func WriteLDIFFile(filename string, entries []*ldap.Entry) error {
	// ldif.ToLDIF wraps our entries in an LDIF struct.
	// This struct is what the ldif package uses internally.
	// It can accept *ldap.Entry or LDAP request types (Add/Modify/Delete, etc.).
	ldifData, err := ldif.ToLDIF(entries)
	if err != nil {
		// If building the LDIF struct fails, we return the error to the caller.
		return fmt.Errorf("failed to build LDIF struct: %w", err)
	}

	// ldif.Marshal takes the LDIF struct and turns it into a text LDIF string.
	// This string is what you would usually save to a *.ldif file.
	ldifString, err := ldif.Marshal(ldifData)
	if err != nil {
		return fmt.Errorf("failed to marshal LDIF: %w", err)
	}

	// os.WriteFile writes the LDIF string to disk with the given permissions.
	// Here we use 0644 which means:
	// - Owner can read/write
	// - Group and Others can read
	err = os.WriteFile(filename, []byte(ldifString), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write LDIF file: %w", err)
	}

	return nil
}

// main is the entry point of our Go program.
// Here we build one example user, convert it to an LDAP entry,
// and write an LDIF file containing that user.
func main() {
	// First, we create an LDIFUser using our initializer function.
	// In a real program, these values might come from a CSV, a database,
	// or command-line arguments.
	user := NewLDIFUser(
		"uid=jdoe,ou=people,dc=example,dc=com", // DN
		"jdoe",                                 // UID
		"John Doe",                             // CN
		"Doe",                                  // SN
		"jdoe@example.com",                     // Mail
	)

	// Next, we convert our LDIFUser into an *ldap.Entry.
	entry := user.ToLDAPEntry()

	// We put the entry into a slice because WriteLDIFFile expects multiple entries.
	entries := []*ldap.Entry{entry}

	// Now we ask WriteLDIFFile to write everything to "output.ldif".
	if err := WriteLDIFFile("output.ldif", entries); err != nil {
		// If something goes wrong, we print the error and exit with a non-zero code.
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// If everything went well, we print a confirmation message.
	fmt.Println("LDIF file generated: output.ldif")
}
