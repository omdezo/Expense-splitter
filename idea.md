# Expense Splitter — Idea

## What it is

A backend program that calculates and fairly splits shared expenses among a
group of people during a trip. Think of a group going somewhere together —
a 3-day trip, for example — where different people pay for different things
along the way. At the end, the program figures out the fair share for everyone
and tells each person whether they should **get money back** or **pay more**.

## The problem it solves

When a group travels together, people don't pay equal amounts. One person pays
for the hotel, another pays for fuel, someone else covers lunch, and so on.
By the end of the trip it's confusing and tiring to manually work out who owes
who. This program does that math automatically and settles everything fairly.

## How it works

1. **Everyone joins a group** for the trip.
2. **Throughout the trip**, anyone who pays for something records that expense
   (what it was for and how much they paid). Each person keeps adding their
   payments over the whole trip — across all 3 days, or however long it lasts.
3. **At the end**, the program:
   - Adds up the **total of all expenses**.
   - Divides it **equally** among all the members of the group to get each
     person's **fair share**.
   - Compares what each person **actually paid** against their **fair share**.
4. **Settlement** — based on that comparison, the program decides for each
   person:
   - If they paid **more** than their fair share → they should **receive**
     the difference back.
   - If they paid **less** than their fair share → they should **pay** the
     difference.

## Example

A group is on a 3-day trip. Suppose the total expenses, divided equally, mean
each person's fair share is **60 rial**.

- If I paid **100 rial** during the whole trip but my fair share is only
  **60 rial**, then I overpaid → I should **get 40 rial back**.
- If someone else paid only **20 rial** but their fair share is **60 rial**,
  then they underpaid → they should **pay 40 rial**.

So money flows from the people who paid less to the people who paid more, until
everyone has effectively contributed the same fair share.

## The result

By the end, everything is balanced: each member ends up having paid exactly
their equal share of the trip, and nobody is owed or owing anything afterward.
The program turns a messy, manual calculation into a clear, automatic, and
fair settlement.
