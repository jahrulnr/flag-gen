#!/usr/bin/env bash
# Parse .env into exports without evaluating values as shell (safe for MYSQL_DSN tcp(...) and &).
# Usage (from repo root):  . ./scripts/load-dotenv.sh ./.env

envfile="${1:-}"
if [[ -z "$envfile" ]]; then
	echo "usage: . ${BASH_SOURCE[0]##*/} /path/to/.env" >&2
	return 1 2>/dev/null || exit 1
fi
if [[ ! -f "$envfile" ]]; then
	return 0 2>/dev/null || exit 0
fi

while IFS= read -r line || [[ -n "$line" ]]; do
	line="${line%$'\r'}"
	[[ "$line" =~ ^[[:space:]]*# ]] && continue
	[[ -z "${line// }" ]] && continue
	if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
		key="${BASH_REMATCH[1]}"
		val="${BASH_REMATCH[2]}"
		if [[ "$val" =~ ^\'(.*)\'$ ]]; then
			val="${BASH_REMATCH[1]}"
		elif [[ "$val" =~ ^\"(.*)\"$ ]]; then
			val="${BASH_REMATCH[1]}"
		fi
		export "$key=$val"
	fi
done <"$envfile"
