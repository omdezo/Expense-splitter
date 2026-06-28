package middleware

// Authorization middleware will live here and run after Auth has established the
// caller's Identity. It will:
//   - load the caller's Principal (the users row: id, is_global_admin,
//     verification_status) and require a verified account;
//   - for group endpoints, resolve the caller's per-group role from memberships
//     (RequireGroupRole), treating the global admin as group-admin everywhere;
//   - require global-admin for system-wide actions.
//
// Authentication (who the caller is) stays in auth.go and jwt.go; authorization
// (what they may do) belongs here. Intentionally empty for now.
