// Stub for #24.
//
// Scope reminder (also on the issue): display name + bio are buildable now.
// Avatar upload needs #14, password change needs auth (#32/#33), payout
// account needs the payments module - render those as visible but disabled
// controls so the page reads as finished without pretending to work.
//
// Don't add a form library here: #47 hasn't picked one yet (React Hook Form +
// Zod is the default suggestion), and whoever does #47 will retrofit this.
//
import Avatar from "../components/Avatar.tsx"
import Button from "../components/Button.tsx"

export default function Profile() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8 space-y-5">
      <h1 className="text-foreground text-3xl font-bold">Profile & Settings</h1>
      {/* TODO(#24): display name + bio form. The backend doesn't create a
          profiles row on signup yet, so there's nothing to load - hardcode
          values for now. */}
      {/* Tailwind alternative to inline-block is flex */}
      <div className="flex flex-row gap-4">
        <div><Avatar /></div>
        <div className="flex flex-col text-accent text-1xl">
          <div className="font-bold">Oscar Roff</div>
          <div className="font-normal">oscarroff@gmail.com</div>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Contact Details</h2>
        <div className="flex flex-row gap-4">
          <div>First Name</div>
          <div>Last Name</div>
          <div>Telephone Number</div>
          <div>Location</div>
        </div>
        <div className="flex flex-row gap-2">
          <Button variant="primary" onClick={() => console.log("edit!")}>
            Edit Details
          </Button>
          <Button variant="secondary" onClick={() => console.log("cancel!")}>
            Cancel
          </Button>
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-1.5xl font-bold">Password</h2>
        <div className="">
          ************
        </div>
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
          Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Fusce auctor at fringilla aliquam massa iaculis et ad potenti cras purus. Curabitur himenaeos maximus viverra iaculis consectetur a enim. Magna in augue viverra primis aenean magna magna donec et quisque hendrerit etiam. Ullamcorper fames varius elementum sagittis elementum vitae eu inceptos quam imperdiet. A conubia aliquet libero molestie ultricies sagittis quam nostra cubilia elementum amet porta.
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
        {/*<svg><use href="/icons.svg#profile-icon" width="32" height="32"/></svg>*/}
      <p className="text-muted mt-2">
      </p>
    </div>
  );
}
