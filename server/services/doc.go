// Package services holds the business-logic layer that sits between the HTTP
// handlers and the database. Services own transactions, authorization checks,
// audit-log writes, and the state-machine guards. Handlers call their methods,
// which return either a result or a types.APIError.
package services
