.PHONY: setup certs certs-force

# One command to make a fresh checkout runnable, in the order that matters.
#
# The two directories are created here rather than left to Docker. A container
# cannot mount onto a path that does not exist, so the daemon creates any
# missing subdirectory of a bind mount - as root, inside your working tree.
# Both are gitignored, so both are absent on a fresh clone and both get made
# this way: frontend/node_modules under ./frontend:/app, and backend/uploads
# under ./backend:/app. The result is an EACCES from the next host-side command
# that writes there - `npm ci` for one, `make run` saving an image for the other
# - with an error that blames permissions and never mentions Docker. Creating
# them first means Docker mounts over directories that already exist and leaves
# ownership alone.
#
# Linux only, in practice. Docker Desktop for macOS maps bind-mount ownership to
# the host user, so nothing lands root-owned there. mkdir -p costs nothing
# either way, so the instruction stays the same on both.
#
# Running the container as the host user does NOT fix this: the daemon prepares
# the mount point before the container starts, so its user is irrelevant.
# Measured both ways - see the PR for #184.
setup: certs
	@mkdir -p frontend/node_modules backend/uploads
	@test -f .env || cp .env.example .env
	@test -f backend/.env || cp .env.example backend/.env
	@echo "✓ ready: run 'docker compose up'"

# A literal comma cannot appear inside $(if ...) - commas separate a function's
# arguments - so it goes in a variable first.
comma := ,

# Extra subjectAltName entries, in openssl's own syntax and comma-separated:
#   make certs-force CERT_EXTRA_SAN=IP:10.18.185.59
#   make certs-force CERT_EXTRA_SAN='IP:10.18.185.59,DNS:metsatori.local'
# Needed to reach this stack from another machine. A browser matches the host
# you typed against the SAN, and localhost/127.0.0.1 do not cover a LAN address.
CERT_EXTRA_SAN ?=
CERT_SAN := DNS:localhost,IP:127.0.0.1$(if $(CERT_EXTRA_SAN),$(comma)$(CERT_EXTRA_SAN))

# A self-signed certificate for the production-shaped stack, which terminates
# TLS. Generated rather than committed: a private key in a repository is a
# leaked key even when it is a throwaway one.
#
# subjectAltName is not optional decoration - browsers ignore CN entirely and
# reject a certificate without a SAN that matches the host.
#
# Browsers will still warn: nothing trusts this issuer. That is expected for a
# local stack. `mkcert -install && mkcert localhost` produces a trusted one for
# anyone who has it, written to the same two paths.
certs:
	@command -v openssl >/dev/null || { \
		echo "openssl not found - it generates certs/ (apt install openssl, brew install openssl)"; \
		exit 1; }
	@mkdir -p certs
	@if [ ! -f certs/localhost.pem ] || [ ! -f certs/localhost-key.pem ]; then \
		out=$$(openssl req -x509 -newkey rsa:2048 -nodes \
			-keyout certs/localhost-key.pem -out certs/localhost.pem -days 365 \
			-subj "/CN=localhost" \
			-addext "subjectAltName=$(CERT_SAN)" 2>&1) \
			|| { echo "$$out" >&2; exit 1; }; \
	elif [ -n "$(CERT_EXTRA_SAN)" ]; then \
		echo "! certs/ already exists, so CERT_EXTRA_SAN did nothing."; \
		echo "  make certs-force CERT_EXTRA_SAN='$(CERT_EXTRA_SAN)'  reissues it."; \
	fi
	@# openssl writes the key 0600, and the frontend image runs nginx as uid 101,
	@# which then cannot read it. Readable rather than root-owned or a privileged
	@# container: this key is generated, gitignored, valid only for a local stack
	@# and worth nothing to anyone. Do not copy this line for a real certificate.
	@chmod 0644 certs/localhost-key.pem
	@echo "✓ certs/ ready (self-signed - browsers will warn, which is expected)"
	@# Read back from the file rather than echoing $(CERT_SAN): when the variable
	@# was ignored above, the two differ, and only the file tells the truth.
	@openssl x509 -in certs/localhost.pem -noout -ext subjectAltName \
		| tail -1 | sed 's/^ */  covers: /'

# Reissuing rather than editing: the SAN list is signed, so a name cannot be
# added to a certificate that already exists. Deletes and delegates, so the
# openssl command lives in exactly one place.
certs-force:
	@rm -f certs/localhost.pem certs/localhost-key.pem
	@$(MAKE) --no-print-directory certs CERT_EXTRA_SAN='$(CERT_EXTRA_SAN)'
