// Profile & settings page #24, now reading from GET /me/profile.
//
// Username and email are identity - the API offers no way to change them, so
// they render read-only. Contact details and bio are editable in their
// sections below (PATCH /me/profile). Password change is handled via
// POST /me/password (requires the current password). The preference toggles
// have no matching settings fields, so they stay commented out below until
// then.
//
// This page is only viewable for the logged-in user of the same profile.
// To view another user's profile we have User.tsx
import Avatar from "../components/objects/Avatar.tsx";
import Button from "../components/objects/Button.tsx";
import { ContactDetailsSection } from "../components/forms/ContactDetailsSection.tsx";
import { ChangePasswordSection } from "../components/forms/ChangePasswordSection.tsx";
import { BioSection } from "../components/forms/BioSection.tsx";
import { ApiKeysSection } from "../components/forms/ApiKeysSection.tsx";
import { BlockedUsersSection } from "../components/forms/BlockedUsersSection";
import { useEffect, useMemo, useState } from "react";
import { useModal } from "../providers/modalContext";
import { useAuth } from "../hooks/useAuth";
import { useDeleteAvatar, useOwnProfile, useUploadAvatar } from "../api/profile";
import { isApiError } from "../api/client";
import { deriveInitials } from "../lib/initials";
import { useTranslation } from "react-i18next";

export default function Profile() {
  const { t } = useTranslation();
  const { openModal } = useModal();
  const { user, isLoading: authLoading } = useAuth();

  const { data: profile, isLoading, error } = useOwnProfile();

  const signedOut = isApiError(error) && error.status === 401;

  const uploadAvatar = useUploadAvatar(profile?.id);
  const deleteAvatar = useDeleteAvatar(profile?.id);
  const [avatarError, setAvatarError] = useState<string | null>(null);

  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const avatarPreviewUrl = useMemo(
    () => (avatarFile ? URL.createObjectURL(avatarFile) : undefined),
    [avatarFile],
  );
  useEffect(() => {
    return () => {
      if (avatarPreviewUrl) URL.revokeObjectURL(avatarPreviewUrl);
    };
  }, [avatarPreviewUrl]);

  async function handleAvatarSelected(file: File) {
    setAvatarError(null);
    setAvatarFile(file);
    try {
      await uploadAvatar.mutateAsync(file);
    } catch (err) {
      setAvatarError(isApiError(err) ? err.message : t("common.somethingWentWrong"));
    } finally {
      setAvatarFile(null);
    }
  }

  async function handleAvatarRemove() {
    setAvatarError(null);
    try {
      await deleteAvatar.mutateAsync();
    } catch (err) {
      setAvatarError(isApiError(err) ? err.message : t("common.somethingWentWrong"));
    }
  }

  return (
    <div className="max-w-column mx-auto space-y-5 px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.profile.title")}</h1>
      {signedOut ? (
        <div className="space-y-3">
          <p className="text-muted text-sm">{t("pages.profile.signedOut")}</p>
          <Button variant="primary" onClick={() => openModal("login")}>
            {t("common.logIn")}
          </Button>
        </div>
      ) : error ? (
        <p className="text-berry-500 text-sm">
          {isApiError(error) ? error.message : t("common.somethingWentWrong")}
        </p>
      ) : authLoading || isLoading || !profile ? (
        // authLoading too: has_password decides whether the password section
        // renders, and AuthProvider reports user as null until the session is
        // restored - so without this a password account is told it has none.
        <p className="text-muted text-sm">{t("common.loading")}</p>
      ) : (
        <>
          <div className="flex flex-row gap-4">
            <div>
              <Avatar
                size="lg"
                // Username initial only, same rule as the header mini avatar.
                initials={deriveInitials(profile.username)}
                editable
                imageUrl={avatarPreviewUrl ?? profile.avatar_url ?? undefined}
                onImageSelected={(file) => void handleAvatarSelected(file)}
              />
            </div>
            <div className="text-accent my-auto flex flex-col gap-1 text-base">
              <div className="font-bold">{profile.username}</div>
              <div className="font-normal">{profile.email}</div>
              {uploadAvatar.isPending ? (
                <p className="text-muted text-sm">{t("avatar.uploading")}</p>
              ) : (
                profile.avatar_url && (
                  <button
                    type="button"
                    disabled={deleteAvatar.isPending}
                    onClick={() => void handleAvatarRemove()}
                    className="text-muted hover:text-foreground w-fit text-sm underline"
                  >
                    {deleteAvatar.isPending ? t("avatar.removing") : t("avatar.remove")}
                  </button>
                )
              )}
            </div>
          </div>
          {avatarError && (
            <p role="alert" className="text-berry-500 text-sm">
              {avatarError}
            </p>
          )}
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("pages.profile.contactDetails")}
            </h2>
            <ContactDetailsSection />
          </div>
          {/* Password section only makes sense for accounts that sign in with
            a password - a provider-only (OAuth) account has nothing to change. */}
          {user?.has_password && (
            <div className="space-y-1">
              <h2 className="text-foreground text-section font-bold">
                {t("pages.profile.password")}
              </h2>
              <ChangePasswordSection />
            </div>
          )}
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">{t("pages.profile.bio")}</h2>
            <BioSection />
          </div>
          {/* Not gated on has_password: an OAuth account needs keys too. */}
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">{t("pages.profile.apiKeys")}</h2>
            <ApiKeysSection />
          </div>
          {/* NOTE: No backend for these kind of preferences yet. Discussions were held about
              having a setting for showing personal details, but maybe now handled by the friend
              system, in which case following toggles could be removed/changed */}

          {/* <div className="space-y-1"> */}
          {/*   <h2 className="text-foreground text-lg font-bold">Preferences</h2> */}
          {/*   <div className="flex flex-row gap-6"> */}
          {/*     <Toggle */}
          {/*       checked={marketing} */}
          {/*       onChange={setMarketing} */}
          {/*       label="Receive marketing emails from us" */}
          {/*     /> */}
          {/*     <Toggle */}
          {/*       checked={hideDetails} */}
          {/*       onChange={setHideDetails} */}
          {/*       label="Show only first name and initials to other users" */}
          {/*     /> */}
          {/*   </div> */}
          {/* </div> */}
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">{t("blocks.listTitle")}</h2>
            <BlockedUsersSection />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-section font-bold">
              {t("pages.profile.accountManagement")}
            </h2>
            <div className="flex flex-row gap-4">
              {/* NOTE: Delete Account currently waiting for backend functionality to wire into */}
              <Button variant="secondary" onClick={() => openModal("deleteAccount")}>
                {t("common.deleteAccount")}
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
