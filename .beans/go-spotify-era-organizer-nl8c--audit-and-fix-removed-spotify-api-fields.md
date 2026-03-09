---
# go-spotify-era-organizer-nl8c
title: Audit and fix removed Spotify API fields
status: completed
type: task
priority: normal
created_at: 2026-03-09T18:02:23Z
updated_at: 2026-03-09T18:02:53Z
---

Remove usage of spotifyUser.Email and drop email column from users table

## Summary of Changes

- Removed `Email` field from `db.User` struct in `internal/db/models.go`
- Removed email from all SQL queries in `internal/db/users.go` (Create, Get, Upsert)
- Removed `spotifyUser.Email` usage in `internal/web/handlers.go`
- Added migration `000008_drop_users_email` to drop the email column from the users table
- Verified no other removed Spotify fields (Popularity, Followers, AvailableMarkets, LinkedFrom, ExplicitContent, Country, Product, Label) are used anywhere in the codebase
