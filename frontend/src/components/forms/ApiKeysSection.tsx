import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useApiKeys, useCreateApiKey, useRevokeApiKey } from "../../api/apiKeys";
import { isApiError } from "../../api/client";
import { ApiKeyCreatedDialog } from "../modal/ApiKeyCreatedDialog";
import Button from "../objects/Button";
import { Skeleton } from "../objects/Skeleton";
import type { ApiKey, CreatedApiKey } from "../../api/types";

// The backend's own limit (api_key.go), mirrored so the field stops the user
// rather than the server rejecting a full round trip.
const MAX_NAME = 60;

export function ApiKeysSection() {
  const { t } = useTranslation();
  const { data: apiKeys, isPending, isError, refetch } = useApiKeys();
  const create = useCreateApiKey();

  const [name, setName] = useState("");
  const [created, setCreated] = useState<CreatedApiKey | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleCreate(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    try {
      setCreated(await create.mutateAsync(name.trim()));
      setName("");
    } catch (err) {
      setError(isApiError(err) ? err.message : t("apiKeys.createFailed"));
    }
  }

  const valid = name.trim().length > 0 && name.trim().length <= MAX_NAME;

  return (
    <div className="space-y-4">
      <p className="text-muted text-sm">{t("apiKeys.intro")}</p>

      <form onSubmit={(e) => void handleCreate(e)} className="flex items-end gap-2">
        {/* min-w-0: a flex item defaults to min-width:auto, and an input's
            content width is ~20 characters, so without this the field holds a
            ~206px floor and the button is the one that gets squeezed - down to
            the width of "Create", with "key" wrapping under it. */}
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <label htmlFor="api-key-name" className="text-muted text-sm">
            {t("apiKeys.nameLabel")}
          </label>
          <input
            id="api-key-name"
            value={name}
            maxLength={MAX_NAME}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("apiKeys.namePlaceholder")}
            className="border-line bg-surface text-foreground rounded-md border px-3 py-2"
          />
        </div>
        {/* Not a className on Button: its label is what must not wrap, and
            shrink-0 on every button in the app would stop the admin action
            rows narrowing, which is what lets them wrap (b4d0cee). */}
        <div className="shrink-0">
          <Button type="submit" variant="primary" disabled={!valid || create.isPending}>
            {create.isPending ? t("common.saving") : t("apiKeys.create")}
          </Button>
        </div>
      </form>

      {error && (
        <p role="alert" className="text-danger text-sm">
          {error}
        </p>
      )}

      {isPending && <Skeleton className="h-20 w-full" />}

      {isError && (
        <Skeleton
          variant="error"
          className="h-20 w-full"
          message={t("apiKeys.listError")}
          onRetry={() => refetch()}
        />
      )}

      {apiKeys?.length === 0 && <p className="text-muted text-sm">{t("apiKeys.empty")}</p>}

      {apiKeys && apiKeys.length > 0 && (
        <ul className="divide-line border-line divide-y rounded-lg border">
          {apiKeys.map((apiKey) => (
            <li key={apiKey.id}>
              <ApiKeyRow apiKey={apiKey} />
            </li>
          ))}
        </ul>
      )}

      {created && (
        <ApiKeyCreatedDialog
          apiKey={created}
          onClose={() => {
            setCreated(null);
            // mutateAsync's result also lives on create.data until reset() -
            // clearing only our own copy would leave the key in the mutation
            // cache for as long as this page stays mounted.
            create.reset();
          }}
        />
      )}
    </div>
  );
}

function ApiKeyRow({ apiKey }: { apiKey: ApiKey }) {
  const { t } = useTranslation();
  const revoke = useRevokeApiKey();
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const revoked = apiKey.revoked_at !== null;

  async function handleRevoke() {
    setError(null);
    try {
      await revoke.mutateAsync(apiKey.id);
      // Only on success: a failed row stays in its confirming state so the
      // error sits next to the button that produced it, ready to retry.
      setConfirming(false);
    } catch (err) {
      // RevokeKey matches only revoked_at IS NULL, so revoking twice - two tabs,
      // or a stale list - answers 404 for a row that is plainly on screen.
      if (isApiError(err) && err.status === 404) {
        setError(t("apiKeys.alreadyRevoked"));
      } else {
        setError(isApiError(err) ? err.message : t("apiKeys.revokeFailed"));
      }
    }
  }

  return (
    <div className="flex items-start justify-between gap-3 p-3">
      <div className="min-w-0">
        <p className={`font-medium ${revoked ? "text-muted line-through" : "text-foreground"}`}>
          {apiKey.name}
        </p>
        <p className="text-muted font-mono text-sm">{apiKey.key_prefix}…</p>
        <p className="text-muted text-sm">
          {t("apiKeys.fields.created", { date: new Date(apiKey.created_at).toLocaleDateString() })}
          {" · "}
          {/* Written at most once a minute, so this is "have they used it
              lately", not a request log. */}
          {apiKey.last_used_at
            ? t("apiKeys.fields.lastSeen", {
                date: new Date(apiKey.last_used_at).toLocaleDateString(),
              })
            : t("apiKeys.fields.neverUsed")}
        </p>
        {error && (
          <p role="alert" className="text-danger mt-1 text-sm">
            {error}
          </p>
        )}
      </div>

      {/* Revoked rows stay listed so a user can see what they switched off. */}
      {revoked ? (
        <span className="text-muted shrink-0 text-sm">{t("apiKeys.revoked")}</span>
      ) : confirming ? (
        <div className="flex shrink-0 gap-2">
          <Button variant="primary" disabled={revoke.isPending} onClick={() => void handleRevoke()}>
            {revoke.isPending ? t("apiKeys.revoking") : t("apiKeys.confirmRevoke")}
          </Button>
          <Button variant="secondary" onClick={() => setConfirming(false)}>
            {t("common.cancel")}
          </Button>
        </div>
      ) : (
        <Button variant="secondary" onClick={() => setConfirming(true)}>
          {t("apiKeys.revoke")}
        </Button>
      )}
    </div>
  );
}
