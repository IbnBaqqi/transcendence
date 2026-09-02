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
import { useEffect, useMemo, useState } from "react";
import { useModal } from "../providers/modalContext";
import { useAuth } from "../hooks/useAuth";
import { useOwnProfile } from "../api/profile";
import { isApiError } from "../api/client";
import { deriveInitials } from "../lib/initials";
import { useTranslation } from "react-i18next";

export default function Profile() {
  const { t } = useTranslation();
  const { openModal } = useModal();
  const { logout } = useAuth();

  const { data: profile, isLoading, error } = useOwnProfile();

  const signedOut = isApiError(error) && error.status === 401;

  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const handleLogout = async () => {
    setIsLoggingOut(true);
    try {
      // Clears the session (and its cache) - the profile query then reruns,
      // gets a 401, and the signed-out branch above takes over.
      await logout();
    } finally {
      setIsLoggingOut(false);
    }
  };

  // The image the user picked in the "imageUpload" modal. There's no avatar
  // upload endpoint yet (#14), so this only lives in memory as a preview -
  // once the backend supports it, swap this for the real upload + persisted
  // URL.
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

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.profile.title")}</h1>
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
      ) : isLoading || !profile ? (
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
                imageUrl={avatarPreviewUrl}
                onImageSelected={setAvatarFile}
              />
            </div>
            <div className="text-accent my-auto flex flex-col text-base">
              <div className="font-bold">{profile.username}</div>
              <div className="font-normal">{profile.email}</div>
            </div>
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">
              {t("pages.profile.contactDetails")}
            </h2>
            <ContactDetailsSection />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">{t("pages.profile.password")}</h2>
            <ChangePasswordSection />
          </div>
          <div className="space-y-1">
            <h2 className="text-foreground text-lg font-bold">{t("pages.profile.bio")}</h2>
            <BioSection />
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
            <h2 className="text-foreground text-lg font-bold">
              {t("pages.profile.accountManagement")}
            </h2>
            <div className="flex flex-row gap-4">
              <Button variant="primary" onClick={handleLogout} disabled={isLoggingOut}>
                {isLoggingOut ? t("common.loggingOut") : t("common.logOut")}
              </Button>
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
