# { BE Expense Splitter — Idea }

## Background

Group trips are a logistics nightmare for one reason: **nobody pays equally as they go.** One person books the hotel, another keeps filling the tank, a third covers every lunch, and somebody pays nothing the whole trip but eats the most. By the last day the group chat is a forensic accounting thread — receipts, "who paid for the second night?", and one friend with a calculator who gets it wrong.

You're building the backend that ends that argument. Members record what they paid throughout the trip. At settlement, the system computes each person's fair share, who owes whom, and the **minimum set of payments** to make everyone whole. Each debtor then pays their creditor **in real life** — cash or a personal bank transfer — and uploads **proof**. The creditor confirms they received it, an admin finalizes, and only then is that payment marked **settled**. No money ever flows through the system; its job is correct math and trustworthy, evidence-backed confirmation.

**Build the system that makes the group chat shut up  💸🧾**

## Problem statement

Develop a backend server system (API) for a shared-expense settlement tracker. Users register and verify their accounts, form trip groups, record expenses, and at settlement the system computes an optimized who-pays-whom plan. Settlement happens **off-platform** (cash / personal transfer); the system tracks each computed payment through a **two-key confirmation** workflow: the debtor uploads proof, the creditor attests receipt, and a group-admin or global admin finalizes.

The system **holds no money and integrates no payment provider.** Integrity comes from evidence (uploaded proof) plus human confirmation, not from a gateway. Its responsibilities are: correct fair-share math, an exact-baisa settlement algorithm, robust proof handling, and a tamper-resistant confirmation state machine with a full audit trail.


## 📌 Overall Requirements and APIs

1. The system must start with a **default global admin**.

2. **Account verification.** A registered user cannot join a group, record expenses, or be settled against until **verified**. State: `registered → pending_verification → verified | rejected`. Only verified users participate.

