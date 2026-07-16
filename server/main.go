package main

import (
	"expense-splitter/cmd"

	_ "expense-splitter/docs"
)

// @title                      Expense Splitter API
// @version                    1.0
// @description                Shared-expense settlement tracker for group trips. Members record what they paid; closing a group computes each member's fair share and the minimum set of payments that squares everyone up. Money moves off-platform — each computed payment is tracked through a two-key confirmation flow (debtor proves, creditor attests, admin finalizes).
// @description
// @description                All amounts are integer **baisa** (1.000 OMR = 1000). There are no floats in the money path.
// @description
// @description                **Authenticating here:** call `POST /auth/login` (try `admin@expense-splitter.local` / `admin`), copy the `access_token` from the response, click **Authorize** above and paste `Bearer <access_token>`.

// @host                       localhost:8080
// @BasePath                   /
// @schemes                    http

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Bearer token from POST /auth/login. Format: `Bearer <access_token>`

// @tag.name                   auth
// @tag.description            Public session endpoints — login, refresh, logout, sign-up. No bearer token required.
// @tag.name                   account
// @tag.description            The caller's own account: identity, local-row linking, verification submission.
// @tag.name                   groups
// @tag.description            Trip groups — create, read, update, close (close computes the settlement).
// @tag.name                   membership
// @tag.description            Join requests, approval/rejection, admin handoff, member removal.
// @tag.name                   expenses
// @tag.description            Recording and listing what each member paid, while the group is open.
// @tag.name                   settlement
// @tag.description            Fair-share summary, the computed payment plan, and the report PDF.
// @tag.name                   payments
// @tag.description            The two-key confirmation flow: proof upload, creditor confirm/dispute, admin finalize/reject.
// @tag.name                   admin
// @tag.description            Global-admin only — user verification and system-wide user/group management.
// @tag.name                   ops
// @tag.description            Audit trail and reminder nudges.
// @tag.name                   public
// @tag.description            Unauthenticated endpoints — health and share-token group status.
func main() {
	cmd.Execute()
}
