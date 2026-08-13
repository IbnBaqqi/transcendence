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
import Avatar from "../components/Avatar.tsx";
import Button from "../components/Button.tsx";
import Toggle from "../components/Toggle.tsx";
import { useState } from "react";

export default function Profile() {
  const [marketing, setMarketing] = useState(false);
  const [hideDetails, setHideDetails] = useState(false);
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">Profile & Settings</h1>
      {/* TODO(#): The backend doesn't create a profiles row on signup yet, so there's nothing to load - hardcoded values for now. */}
      <div className="flex flex-row gap-4">
        <div>
          <Avatar size="lg" initials="OR" editable />
        </div>
        <div className="text-accent text-1xl flex flex-col">
          <div className="font-bold">Oscar Roff</div>
          <div className="font-normal">oscarroff@gmail.com</div>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Contact Details</h2>
        <div className="flex flex-row gap-4">
          <div className="flex flex-col">
            <div className="text-muted">First Name</div>
            <div>Oscar</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Last Name</div>
            <div>Roff</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Telephone</div>
            <div>1234567890</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Location</div>
            <div>Open Maps integration?</div>
          </div>
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Edit Details
          </Button>
          {/* TODO(#): Using states, once forms are live we can make cancel only appear if user is in edit mode */}
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Password</h2>
        <div className="">************</div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Edit Password
          </Button>
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Bio</h2>
        <div className="">
          Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
          ut labore et dolore magna aliqua. Fusce auctor at fringilla aliquam massa iaculis et ad
          potenti cras purus. Curabitur himenaeos maximus viverra iaculis consectetur a enim. Magna
          in augue viverra primis aenean magna magna donec et quisque hendrerit etiam. Ullamcorper
          fames varius elementum sagittis elementum vitae eu inceptos quam imperdiet. A conubia
          aliquet libero molestie ultricies sagittis quam nostra cubilia elementum amet porta.
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Edit Bio
          </Button>
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Preferences</h2>
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
        <h2 className="text-foreground text-1.5xl font-bold">Account Deletion</h2>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Delete Account
          </Button>
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}
