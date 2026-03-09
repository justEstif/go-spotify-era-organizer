---
# go-spotify-era-organizer-osk0
title: Audit codebase for removed Spotify API fields
status: in-progress
type: task
priority: high
created_at: 2026-03-09T17:49:42Z
updated_at: 2026-03-09T18:02:01Z
parent: go-spotify-era-organizer-t6b9
---

Scan all code that reads Spotify API responses for usage of removed fields: track.Popularity, artist.Popularity, artist.Followers, user.Email, user.Country, user.Product, track.AvailableMarkets. Remove or replace with alternatives.
