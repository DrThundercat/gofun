go run ./cmd/fakeldap \
  --suffix-dn "ou=employee,ou=users,o=rtx" \
  --count 5 \
  --mode ldif \
  --ldif-file fake_users.ldif

go run ./cmd/fakeldap \
  --suffix-dn "ou=employee,ou=users,o=rtx" \
  --count 3 \
  --mode ldif \
  --ldif-file fake_users.ldif \
  --input-file examples/entry_template.json

go run ./cmd/fakeldap \
  --suffix-dn "ou=employee,ou=users,o=rtx" \
  --count 3 \
  --mode ldap \
  --ldap-url "ldaps://localhost:636" \
  --bind-dn "cn=Directory Manager" \
  --bind-password "secret"
