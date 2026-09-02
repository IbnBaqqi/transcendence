.PHONY: setup

# One command to make a fresh checkout runnable, in the order that matters.
#
# frontend/node_modules is created here rather than left to Docker. A container
# cannot mount onto a path that does not exist, so the daemon creates any
# missing subdirectory of a bind mount - as root, inside your working tree. The
# result is an empty root-owned frontend/node_modules and an EACCES from every
# later host-side `npm ci`, with an error that blames permissions and never
# mentions Docker. Creating it first means Docker mounts over a directory that
# already exists and leaves ownership alone.
#
# Running the container as the host user does NOT fix this: the daemon prepares
# the mount point before the container starts, so its user is irrelevant.
# Measured both ways - see the PR for #184.
setup:
	@mkdir -p frontend/node_modules
	@test -f .env || cp .env.example .env
	@echo "✓ ready: run 'docker compose up'"
