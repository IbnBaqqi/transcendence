// Hard coded profile & settings page #24
//
// Scope reminder (also on the issue): display name + bio are buildable now.
// Avatar upload needs #14, password change needs auth (#32/#33), payout
// account needs the payments module - render those as visible but disabled
// controls so the page reads as finished without pretending to work.
//
// Don't add a form library here: #47 hasn't picked one yet (React Hook Form +
// Zod is the default suggestion), and whoever does #47 will retrofit this.
//
// N.B. This page is only viewable for the logged in user of the same profile
// To view another user's profile we have User.tsx
import Avatar from "../components/objects/Avatar.tsx";
import Button from "../components/objects/Button.tsx";
import Toggle from "../components/objects/Toggle.tsx";
import { ContactDetailsSection } from "../components/forms/ContactDetails.tsx";
import { ChangePasswordSection } from "../components/forms/ChangePassword.tsx";
import { BioSection } from "../components/forms/Bio.tsx";
import { useState } from "react";

export default function Profile() {
  const [marketing, setMarketing] = useState(false);
  const [hideDetails, setHideDetails] = useState(false);
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">Profile & Settings</h1>
      {/* TODO: blocked by #109 nothing to load - hardcoded values for now. */}
      <div className="flex flex-row gap-4">
        <div>
          <Avatar size="lg" initials="OR" editable />
        </div>
        <div className="text-accent flex flex-col text-base">
          <div className="font-bold">Oscar Rogers</div>
          <div className="font-normal">oscarrogers@example.com</div>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Contact Details</h2>
        <ContactDetailsSection />
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Password</h2>
        <ChangePasswordSection />
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Bio</h2>
        <BioSection />
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Preferences</h2>
        <div className="flex flex-row gap-6">
          <Toggle
            checked={marketing}
            onChange={setMarketing}
            label="Receive marketing emails from us"
          />
          <Toggle
            checked={hideDetails}
            onChange={setHideDetails}
            label="Show only first name and initials to other users"
          />
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Account Deletion</h2>
        <div className="flex flex-row gap-2">
          <Button variant="secondary" onClick={() => console.log("delete!")}>
            Delete Account
          </Button>
        </div>
      </div>
    </div>
  );
}
