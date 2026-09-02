#!/usr/bin/env bash
#
# Creates the two accounts and the listing needed to try the buyer order flow
# (#22) by hand. Prints the two logins to use in the browser.
#
#   docker compose up -d
#   cd backend && make migrate-up && cd ..
#   ./scripts/seed-order-demo.sh
set -euo pipefail

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is needed to read the JSON responses" >&2
	exit 1
fi

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
PASSWORD="password123"
# $RANDOM as well as the timestamp: two runs in the same second would collide
# on the username and the second would fail.
suffix=$(date +%s)$RANDOM

json() { python3 -c 'import json,sys; print(json.load(sys.stdin)["'"$1"'"])'; }

# req METHOD PATH EXPECTED_STATUS [curl args...] -> body on stdout
req() {
	local method=$1 path=$2 want=$3
	shift 3

	local out status body
	out=$(curl -sS -w $'\n%{http_code}' -X "$method" "$BASE_URL$path" "$@")
	status=${out##*$'\n'}
	body=${out%$'\n'*}

	if [ "$status" != "$want" ]; then
		printf '\n  %s %s answered %s, expected %s\n%s\n' \
			"$method" "$path" "$status" "$want" "$body" >&2
		exit 1
	fi

	printf '%s' "$body"
}

signup() {
	req POST /auth/signup 201 \
		-H 'Content-Type: application/json' \
		-d "{\"username\":\"$1\",\"email\":\"$1@example.test\",\"password\":\"$PASSWORD\"}"
}

seller="seller$suffix"
buyer="buyer$suffix"

# Captured, then piped separately: a pipeline reports the LAST command's exit
# status, so `signup ... | json` would mask req's exit and leave json parsing
# an error body.
seller_signup=$(signup "$seller")
seller_token=$(printf '%s' "$seller_signup" | json access_token)
signup "$buyer" >/dev/null

listing=$(req POST /listings 201 \
	-H "Authorization: Bearer $seller_token" \
	-H 'Content-Type: application/json' \
	-d '{"title":"Chanterelles","description":"Picked this morning near Nuuksio.","category":"mushrooms","price":18.5,"quantity":4,"unit":"kg"}')

listing_id=$(printf '%s' "$listing" | json id)

cat <<TXT

  Seller   $seller@example.test  /  $PASSWORD
  Buyer    $buyer@example.test  /  $PASSWORD

  Listing  http://localhost:5173/listings/$listing_id
           Chanterelles, 4 kg at EUR 18.50/kg

  Log in as the buyer in one window and the seller in a private window -
  the access token lives in localStorage, so two normal tabs share a session.

TXT
