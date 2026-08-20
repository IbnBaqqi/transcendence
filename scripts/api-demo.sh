#!/usr/bin/env bash
#
# Proves the public-API module end to end against a running stack: a key
# authenticates, does all four verbs, is refused where it should be, gets
# throttled, and stops working once revoked.
#
# This is the only thing in the project that exercises the whole chain -
# migration, query, service, dual auth, rate limiter, handler - in one go. Every
# other test stops at a fake or a throwaway database.
#
#   docker compose up -d --build
#   ./scripts/api-demo.sh
#
# Trip the limit sooner by starting the backend with a smaller one - it has to
# leave room for the six key-authenticated requests the earlier sections make:
#   RATE_LIMIT_PER_MINUTE=10 docker compose up -d --build backend
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
RATE_LIMIT_PER_MINUTE="${RATE_LIMIT_PER_MINUTE:-60}"

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is needed to read the JSON responses" >&2
	exit 1
fi

# One escalation check plus five verbs, all authenticated by key, happen before
# the limit is deliberately tripped. A smaller budget than that throttles the
# demonstration itself - which is correct behaviour, but a confusing way to find
# out about it halfway through.
readonly SPENT_BEFORE_SECTION_5=6
if [ "$RATE_LIMIT_PER_MINUTE" -le "$SPENT_BEFORE_SECTION_5" ]; then
	echo "RATE_LIMIT_PER_MINUTE is $RATE_LIMIT_PER_MINUTE; this script needs more than $SPENT_BEFORE_SECTION_5" >&2
	echo "to reach section 5. Restart the backend with a larger value, e.g. 10." >&2
	exit 1
fi

step() { printf '\n\033[1m== %s\033[0m\n' "$1"; }
pass() { printf '   \033[32mPASS\033[0m  %s\n' "$1"; }
info() { printf '         %s\n' "$1"; }

# req METHOD PATH EXPECTED_STATUS [curl args...] -> response body on stdout
req() {
	local method=$1 path=$2 want=$3
	shift 3

	local out status body
	out=$(curl -sS -w $'\n%{http_code}' -X "$method" "$BASE_URL$path" "$@")
	status=${out##*$'\n'}
	body=${out%$'\n'*}

	if [ "$status" != "$want" ]; then
		printf '\n   \033[31mFAIL\033[0m  %s %s answered %s, expected %s\n%s\n' \
			"$method" "$path" "$status" "$want" "$body" >&2
		exit 1
	fi

	printf '%s' "$body"
}

# json FIELD reads one top-level field from a JSON object on stdin.
json() { python3 -c 'import json,sys; print(json.load(sys.stdin)["'"$1"'"])'; }

suffix=$(date +%s)$RANDOM
email="demo-$suffix@example.test"

# ---------------------------------------------------------------------------
step "1. Sign up (a browser would do this)"
signup=$(req POST /auth/signup 201 \
	-H 'Content-Type: application/json' \
	-d "{\"username\":\"demo$suffix\",\"email\":\"$email\",\"password\":\"password123\"}")
token=$(printf '%s' "$signup" | json access_token)
pass "signed up as $email"

# ---------------------------------------------------------------------------
step "2. Create an API key (needs the session)"
created=$(req POST /me/api-keys 201 \
	-H "Authorization: Bearer $token" \
	-H 'Content-Type: application/json' \
	-d '{"name":"demo script"}')
key=$(printf '%s' "$created" | json key)
key_id=$(printf '%s' "$created" | json id)
auth=(-H "X-API-Key: $key")
pass "key created: $key"
info "this is the only time it is visible - the database holds a SHA-256 hash"

# ---------------------------------------------------------------------------
step "3. A key cannot mint another key"
req POST /me/api-keys 403 "${auth[@]}" \
	-H 'Content-Type: application/json' \
	-d '{"name":"escalation"}' >/dev/null
pass "POST /me/api-keys with the key -> 403"
info "otherwise a leaked key replaces itself and revoking the original does nothing"
info "checked before the budget below is spent: the limiter runs ahead of this,"
info "so an empty bucket would answer 429 before the check is ever reached"

# ---------------------------------------------------------------------------
step "4. Five endpoints, four verbs, with the key alone"
listing=$(req POST /listings 201 "${auth[@]}" \
	-H 'Content-Type: application/json' \
	-d '{"title":"Chanterelles","description":"picked this morning","category":"mushrooms","price":18.5,"quantity":4,"unit":"kg"}')
listing_id=$(printf '%s' "$listing" | json id)
pass "POST   /listings          created listing $listing_id"

req GET /listings 200 "${auth[@]}" >/dev/null
pass "GET    /listings          listed"

req GET "/listings/$listing_id" 200 "${auth[@]}" >/dev/null
pass "GET    /listings/{id}     read"

req PUT "/listings/$listing_id" 200 "${auth[@]}" \
	-H 'Content-Type: application/json' \
	-d '{"title":"Chanterelles","description":"picked this morning","category":"mushrooms","price":19.0,"quantity":3,"unit":"kg"}' >/dev/null
pass "PUT    /listings/{id}     updated"

req DELETE "/listings/$listing_id" 204 "${auth[@]}" >/dev/null
pass "DELETE /listings/{id}     deleted"

# ---------------------------------------------------------------------------
step "5. Rate limiting, per key"
info "limit is $RATE_LIMIT_PER_MINUTE/min; set RATE_LIMIT_PER_MINUTE lower to see this faster"

tripped=""
for _ in $(seq 1 $((RATE_LIMIT_PER_MINUTE + 1))); do
	out=$(curl -sS -o /dev/null -D - "${auth[@]}" "$BASE_URL/listings")
	status=$(printf '%s' "$out" | awk 'NR==1{print $2}')
	remaining=$(printf '%s' "$out" | awk -F': ' 'tolower($1)=="x-ratelimit-remaining"{gsub(/\r/,"",$2); print $2}')

	if [ "$status" = "429" ]; then
		retry=$(printf '%s' "$out" | awk -F': ' 'tolower($1)=="retry-after"{gsub(/\r/,"",$2); print $2}')
		pass "throttled after the bucket emptied: 429, Retry-After: ${retry}s"
		tripped=yes
		break
	fi
	printf '         remaining: %s\r' "${remaining:-?}"
done

if [ -z "$tripped" ]; then
	printf '\n   \033[31mFAIL\033[0m  never hit the limit in %s requests\n' "$((RATE_LIMIT_PER_MINUTE + 1))" >&2
	exit 1
fi

# ---------------------------------------------------------------------------
step "6. Revoke, and the key stops working"
req DELETE "/me/api-keys/$key_id" 204 -H "Authorization: Bearer $token" >/dev/null
pass "revoked key $key_id"

req GET /listings 401 "${auth[@]}" >/dev/null
pass "the same key now answers 401"
info "revocation applies on the next request - the key is looked up every time"

printf '\n\033[32mAll sections passed.\033[0m\n'