3. **Roles are per-group, not global.** A user's role is a property of the **(user, group) pair**, stored in a `memberships` table (`user_id, group_id, role`). The same user can be **group-admin of Trip A** and a plain **member of Trip B** at once. A single **global admin** sits above all groups.

   - **Global Admin:** full control over the entire system. Manage all users, verify/reject accounts, view everything, and **override-confirm or reject any payment in any group**. The global admin holds **group-admin powers in every group implicitly** — any check of the form "is the caller this group's group-admin?" must also pass for the global admin, **without** requiring a membership row. The global admin may also **create a group on behalf of others and assign the group-admin role to one of its members**, then step back (they are not a trip member and are not included in the split).
   - **Group-Admin** (group creator; transferable — see #8): set trip duration/metadata, approve join requests, close the group to compute settlement, transfer their role, and **finalize payments in their own group**.
   - **Member:** view their groups, record expenses **they personally paid** (`paid_by == self`), edit/delete **only their own** expenses while the group is open, upload proof for payments **they owe**, confirm receipt for payments **they are owed**, view results.

4. **Authentication.** Basic Authentication on all APIs **except** the public group-status check (#17). A user authenticates **once**; the target group is set by a `group_id` in the request, not a separate session. Every group endpoint must verify the caller's membership **and role in that specific group** before acting.

5. **Money handling — non-negotiable.**
   - All amounts are integer **baisa** (minor units). `1.000 OMR` = `1000`. **No floats** in money math.
   - The system holds **no balances and moves no money.** It records expenses, computes shares, and tracks the confirmation state of each computed payment.

6. **Supporting endpoints (assumed, not enumerated).** Beyond the numbered APIs below, the system is expected to provide the obvious supporting endpoints needed for a coherent system — e.g. user **registration** and the **verification submission** flow, listing **the groups a caller belongs to**, fetching a **group's metadata and member list** (distinct from the financial summary in #12), the **group-admin's view of pending join requests**, and a member **leaving a group** when they have no recorded expenses. These follow the same auth and per-group-role rules as everything else; they are not spelled out individually so the builder can design them sensibly. The numbered APIs are the ones with **non-trivial rules** that must be implemented exactly as described.

7. **Group Management APIs:**
   - **Create group** (any verified user, **or the global admin on behalf of others**): name, **trip start/end dates** (duration), base currency = OMR. A verified user who creates a group becomes its **group-admin**; when the **global admin** creates a group, they **assign the group-admin role to a chosen member** and are not themselves part of the trip or the split. An optional `expected_member_count` is a **planning hint only** — it must have **zero effect** on any calculation. Fair-share always divides by the count of **actual current members** at settlement.
   - **Update group** (group-admin, open groups only).
   - **Close group** (group-admin) → freezes expenses and computes the settlement plan.

8. **Membership APIs (invite + approve):**
   - A group has a **shareable invite token**. A verified user uses it to **request** to join.
   - The **group-admin approves or rejects** each request: `requested → approved | rejected`. No open self-join.
   - A member with **any recorded expense** cannot be removed (it would corrupt the split).

9. **Group-Admin Handoff API:**
   - The group-admin can **transfer the role** to another approved member. Exactly one group-admin per group at all times. Logged.

10. **Expense Recording API:**
   - Record an expense: `paid_by` (**must equal caller**), `amount` (baisa), `description`, `category` (lodging, fuel, food, transport, other), `occurred_on`.
   - Validate: positive amount, payer is an approved member, group **open**, date within trip range.
   - Any change to an expense amount is written to the audit log with **before/after** values.

11. **Expense Listing API:**
    - Return all expenses for a group: Expense ID, Paid By, Amount, Category, Description, Occurred On, Created At.
    - Description ≤ **80 characters**; if longer, truncate with `...` keeping the **last word complete** (accept `"dinner at the ..."`, reject `"dinner at the har..."`).
    - Filter by category and payer; search by description.

12. **Group Summary API:**
    - Given a group ID: trip name, dates, member count, total spent, spend per category, total **paid** per member, each member's **fair share**, and **net balance** (paid − fair share).

13. **Settlement Computation — the core algorithm:**
    - For a **closed** group, compute each member's net balance, then produce the **payment plan**: a list of `{ from, to, amount }` that brings every net balance to zero.
    - **Minimize the number of payments.** Naive settlement yields up to N×(N−1); do far better. Reference case: shares 60 each, total 180 — Ahmed paid 100 (+40), Omar paid 80 (+20), Mohammed paid 0 (−60) → **Mohammed → Ahmed 40, Mohammed → Omar 20** (2 payments, reconciles to zero). The algorithm must also handle the **multi-debtor, multi-creditor** case (two owed, two owing) — the case the single-debtor example hides. Document the algorithm + complexity in the README; optimal minimum-transfer is NP-hard, so a justified greedy heuristic is expected — name it.
    - **Rounding:** when total ÷ members doesn't divide evenly, distribute the remainder baisa deterministically (e.g. first *k* members by stable order each absorb one extra baisa). Same input → same plan. Documented and tested.
    - Sum of payments out must equal sum in. Include a test proving it.

14. **Settlement Plan API:**
    - Group-admin/global-admin/members fetch the computed plan for a closed group: the full `{ from, to, amount, status }` list. All payments start `pending`.

15. **Proof Upload & Two-Key Confirmation — integrity core:**
    Each computed payment moves through this state machine:
    - `pending` — nothing submitted yet.
    - The **debtor uploads proof** (a receipt/transfer-screenshot **image**, or a text note e.g. "paid 40 cash on day 3") → `proof_submitted`.
      - Images must be **validated as actual images** (not arbitrary files renamed). Store the image and its size.
    - The **creditor** (the person owed) reviews and **attests receipt** → `creditor_confirmed`. If the creditor says they did **not** receive it → `disputed`.
    - A **group-admin** (own group) or **global admin** (any group) **finalizes** → `settled`. An admin may also **reject** → `disputed`, returning it to the debtor to re-submit.
    - **Global admin override:** may confirm or reject at any step.
    - **No self-marking shortcut exists:** a debtor cannot move their own payment past `proof_submitted`. Settlement requires the creditor's attestation **and** an admin's finalization. Enforce this in authorization, not just convention.
    - **Partial settlement is normal, not an error.** Report per-payment status and an aggregate ("3 of 5 settled"). The group becomes `fully_settled` only when every payment is `settled`, then goes read-only.

16. **Audit Log API:**
    - Log every: expense create/update/delete, group close, membership approval/rejection, group-admin handoff, proof upload, creditor confirmation, admin finalize/reject, and override. Capture who, what, before/after, when.
    - Admin API to return the audit log for a group.

17. **Public Group Status API:**
    - **Public** (no auth), given a shareable group token: trip name, open/closed/fully_settled, total spent, member count — **no per-person financial detail.**

18. **Proof Retrieval API:**
    - Return a payment's proof entry given its ID. If the proof is an image, also return the **image size**. Provide a separate endpoint that returns the **image bytes** itself, handling the case where the proof is a text note, not an image.

19. **Settlement Report API:**
    - Generate a **PDF** for a fully-settled group: trip details, full expense list, per-member paid/fair-share/net, and the payment plan with each payment's final status and who confirmed/finalized it.

## Technical Requirements

1. Use **git**. Code on GitHub; submit the repo link.
2. Any language/framework.
3. Database: **PostgreSQL**.
4. File storage (proof images, PDFs): any (MinIO, GCS, local volume).

## 💡 Bonus Challenges

### 1. 🔒 Tamper-Resistant Confirmation (treat as near-mandatory)
Prove with tests that no path lets a debtor mark their own payment `settled`, that `settled` requires both creditor attestation and admin finalization, and that a `disputed` payment cannot be reported as settled. This is the integrity spine now that there's no gateway.

### 2. 🧾 Proof Integrity
Validate uploaded images are genuine images (magic-byte check, not just extension). Optionally compute and store a hash of each proof so a swapped file is detectable. Reject non-image binaries masquerading as receipts.

### 3. 🔐 Concurrency Safety
Two admins finalize the same payment at once, or an expense lands the instant a group closes. Guarantee settlement is computed **exactly once** over a consistent snapshot and each payment is finalized **exactly once**. Document your locking/transaction strategy.

### 4. ⚖️ Weighted & Subset Splits
Support expenses split only among a named subset of members (fuel among the 4 in the car, not all 8), and per-member weights (single-room occupant owes more of the lodging). Settlement must still reconcile to zero.

### 5. 🔔 Settlement Nudges
For a partially-settled group, generate reminders for debtors whose payments are still `pending` past a threshold (email/log), and for creditors sitting on a `proof_submitted` payment awaiting their attestation. Keep it idempotent — don't spam on every poll.

### 6. 🚀 Deployment
Dockerize with Docker Compose, deploy to any cloud, submit the live link and a README.

## NOTE

Purely backend. No frontend. No payment provider — settlement happens off-platform and is confirmed by evidence + people. 

---
