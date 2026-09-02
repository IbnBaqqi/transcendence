.PHONY: setup certs

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
	@echo "✓ ready: run 'docker compose up'"

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
	@mkdir -p certs
	@test -f certs/localhost.pem || openssl req -x509 -newkey rsa:2048 -nodes \
		-keyout certs/localhost-key.pem -out certs/localhost.pem -days 365 \
		-subj "/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1" 2>/dev/null
	@# openssl writes the key 0600, and the frontend image runs nginx as uid 101,
	@# which then cannot read it. Readable rather than root-owned or a privileged
	@# container: this key is generated, gitignored, valid only for localhost and
	@# worth nothing to anyone. Do not copy this line for a real certificate.
	@chmod 0644 certs/localhost-key.pem
	@echo "✓ certs/ ready (self-signed - browsers will warn, which is expected)"
