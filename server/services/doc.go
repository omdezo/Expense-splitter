// Package services holds the business-logic layer that sits between the HTTP
// handlers and the database. Services own transactions, authorization checks
// (requireGroupRole), audit-log writes, and the state-machine guards. Empty for now.
package services
