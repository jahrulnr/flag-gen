#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:6123/v1}"
PROJECT_KEY="${PROJECT_KEY:-development}"
PROJECT_ID="${PROJECT_ID:-}"
ENVIRONMENT_KEY="${ENVIRONMENT_KEY:-development}"
PUBLIC_READ_API_KEY="${PUBLIC_READ_API_KEY:-dev-public-key}"
API_KEY="${API_KEY:-}"
CLEANUP="${CLEANUP:-true}"

FLAG_KEY="${FLAG_KEY:-e2e_eval_$(date +%s)}"
IDEMPOTENCY_PREFIX="${IDEMPOTENCY_PREFIX:-e2e-$FLAG_KEY}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

headers=(-H 'Content-Type: application/json')
if [[ -n "$API_KEY" ]]; then
  headers+=(-H "X-API-Key: $API_KEY")
fi

curl_json() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  local output="$4"
  local status
  if [[ -n "$body" ]]; then
    status="$(curl -sS -o "$output" -w '%{http_code}' -X "$method" "${headers[@]}" "$url" -d "$body")"
  else
    status="$(curl -sS -o "$output" -w '%{http_code}' -X "$method" "${headers[@]}" "$url")"
  fi
  printf '%s' "$status"
}

assert_status() {
  local got="$1"
  local want="$2"
  local body="$3"
  if [[ "$got" != "$want" ]]; then
    echo "expected HTTP $want, got $got" >&2
    cat "$body" >&2
    exit 1
  fi
}

python_assert() {
  local file="$1"
  local expr="$2"
  python3 - "$file" "$expr" <<'PY'
import json
import sys

path, expr = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)
if not eval(expr, {"data": data}):
    raise SystemExit(f"assertion failed: {expr}\n{json.dumps(data, indent=2)}")
PY
}

echo "base=$BASE_URL project=$PROJECT_ID env=$ENVIRONMENT_KEY flag=$FLAG_KEY"

status_body="$tmpdir/status.json"
status="$(curl_json GET "$BASE_URL/system/status" "" "$status_body")"
assert_status "$status" 200 "$status_body"
python_assert "$status_body" "data['database']['state'] in ('db_ready', 'migration_optional')"

if [[ -z "$PROJECT_ID" ]]; then
  projects_body="$tmpdir/projects.json"
  status="$(curl_json GET "$BASE_URL/projects" "" "$projects_body")"
  assert_status "$status" 200 "$projects_body"
  PROJECT_ID="$(python3 - "$projects_body" "$PROJECT_KEY" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
for project in data.get("projects", []):
    if project.get("Key") == sys.argv[2] or project.get("key") == sys.argv[2]:
        print(project.get("ID") or project.get("id"))
        break
else:
    raise SystemExit(f"project key not found: {sys.argv[2]}")
PY
)"
fi
echo "resolved_project_id=$PROJECT_ID"

create_body="$tmpdir/create.json"
create_payload="$(cat <<JSON
{
  "key": "$FLAG_KEY",
  "name": "E2E Eval Contract",
  "description": "temporary e2e flag",
  "enabled": true,
  "clientSide": true,
  "value": {"type": "boolean", "default": false},
  "rules": [
    {"attribute": "country", "operator": "equals", "value": "ID", "serve": true, "priority": 10}
  ],
  "rollout": {"percentage": 0, "stickiness": "userId"}
}
JSON
)"
status="$(curl -sS -o "$create_body" -w '%{http_code}' -X POST "${headers[@]}" \
  -H "Idempotency-Key: $IDEMPOTENCY_PREFIX-create" \
  "$BASE_URL/projects/$PROJECT_ID/environments/$ENVIRONMENT_KEY/flags" \
  -d "$create_payload")"
assert_status "$status" 201 "$create_body"
python_assert "$create_body" "data['flag']['Key'] == '$FLAG_KEY' or data['flag']['key'] == '$FLAG_KEY'"
FLAG_ID="$(python3 - "$create_body" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    flag = json.load(fh)["flag"]
print(flag.get("ID") or flag.get("id"))
PY
)"
FLAG_VERSION="$(python3 - "$create_body" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    flag = json.load(fh)["flag"]
print(flag.get("Version") or flag.get("version") or 1)
PY
)"

post_eval_body="$tmpdir/post-evaluate.json"
post_eval_payload="$(cat <<JSON
{
  "context": {"userId": "user-1", "attributes": {"country": "ID"}},
  "flags": ["$FLAG_KEY"]
}
JSON
)"
status="$(curl_json POST "$BASE_URL/projects/$PROJECT_ID/environments/$ENVIRONMENT_KEY/flags/evaluate" "$post_eval_payload" "$post_eval_body")"
assert_status "$status" 200 "$post_eval_body"
python_assert "$post_eval_body" "data['results']['$FLAG_KEY']['reason'] == 'rule_match'"
python_assert "$post_eval_body" "data['results']['$FLAG_KEY']['value'] is True"
python_assert "$post_eval_body" "data['results']['$FLAG_KEY']['source'] in ('cache_l1', 'cache_l2', 'database', 'snapshot')"
python_assert "$post_eval_body" "'timestamp' in data"

get_eval_body="$tmpdir/get-evaluate.json"
status="$(curl_json GET "$BASE_URL/projects/$PROJECT_ID/environments/$ENVIRONMENT_KEY/flags/$FLAG_KEY/evaluate?userId=user-1" "" "$get_eval_body")"
assert_status "$status" 200 "$get_eval_body"
python_assert "$get_eval_body" "data['result']['reason'] in ('rollout_miss', 'default')"
python_assert "$get_eval_body" "'timestamp' in data"

public_eval_body="$tmpdir/public-evaluate.json"
status="$(curl -sS -o "$public_eval_body" -w '%{http_code}' \
  -H "X-Public-Api-Key: $PUBLIC_READ_API_KEY" \
  "$BASE_URL/client/projects/$PROJECT_ID/environments/$ENVIRONMENT_KEY/flags/$FLAG_KEY/evaluate?userId=user-1")"
assert_status "$status" 200 "$public_eval_body"
python_assert "$public_eval_body" "data['result']['reason'] in ('rollout_miss', 'default')"

if [[ "$CLEANUP" == "true" ]]; then
  delete_body="$tmpdir/delete.json"
  status="$(curl -sS -o "$delete_body" -w '%{http_code}' -X DELETE "${headers[@]}" \
    -H "Idempotency-Key: $IDEMPOTENCY_PREFIX-delete" \
    "$BASE_URL/projects/$PROJECT_ID/environments/$ENVIRONMENT_KEY/flags/$FLAG_ID?version=$FLAG_VERSION")"
  assert_status "$status" 204 "$delete_body"
fi

echo "ok: api contract e2e passed for $FLAG_KEY"
