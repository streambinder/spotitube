#!/bin/bash

if [ -n "${SPOTIFY_ID}" ] || [ -n "${SPOTIFY_KEY}" ]; then
    sed -i'.bak' 's|clientID[[:space:]].*""|clientID = "'"${SPOTIFY_ID}"'"|g;
                  s|clientSecret[[:space:]].*""|clientSecret = "'"${SPOTIFY_KEY}"'"|g' spotify/api.go
fi
if [ -n "${GENIUS_TOKEN}" ]; then
    sed -i'.bak' 's|geniusToken[[:space:]].*""|geniusToken = "'"${GENIUS_TOKEN}"'"|g' lyrics/genius.go
fi