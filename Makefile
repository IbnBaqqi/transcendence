.PHONY: setup

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
setup:
	@mkdir -p frontend/node_modules backend/uploads
	@test -f .env || cp .env.example .env
	@echo "✓ ready: run 'docker compose up'"
