#!/usr/bin/env bash
#
# Creates the two accounts and the listing needed to try the buyer order flow
# (#22) by hand. Prints the two logins to use in the browser.
#
#   docker compose up -d
#   cd backend && make migrate-up && cd ..
#   ./scripts/seed-order-demo.sh
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
PASSWORD="password123"
suffix=$(date +%s)

json() { python3 -c 'import json,sys; print(json.load(sys.stdin)["'"$1"'"])'; }

signup() {
	curl -sS -X POST "$BASE_URL/auth/signup" \
		-H 'Content-Type: application/json' \
		-d "{\"username\":\"$1\",\"email\":\"$1@example.test\",\"password\":\"$PASSWORD\"}"
}

seller="seller$suffix"
buyer="buyer$suffix"

seller_token=$(signup "$seller" | json access_token)
signup "$buyer" >/dev/null

listing=$(curl -sS -X POST "$BASE_URL/listings" \
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
