---
# go-spotify-era-organizer-osk0
title: Audit codebase for removed Spotify API fields
status: completed
type: task
priority: high
created_at: 2026-03-09T17:49:42Z
updated_at: 2026-03-09T18:04:53Z
parent: go-spotify-era-organizer-t6b9
---

Scan all code that reads Spotify API responses for usage of removed fields: track.Popularity, artist.Popularity, artist.Followers, user.Email, user.Country, user.Product, track.AvailableMarkets. Remove or replace with alternatives.

## Summary of Changes\nRemoved Email from User model/DB. Added migration 000008. Audit found no other removed field usage.
