package main

import (
	"fmt" // used to print simple messages to the terminal
	"os"  // used for exiting with an error code
	"time"

	"github.com/brianvoe/gofakeit/v6" // used to create fake test data
	"github.com/go-ldap/ldap/v3"      // used to hold LDAP entry types
	ldif "github.com/go-ldap/ldif"    // used to convert entries into LDIF text
)

// LDIFUser holds the data we want to store in LDAP.
// This struct is separate from any fake data generator so we can reuse it
// with real or fake values.
type LDIFUser struct {
	DN   string // full distinguished name such as "uid=jdoe,ou=employee,dc=example,dc=com"
	UID  string // LDAP uid attribute
	CN   string // common name
	SN   string // surname / last name
	Mail string // email address
}

// NewLDIFUser is an initializer function that builds an LDIFUser.
// This function keeps construction logic in one place, which makes the code easier to change.
func NewLDIFUser(dn, uid, cn, sn, mail string) *LDIFUser {
	return &LDIFUser{
		DN:   dn,
		UID:  uid,
		CN:   cn,
		SN:   sn,
		Mail: mail,
	}
}

// ToLDAPEntry converts an LDIFUser into an *ldap.Entry so that the ldif package
// can later turn it into LDIF text for writing to a file.
func (u *LDIFUser) ToLDAPEntry() *ldap.Entry {
	attrs := map[string][]string{
		"objectClass": {"inetOrgPerson"},
		"uid":         {u.UID},
		"cn":          {u.CN},
		"sn":          {u.SN},
		"mail":        {u.Mail},
	}
	return ldap.NewEntry(u.DN, attrs)
}

// NewFakeLDIFUser creates a new LDIFUser filled with fake data using gofakeit.
// This lets you create LDIFs for testing without needing real user data.
func NewFakeLDIFUser() *LDIFUser {
	// Seed with 0 for deterministic output each run.
	// If you want different results every time, use time.Now().UnixNano().
	gofakeit.Seed(time.Now().UnixNano())

	first := gofakeit.FirstName()
	last := gofakeit.LastName()
	email := gofakeit.Email()
	uid := gofakeit.Username()

	// We build a DN that looks like: uid=<uid>,ou=employee,dc=example,dc=com
	dn := fmt.Sprintf("uid=%s,ou=employee,dc=example,dc=com", uid)

	return NewLDIFUser(
		dn,             // DN of the entry
		uid,            // UID attribute
		first+" "+last, // CN attribute
		last,           // SN attribute
		email,          // Mail attribute
	)
}

// WriteLDIF writes a slice of *ldap.Entry as a single LDIF file on disk.
func WriteLDIF(filename string, entries []*ldap.Entry) error {
	ldifData, err := ldif.ToLDIF(entries)
	if err != nil {
		return fmt.Errorf("failed to build LDIF struct: %w", err)
	}

	ldifText, err := ldif.Marshal(ldifData)
	if err != nil {
		return fmt.Errorf("failed to marshal LDIF: %w", err)
	}

	if err := os.WriteFile(filename, []byte(ldifText), 0o644); err != nil {
		return fmt.Errorf("failed to write LDIF file: %w", err)
	}

	return nil
}

// main is the entry point of the program.
// Here we create one fake user, convert it to an LDAP entry, and save it as LDIF.
func main() {
	// Create a fake user with realistic test values.
	fakeUser := NewFakeLDIFUser()

	// Convert the fake user into an LDAP entry.
	entry := fakeUser.ToLDAPEntry()

	// Write the LDIF file containing this single entry.
	if err := WriteLDIF("fake_users.ldif", []*ldap.Entry{entry}); err != nil {
		fmt.Println("Error writing LDIF:", err)
		os.Exit(1)
	}

	fmt.Println("Generated fake LDIF file: fake_users.ldif")
}
