#!/usr/bin/env bash
#
# Creates the accounts and listing needed to try the chat consent flow (#88)
# by hand: a seller, a buyer with a pending request waiting for an answer,
# and a second buyer whose request is already accepted.
#
#   docker compose up -d
#   cd backend && make migrate-up && cd ..
#   ./scripts/seed-chat-demo.sh
set -euo pipefail

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is needed to read the JSON responses" >&2
	exit 1
fi

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
PASSWORD="password123"
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
buyer2="buyer2$suffix"

# Captured, then piped separately: a pipeline reports the LAST command's exit
# status, so `signup ... | json` would mask req's exit.
seller_signup=$(signup "$seller")
seller_token=$(printf '%s' "$seller_signup" | json access_token)

buyer_signup=$(signup "$buyer")
buyer_token=$(printf '%s' "$buyer_signup" | json access_token)

buyer2_signup=$(signup "$buyer2")
buyer2_token=$(printf '%s' "$buyer2_signup" | json access_token)

listing=$(req POST /listings 201 \
	-H "Authorization: Bearer $seller_token" \
	-H 'Content-Type: application/json' \
	-d '{"title":"Cloudberries","description":"Picked in the bog this morning.","category":"berries","price":25,"quantity":3,"unit":"litre"}')
listing_id=$(printf '%s' "$listing" | json id)

# Left pending on purpose: this is the request the seller answers by hand.
req POST /conversations 201 \
	-H "Authorization: Bearer $buyer_token" \
	-H 'Content-Type: application/json' \
	-d "{\"listing_id\":\"$listing_id\",\"body\":\"Is this still available?\"}" >/dev/null

# Already accepted, so there is a live thread to send messages in immediately.
accepted=$(req POST /conversations 201 \
	-H "Authorization: Bearer $buyer2_token" \
	-H 'Content-Type: application/json' \
	-d "{\"listing_id\":\"$listing_id\",\"body\":\"Could I get 2 litres?\"}")
accepted_id=$(printf '%s' "$accepted" | json id)

req POST "/conversations/$accepted_id/accept" 200 \
	-H "Authorization: Bearer $seller_token" >/dev/null

req POST "/conversations/$accepted_id/messages" 201 \
	-H "Authorization: Bearer $seller_token" \
	-H 'Content-Type: application/json' \
	-d '{"body":"Yes - I can set 2 aside."}' >/dev/null

cat <<TXT

  Seller    $seller@example.test  /  $PASSWORD
            one pending request to answer, one accepted thread

  Buyer 1   $buyer@example.test  /  $PASSWORD
            request is PENDING - send box should be shut

  Buyer 2   $buyer2@example.test  /  $PASSWORD
            request ACCEPTED, one unread reply waiting

  Listing   http://localhost:5173/listings/$listing_id

  Use a normal window and a private window - the access token lives in
  localStorage, so two normal tabs share one session.

TXT
